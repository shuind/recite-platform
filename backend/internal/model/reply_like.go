// file: internal/model/reply_like.go

package model

import "time"

// ReplyLike 代表用户对一个评论的点赞关系
type ReplyLike struct {
	UserID    uint `gorm:"primaryKey"`
	ReplyID   uint `gorm:"primaryKey"`
	CreatedAt time.Time
}
