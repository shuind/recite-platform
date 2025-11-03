package model

import "time"

// QuestionFollow 代表用户关注一个问题(Post)的关系
type QuestionFollow struct {
	UserID    uint `gorm:"primaryKey"`
	PostID    uint `gorm:"primaryKey"`
	CreatedAt time.Time
}
