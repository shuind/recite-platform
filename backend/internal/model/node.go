package model

import "gorm.io/gorm"

// Node 代表一个用户创建的节点，可以是文件夹或文本
type Node struct {
	// 显式地定义 gorm.Model 的字段，以便添加 json 标签
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt int64          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt int64          `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 使用 json:"-" 在API中隐藏此字段

	// 自定义字段，并添加 gorm 和 json 标签
	UserID   uint   `gorm:"not null;index" json:"user_id"`
	ParentID *uint  `gorm:"index" json:"parent_id"`                     // 使用指针以允许 NULL 值
	NodeType string `gorm:"type:varchar(10);not null" json:"node_type"` // 'folder' 或 'text'
	Title    string `gorm:"type:varchar(255);not null" json:"title"`
	Content  string `gorm:"type:text" json:"content"`

	// 关联关系仅用于 GORM，不需要 JSON 标签，它们不会被序列化
	Parent   *Node  `gorm:"foreignKey:ParentID;references:ID"`
	Children []Node `gorm:"foreignKey:ParentID;references:ID"`
}
