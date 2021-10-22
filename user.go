package dc

import "fmt"

const (
	Member        UserFlag = 1 << iota // 가입한 사용자
	Fixed                              // 고닉
	Manager                            // 마이너 또는 미니 갤러리 주딱
	Moderator                          // 마이너 또는 미니 갤러리 파딱
	Hit                                // 힛갤 선정자
	Administrator                      // 전역 관리자
)

type User struct {
	Username string
	Nickname string
	Password string
	Flags    UserFlag
}

type UserFlag int

func (f *UserFlag) Set(flag UserFlag)      { *f |= flag }
func (f *UserFlag) Clear(flag UserFlag)    { *f &^= flag }
func (f *UserFlag) Toggle(flag UserFlag)   { *f ^= flag }
func (f *UserFlag) Has(flag UserFlag) bool { return *f&flag != 0 }

func (user User) String() string {
	return fmt.Sprintf("%s(%s)", user.Nickname, user.Username)
}
