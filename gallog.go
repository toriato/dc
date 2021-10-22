package dc

import (
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

type Gallog struct {
	session *Session

	User *User
}

func (session *Session) NewGallog(username string) *Gallog {
	return &Gallog{
		session: session,
		User: &User{
			Username: username,
			Flags:    Member,
		},
	}
}

func (gallog Gallog) Write(entry GuestbookEntry) error {
	payload := url.Values{}
	payload.Set("memo", entry.Content)

	// 사용자가 익명일 경우 닉네임과 비밀번호 설정하기
	if !entry.User.Flags.Has(Member) {
		payload.Set("name", entry.User.Nickname)
		payload.Set("password", entry.User.Password)
	}

	if entry.Secret {
		payload.Set("is_secret", "1")
	}

	_, err := gallog.session.Client.R().
		SetHeader("X-Requested-With", "X-Requested-With").
		SetFormDataFromValues(payload).
		Post(fmt.Sprintf("https://gallog.dcinside.com/%s/ajax/guestbook_ajax/write", gallog.User.Username))
	if err != nil {
		return errors.WithMessage(err, "갤로그 방명록 작성 페이지 요청 중 오류가 발생했습니다")
	}

	return nil
}

func (gallog *Gallog) Guestbook(year, page int64) ([]GuestbookEntry, error) {
	entries := []GuestbookEntry{}

	res, err := gallog.session.Client.R().
		SetQueryParam("y", strconv.FormatInt(year, 10)).
		SetQueryParam("p", strconv.FormatInt(page, 10)).
		Get(fmt.Sprintf("https://gallog.dcinside.com/%s/guestbook", gallog.User.Username))
	if err != nil {
		return nil, errors.WithMessage(err, "갤로그 방명록 목록 페이지 요청 중 오류가 발생했습니다")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if err != nil {
		return nil, errors.WithMessage(err, "갤로그 방명록 목록 페이지 파싱 중 오류가 발생했습니다")
	}

	doc.Find("#gb_comments > li").Each(func(_ int, s *goquery.Selection) {
		entry := GuestbookEntry{
			gallog:  gallog,
			User:    &User{},
			Content: s.Find(".memo").Text(),
		}

		// 방명록 아이디와 헤드 값
		id, _ := strconv.ParseInt(s.AttrOr("data-no", ""), 10, 64)
		head, _ := strconv.ParseInt(s.AttrOr("data-headnum", ""), 10, 64)
		entry.ID = id
		entry.Head = head

		if s.Find(".icon_secretsquare").AttrOr("style", "") == "display:" {
			entry.Secret = true
		}

		// 작성 시각
		date, _ := time.Parse("2006.01.02 03:04:05", s.Find(".date").Text())
		entry.CreatedAt = date

		// 작성자 정보
		writerRef := s.Find(".writer_info")

		if ipRef := writerRef.Find(".ip"); ipRef.Length() > 0 {
			ip := ipRef.Text()
			entry.User.Username = ip[1 : len(ip)-1]
		} else {
			entry.User.Flags.Set(Member)

			iconRef := writerRef.Find(".writer_nikcon img")

			{
				parts := strings.SplitN(iconRef.AttrOr("onclick", ""), "'", 2)
				parts = strings.SplitN(parts[1], "'", 2)
				entry.User.Username = parts[0][1:]
			}

			// 고닉 아이콘이 존재하는지 확인하기
			if strings.Contains(iconRef.AttrOr("src", ""), "fix") {
				entry.User.Flags.Set(Fixed)
			}
		}

		entry.User.Nickname = writerRef.Find(".nickname").Text()

		entries = append(entries, entry)
	})

	return entries, nil
}
