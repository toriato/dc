package dc_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/toriato/dc"
)

func TestGallogGuestbook(t *testing.T) {
	session := dc.NewSession()

	if err := session.Login(&testdata.Gallog.Guestbook.Credentials); err != nil {
		assert.Fail(t, "", err)
		return
	}

	gallog := session.NewGallog(session.User.Username)
	entries, err := gallog.Guestbook(int64(time.Now().Year()), 1)
	assert.NoError(t, err)

	for _, entry := range entries {
		t.Logf(`%s said "%s"`, entry.User, entry.Content)
	}
}

func TestGallogGuestbookDelete(t *testing.T) {
	session := dc.NewSession()

	if err := session.Login(&testdata.Gallog.Guestbook.Credentials); err != nil {
		assert.Fail(t, "", err)
		return
	}

	gallog := session.NewGallog(session.User.Username)
	entries, err := gallog.Guestbook(int64(time.Now().Year()), 1)
	assert.NoError(t, err)

	for _, entry := range entries {
		assert.NoError(t, entry.Delete())
	}
}
