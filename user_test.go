package dc_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/toriato/dc"
)

var mappedUserFlag = map[string]dc.UserFlag{
	"Member":        dc.Member,        // 유동
	"Fixed":         dc.Fixed,         // 고닉
	"Manager":       dc.Manager,       // 마이너 또는 미니 갤러리 주딱
	"Moderator":     dc.Moderator,     // 마이너 또는 미니 갤러리 파딱
	"Hit":           dc.Hit,           // 힛갤 선정자
	"Administrator": dc.Administrator, // 갤로그 없는 관리자
}

func TestUserFlag(t *testing.T) {
	session := dc.NewSession()

	for _, c := range testdata.User.Credentials {
		if err := session.Login(&c.Credentials); err != nil {
			assert.Fail(t, "", err)
			continue
		}

		for _, flag := range c.Flags {
			expected := true

			if strings.HasPrefix("!", flag) {
				expected = false
				flag = flag[1:]
			}

			assert.Equal(t, expected, session.User.Flags.Has(mappedUserFlag[flag]), session.User)
		}
	}
}
