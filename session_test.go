package dc_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toriato/dc"
)

func TestSessionLogin(t *testing.T) {
	session := dc.NewSession()

	assert.ErrorIs(t, session.Login(&testdata.Session.Login.Invalid), dc.ErrInvalidCredentials)

	assert.ErrorIs(t, session.Login(&testdata.Session.Login.InvalidTOTPKey), dc.ErrInvalidTOTP)

	c := testdata.Session.Login.Valid
	assert.NoError(t, session.Login(&c.Credentials))
	assert.Equal(t, c.Nickname, session.User.Nickname)
	t.Logf("Logged in %s with valid credentials (%s)", session.User, session.Get())

	c = testdata.Session.Login.ValidTOTP
	assert.NoError(t, session.Login(&c.Credentials))
	assert.Equal(t, c.Nickname, session.User.Nickname)
	t.Logf("Logged in %s with valid TOTP (%s)", session.User, session.Get())
}

func TestSessionUpdate(t *testing.T) {
	session := dc.NewSession()

	// 로그인 안한 상태에서 업데이트하면 인증 오류를 반환해야함
	assert.ErrorIs(t, session.Update(), dc.ErrUnauthorized)

	// 정상적으로 로그인된 상태에선 오류를 반환해선 안됨
	assert.NoError(t, session.Login(&testdata.Session.Login.Valid.Credentials))
	assert.NoError(t, session.Update())
	t.Logf("Logged in %s", session.User)
}

func TestSessionGet(t *testing.T) {
	session := dc.NewSession()

	// 사이트와 통신한 적 없다면 세션 아이디는 비어있어야함
	assert.Equal(t, "", session.Get())

	if err := session.Login(&testdata.Session.Login.Valid.Credentials); err != nil {
		assert.Fail(t, "", err)
		return
	}

	// 사이트와 통신한 상태에서 세션 아이디는 비어있을 수 없음
	assert.NotEqual(t, "", session.Get())

	t.Logf("Session ID of %s is %s", session.User, session.Get())
}

func TestSessionSet(t *testing.T) {
	session := dc.NewSession()

	// 계정 정보가 nil 이 아니라면 기존 계정 정보를 대체해야함
	assert.ErrorIs(t, dc.ErrUnauthorized, session.Set(""))
	assert.NoError(t, session.Login(&testdata.Session.Login.Valid.Credentials))

	// 이후 정상 세션 아이디 테스트를 위해 세션 아이디 불러오기
	id := session.Get()
	assert.NotEqual(t, "", id)

	// 잘못된 세션 아이디를 전달하면 인증 실패 오류가 발생해야함
	assert.ErrorIs(t, dc.ErrUnauthorized, session.Set(""))

	// 정상 세션 아이디는 오류를 반환해선 안됨
	assert.NoError(t, session.Set(id))

	t.Logf("Session ID of %s is %s", session.User, session.Get())
}
