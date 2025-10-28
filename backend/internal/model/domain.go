package model

import (
	"time"

	"gorm.io/gorm"
)

type Domain struct {
	// 1. 显式定义 gorm.Model 的所有字段，并添加 json 标签
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间通常不在API中返回

	// 2. 你自己定义的字段，保持已有的 json 标签
	OwnerID     uint   `gorm:"not null;index" json:"owner_id"`
	Name        string `gorm:"type:varchar(100);not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	JoinCode    string `gorm:"type:varchar(8);not null;unique" json:"join_code"`
}

// (可选但推荐) 自定义表名
func (Domain) TableName() string {
	return "domains"
}
