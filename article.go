package dc

import "time"

type Article struct {
	ID               int64
	Gallery          *Gallery
	Author           *User
	Subject          string // 제목
	Content          string // 내용
	TextComments     int    // 댓글 수
	VoiceComments    int    // 보이스 리플 수
	Upvotes          int    // 추천 수 (종합)
	CertifiedUpvotes int    // 추천 수 (고닉)
	Downvotes        int    // 비추 수
	CreatedAt        time.Time
}
