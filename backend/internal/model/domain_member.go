package model

import (
	"time"
)

type DomainMember struct {
	// 我们不使用 gorm.Model，因为我们想自定义主键和字段
	ID       uint      `gorm:"primaryKey"`
	DomainID uint      `gorm:"not null;uniqueIndex:idx_domain_user"` // 复合唯一索引
	UserID   uint      `gorm:"not null;uniqueIndex:idx_domain_user"` // 复合唯一索引
	Role     string    `gorm:"type:varchar(20);not null;default:'member'"`
	JoinedAt time.Time `gorm:"default:now()"`

	// 定义关联关系
	Domain Domain `gorm:"foreignKey:DomainID"`
	User   User   `gorm:"foreignKey:UserID"`
}
