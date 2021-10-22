package dc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type GuestbookEntry struct {
	gallog *Gallog

	ID        int64
	Head      int64
	User      *User
	Content   string
	Secret    bool
	CreatedAt time.Time
}

func (entry GuestbookEntry) Delete() error {
	if entry.gallog == nil {
		return nil
	}

	_, err := entry.gallog.session.Client.R().
		SetHeader("X-Requested-With", "XMLHttpRequest").
		SetFormData(map[string]string{
			"headnum": strconv.FormatInt(entry.Head, 10),
		}).
		Post(fmt.Sprintf("https://gallog.dcinside.com/%s/ajax/guestbook_ajax/delete", entry.gallog.User.Username))
	if err != nil {
		return errors.WithMessage(err, "갤로그 방명록 삭제 페이지 요청 중 오류가 발생했습니다")
	}

	return nil
}
