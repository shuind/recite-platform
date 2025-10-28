package model

import (
	"time"

	"gorm.io/gorm"
)

type Recording struct {
	// 1. 将 gorm.Model 的字段显式地写出来，并添加 json 标签
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间通常不在API中返回

	// 2. 为你自己的所有字段也添加 json 标签
	UserID         uint   `gorm:"not null;index" json:"user_id"`
	TextID         *uint  `gorm:"index" json:"text_id"`
	NodeID         *uint  `gorm:"index" json:"node_id"`
	DomainNodeID   *uint  `gorm:"index" json:"domain_node_id"`
	Title          string `gorm:"type:varchar(255)" json:"title"`
	Status         string `gorm:"type:varchar(20);default:'processing'" json:"status"`
	AudioURL       string `gorm:"type:varchar(512)" json:"audio_url"`
	AiStatus       string `gorm:"type:varchar(20);default:'pending'" json:"ai_status"`
	RecognizedText string `gorm:"type:text" json:"recognized_text"`

	// Preload("User") 会将查询到的 User 信息填充到这个字段
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	LikesCount       int  `gorm:"default:0" json:"likes_count"`
	CommentsCount    int  `gorm:"default:0" json:"comments_count"`
	IsDomainFeatured bool `gorm:"default:false" json:"is_domain_featured"`

	// Preload("DomainNode") 会将查询到的 DomainNode 信息填充到这个字段
	DomainNode DomainNode `gorm:"foreignKey:DomainNodeID" json:"domain_node,omitempty"`
	// 3. 关联关系在常规列表中通常不返回，以避免循环引用和过大的响应体
	//    可以使用 json:"-" 来在序列化时忽略它们。
	//    如果需要返回，可以考虑创建专门的 DTO (Data Transfer Object)。
	Text Text `gorm:"foreignKey:TextID" json:"-"`
	Node Node `gorm:"foreignKey:NodeID" json:"-"`
}

// 4. (可选但推荐) 自定义表名，GORM 默认会转为复数 "recordings"
func (Recording) TableName() string {
	return "recordings"
}
