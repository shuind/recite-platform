// internal/model/post_like.go
package model

import "time"

// PostLike 模型代表一个用户对一个帖子的点赞关系
type PostLike struct {
	UserID    uint      `gorm:"primaryKey"`
	PostID    uint      `gorm:"primaryKey"`
	CreatedAt time.Time // GORM 会自动处理这个字段
}