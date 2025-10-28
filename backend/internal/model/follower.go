package model

import "time"

// Follower 模型定义了用户之间的关注关系。
// 它对应数据库中的 "followers" 表。
type Follower struct {
	// --- 核心字段 ---

	// FollowerID 是发起关注的人（粉丝）的用户ID。
	// `gorm:"primaryKey"`: 告诉 GORM，这个字段是主键的一部分。
	// `gorm:"column:follower_id"`: 明确指定它映射到数据库的 follower_id 列。
	// (如果字段名和列名完全匹配，如 FollowerID -> follower_id，此标签可省略，但写上更清晰)
	FollowerID uint `gorm:"primaryKey;column:follower_id"`

	// FollowingID 是被关注的人的用户ID。
	// `gorm:"primaryKey"`: 它也是主键的一部分。与上面的 FollowerID 共同组成联合主键。
	FollowingID uint `gorm:"primaryKey;column:following_id"`

	// --- 时间戳 ---

	// FollowedAt 记录了关注关系创建的时间。
	// GORM 在创建记录时会自动填充当前时间，但我们最好在SQL中用DEFAULT NOW()定义。
	FollowedAt time.Time `gorm:"column:followed_at"`

	// --- GORM 关联关系 (Belongs To) ---
	// 这部分是可选的，但对于进行 JOIN 查询和预加载非常有用。

	// Follower 字段代表“关注者”这个用户实体。
	// `gorm:"foreignKey:FollowerID"`: 明确告诉 GORM，这个 User 结构体是通过
	// 本结构体 (Follower) 的 FollowerID 字段关联的。
	Follower User `gorm:"foreignKey:FollowerID"`

	// Following 字段代表“被关注者”这个用户实体。
	// `gorm:"foreignKey:FollowingID"`: 同理，它通过 FollowingID 关联。
	Following User `gorm:"foreignKey:FollowingID"`
}
