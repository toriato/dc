package dc

import (
	"bytes"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

type Gallery struct {
	session *Session

	ID   string
	Name string
	Type GalleryType
}

type GalleryType int

const (
	Major GalleryType = iota
	Minor
	Mini
)

var (
	galleryEndpoints = map[GalleryType]string{
		Major: "https://gall.dcinside.com/board",
		Minor: "https://gall.dcinside.com/mgallery/board",
		Mini:  "https://gall.dcinside.com/mini/board",
	}
)

func (session *Session) NewGallery(id string, mini bool) (*Gallery, error) {
	gallery := &Gallery{
		session: session,
		ID:      id,
	}

	endpoint := "https://m.dcinside.com/board/" + id
	if mini {
		endpoint = "https://m.dcinside.com/mini/" + id
	}

	// 갤러리 종류 확인하기
	{
		res, err := session.Client.R().Get(endpoint)
		if err != nil {
			return nil, errors.WithMessage(err, "갤러리 게이트웨이 페이지를 요청하는 중 오류가 발생했습니다")
		}

		switch res.StatusCode() {
		case 302:
			endpoint = res.Header().Get("Location")

			u, _ := url.Parse(endpoint)
			p := u.Path

			switch {
			case strings.HasPrefix(p, "/board"):
				gallery.Type = Major
			case strings.HasPrefix(p, "/mgallery"):
				gallery.Type = Minor
			case strings.HasPrefix(p, "/mini"):
				gallery.Type = Mini
			default:
				return nil, ErrUnexpected
			}
		case 404:
			return nil, ErrNotFound
		default:
			return nil, ErrUnexpected
		}
	}

	// 갤러리 메인 페이지에서 정보 불러오기
	{
		res, err := session.Client.R().Get(endpoint)
		if err != nil {
			return nil, errors.WithMessage(err, "갤러리 게시글 목록 페이지 요청 중 오류가 발생했습니다")
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
		if err != nil {
			return nil, errors.WithMessage(err, "갤러리 게시글 목록 페이지 파싱 중 오류가 발생했습니다")
		}

		gallery.Name = doc.Find("meta[name=title]").AttrOr("content", "")
	}

	return gallery, nil
}

func (gallery *Gallery) Articles(page int) ([]Article, error) {
	res, err := gallery.session.Client.R().
		SetQueryParam("id", gallery.ID).
		Get(galleryEndpoints[gallery.Type] + "/list")
	if err != nil {
		return nil, errors.WithMessage(err, "갤러리 게시글 목록 페이지 요청 중 오류가 발생했습니다")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if err != nil {
		return nil, errors.WithMessage(err, "갤러리 게시글 목록 페이지 파싱 중 오류가 발생했습니다")
	}

	articles := []Article{}

	doc.Find(".gall_list .us-post").Each(func(_ int, s *goquery.Selection) {
		article := Article{}

		titleAnchorRef := s.Find(".gall_tit a")
		{
			u, _ := url.Parse(titleAnchorRef.First().AttrOr("href", ""))

			// 갤러리 구조 설정하기
			galleryID := u.Query().Get("id")

			if galleryID == gallery.ID {
				// 갤러리 아이디가 현재 갤러리 구조와 일치한다면 그대로 사용하기
				article.Gallery = gallery
			} else {
				// 아이디만 있는 새 구조 만들어 사용하기
				article.Gallery = &Gallery{ID: galleryID}
			}

			// 게시글 번호 파싱하기
			id, _ := strconv.ParseInt(u.Query().Get("no"), 10, 64)
			article.ID = id
		}

		// 작성자 정보
		writerRef := s.Find(".gall_writer")
		article.Author = &User{
			Username: writerRef.AttrOr("data-uid", "") + writerRef.AttrOr("data-ip", ""),
			Nickname: writerRef.AttrOr("data-nick", ""),
		}

		// 작성자 아이콘
		writerIconRef := writerRef.Find(".writer_nikcon img")
		if writerIconRef.Length() > 0 {
			src := writerIconRef.AttrOr("src", "")

			switch {
			case strings.Contains(src, "fix"):
				article.Author.Flags.Set(Fixed)
				fallthrough
			case strings.Contains(src, "sub_manager"):
				article.Author.Flags.Set(Moderator)
			case strings.Contains(src, "manager"):
				article.Author.Flags.Set(Manager)
			}

			article.Author.Flags.Set(Member)
		}

		articles = append(articles, article)
	})

	return articles, nil
}
