package dc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"github.com/pquerna/otp/totp"
)

type Session struct {
	Client      *resty.Client
	Cookies     *cookiejar.Jar
	Credentials *Credentials
	User        *User
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPKey  string `json:"totpKey"`
}

var (
	ErrUnauthorized       = errors.New("로그인이 필요합니다")
	ErrInvalidCredentials = errors.New("아이디 또는 비밀번호가 잘못됐습니다")
	ErrInvalidTOTP        = errors.New("TOTP 코드가 잘못됐습니다")

	patternJavascriptAlert = regexp.MustCompile(`alert\((.+)\)`)
)

func NewSession() *Session {
	cookies, _ := cookiejar.New(&cookiejar.Options{})

	client := resty.New()
	client.SetCookieJar(cookies)
	client.OnAfterResponse(func(_ *resty.Client, r *resty.Response) error {
		if len(r.Body()) < 1 {
			return ErrTemporaryIPBanned
		}

		// text/html 이 아닌 다른 파일을 반환한다면 검사하지 않기
		if !strings.HasPrefix(r.Header().Get("Content-Type"), "text/html") {
			return nil
		}

		body := r.String()

		// JSON 결과 파싱하기
		if body[0] == '{' && body[len(body)-1] == '}' {
			var result struct {
				Status  string `json:"result"`
				Message string `json:"msg"`
			}

			if err := json.Unmarshal([]byte(body), &result); err != nil {
				return err
			}

			if result.Status == "fail" {
				return errors.WithMessage(ErrUnexpected, result.Message)
			}
		}

		// 비정상 요청 결과 파싱하기
		if body == "정상적인 접근이 아닙니다." {
			return ErrUnexpected
		}

		return nil
	})

	session := &Session{
		Client:  client,
		Cookies: cookies,
	}

	return session
}

// Login 메소드는 제공된 인증 정보를 통해 로그인합니다
func (session *Session) Login(credentials *Credentials) error {
	var redirectURL string
	var resultURL = "https://gall.dcinside.com"
	var client = session.Client

	if credentials != nil {
		session.Credentials = credentials
	}

	// SSO 쿠키 가져오기
	{
		_, err := client.R().
			SetQueryParam("s_url", resultURL).
			Get("https://dcid.dcinside.com/join/login.php")
		if err != nil {
			return errors.WithMessage(err, "SSO 토큰 쿠키 요청 중 오류가 발생했습니다")
		}
	}

	// 인증 정보 전송하기
	{
		form := url.Values{}
		form.Set("user_id", session.Credentials.Username)
		form.Set("pw", session.Credentials.Password)
		form.Set("s_url", resultURL)

		res, err := client.R().
			SetFormDataFromValues(form).
			SetHeader("Referer", "https://dcid.dcinside.com/join/login.php?s_url="). // 레퍼 주소에 s_url 인자 없으면 로그인 불가능
			Post("https://dcid.dcinside.com/join/member_check.php")
		if err != nil {
			return errors.WithMessage(err, "인증 정보 전송 요청 중 오류가 발생했습니다")
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
		if err != nil {
			return errors.WithMessage(err, "인증 정보 전송 후 받은 데이터 파싱 중 오류가 발생했습니다")
		}

		// 자바스크립트 alert 메소드를 통한 안내 메세지가 있는지 확인하기
		var alert string
		{
			script := doc.Find(`body > script:not(:empty)`).Text()
			matches := patternJavascriptAlert.FindStringSubmatch(script)

			if len(matches) > 0 {
				alert = matches[1]
				alert = alert[1 : len(alert)-2] // 문자열을 감싸고 있는 첫번째와 마지막 따옴표 제거하기
			}
		}

		switch alert {
		// alert 메세지가 없을 경우 리다이렉션 진행하기
		case "":
			redirectURL = doc.Find(`meta[http-equiv="refresh"]`).AttrOr("content", "")
			redirectURL = strings.Replace(redirectURL, "0; url=", "", 1)

		case "아이디 또는 비밀번호가 잘못되었습니다":
			return ErrInvalidCredentials

		default:
			return errors.WithMessagef(ErrUnexpected, "인증 정보 전송 후 서버가 예측하지 못한 메세지를 반환했습니다: %s", alert)
		}
	}

	// TOTP 인증이 필요하다면 전송하기
	if strings.HasPrefix(redirectURL, "./login_otp.php") {
		code, err := totp.GenerateCode(session.Credentials.TOTPKey, time.Now())
		if err != nil {
			return errors.WithMessage(err, "TOTP 코드 생성 중 오류가 발생했습니다")
		}

		form := url.Values{}
		form.Set("otp_code", code)

		// XHR 요청으로 TOTP 코드 인증 결과 가져오기
		{
			res, err := client.R().
				SetFormDataFromValues(form).
				SetHeader("Referer", "https://dcid.dcinside.com/join/login_otp.php?s_url="). // 레퍼 주소에 s_url 인자 없으면 로그인 불가능
				SetHeader("X-Requested-With", "XMLHttpRequest").
				Post("https://dcid.dcinside.com/join/member_check.php")
			if err != nil {
				return errors.WithMessage(err, "TOTP 코드 XHR 전송 요청 중 오류가 발생했습니다")
			}

			doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
			if err != nil {
				return errors.WithMessage(err, "TOTP 코드 XHR 전송 후 받은 데이터 파싱 중 오류가 발생했습니다")
			}

			raw := strings.TrimSpace(doc.Find("body").Text())
			result := struct {
				Status  string `json:"result"`
				Message string `json:"msg"`
			}{}

			if err := json.Unmarshal([]byte(raw), &result); err != nil {
				return errors.WithMessage(err, "TOTP 코드 XHR 전송 후 받은 데이터 파싱 중 오류가 발생했습니다")
			}

			if result.Status != "success" {
				switch result.Message {
				case "잘못된 코드입니다. 다시 확인하시고 시도해 주세요.":
					return ErrInvalidTOTP
				default:
					return errors.WithMessagef(ErrUnexpected, "TOTP 코드 XHR 전송 후 서버가 예측하지 못한 메세지를 반환했습니다: %s", result.Message)
				}
			}
		}

		// XHR 없는 요청으로 리다이렉션 주소 가져오기
		res, err := client.R().
			SetFormDataFromValues(form).
			SetHeader("Referer", "https://dcid.dcinside.com/join/login_otp.php?s_url="). // 레퍼 주소에 s_url 인자 없으면 로그인 불가능
			Post("https://dcid.dcinside.com/join/member_check.php")
		if err != nil {
			return errors.WithMessage(err, "TOTP 코드 전송 요청 중 오류가 발생했습니다")
		}

		doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
		if err != nil {
			return errors.WithMessage(err, "TOTP 코드 전송 후 받은 데이터 파싱 중 오류가 발생했습니다")
		}

		redirectURL = doc.Find(`meta[http-equiv="refresh"]`).AttrOr("content", "")
		redirectURL = strings.Replace(redirectURL, "0; url=", "", 1)
	}

	// 모든 작업을 마쳤는데 리다이렉션 주소가 없다면 정상 처리된게 아니므로 오류 반환하기
	if redirectURL == "" {
		return errors.WithMessage(ErrUnexpected, "인증 정보 전송 후 서버가 리다이렉션할 주소를 반환하지 않았습니다")
	}

	// // 비밀번호 변경 안내 페이지라면 결과 페이지로 리다이렉션하기
	// if strings.HasPrefix(redirectURL, "https://dcid.dcinside.com/join_new/pw_campaign.php") {
	// 	redirectURL = resultURL
	// }

	// // 리다이렉션하기
	// {
	// 	_, err := client.R().Get(redirectURL)
	// 	if err != nil {
	// 		return errors.WithMessagef(err, "리다이렉션 요청 중 오류가 발생했습니다: %s", redirectURL)
	// 	}
	// }

	// 사용자 정보 새로 불러오기
	return session.Update()
}

// Update 메소드는 현재 세션 구조에서 사용자 정보를 서버에서 새로 가져옵니다
func (session *Session) Update() error {
	res, err := session.Client.R().Get("https://gall.dcinside.com/")
	if err != nil {
		return errors.WithMessage(err, "갤러리 페이지 요청 중 오류가 발생했습니다")
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(res.Body()))
	if err != nil {
		return errors.WithMessage(err, "갤러리 페이지 파싱 중 오류가 발생했습니다")
	}

	// 사용자 정보 파싱하기
	session.User = &User{}
	session.User.Nickname = strings.TrimSpace(doc.Find(".user_info .nickname em").Text())

	// 페이지에 닉네임 요소가 존재하지 않는다면 로그인된 상태가 아닌 것으로 판단하기
	if session.User.Nickname == "" {
		return ErrUnauthorized
	}

	// 갤로그 주소로부터 사용자 아이디 파싱하기
	{
		parts := strings.SplitN(res.String(), "'//gallog.dcinside.com", 2)
		parts = strings.SplitN(parts[1], "'", 2)
		session.User.Username = parts[0][1:]
	}

	// 닉네임 옆에 붙는 아이콘의 주소 값을 통해 고닉인지 반고닉인지 확인하기
	icon := doc.Find(".writer_nikcon img").AttrOr("src", "")
	if strings.Contains(icon, "fix_nik.gif") {
		session.User.Flags.Set(Fixed)
	}

	return nil
}

// Get 메소드는 현재 사용 중인 세션 아이디를 반환합니다
func (session Session) Get() string {
	u, _ := url.Parse("https://dcinside.com")

	for _, c := range session.Cookies.Cookies(u) {
		if c.Name == "PHPSESSID" {
			return c.Value
		}
	}

	return ""
}

// Set 메소드는 사용할 세션 아이디를 반환합니다
func (session *Session) Set(id string) error {
	u, _ := url.Parse("https://dcinside.com")

	session.Cookies.SetCookies(u, []*http.Cookie{{
		Domain:   ".dcinside.com",
		Name:     "PHPSESSID",
		Value:    id,
		HttpOnly: true,
	}})

	return session.Update()
}
