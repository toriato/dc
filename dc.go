package dc

import "github.com/pkg/errors"

type H = map[string]string

var (
	ErrTemporaryIPBanned = errors.New("아이피가 일시적으로 차단됐습니다")
	ErrUnexpected        = errors.New("예측하지 못한 결과가 발생했습니다")
	ErrNotFound          = errors.New("찾을 수 없거나 존재하지 않습니다")
)
