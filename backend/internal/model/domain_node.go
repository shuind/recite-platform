package model

import (
	"time"

	"gorm.io/gorm"
)

type DomainNode struct {
	// 1. 明确定义主键，并用 gorm tag 指定它
	ID uint `gorm:"primarykey" json:"id"`

	// 2. 明确定义其他所有字段，并用 gorm tag 指定列名
	//    这能保证 Go 字段名和数据库列名精确对应，不受 GORM 默认命名策略的影响。
	DomainID uint   `gorm:"column:domain_id;not null" json:"domain_id"`
	ParentID *uint  `gorm:"column:parent_id" json:"parent_id"`
	NodeType string `gorm:"column:node_type;type:varchar(10);not null" json:"node_type"`
	Title    string `gorm:"column:title;type:varchar(255);not null" json:"title"`
	Content  string `gorm:"column:content;type:text" json:"content"`

	CommentsCount int `gorm:"not null;default:0" json:"comments_count"`

	// 3. 明确定义时间戳和软删除字段
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
