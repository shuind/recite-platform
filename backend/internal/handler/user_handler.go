// file: internal/handler/user_handler.go

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shuind/language-learner/backend/internal/model" // 确保路径正确
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserHandler struct {
	DB *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{DB: db}
}

// === FollowUser 关注一个用户 (使用您的代码) ===
func (h *UserHandler) FollowUser(c *gin.Context) {
	followerID_interface, _ := c.Get("userID")
	followerID := followerID_interface.(uint)

	followingID_str := c.Param("id")
	followingID_uint64, err := strconv.ParseUint(followingID_str, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	followingID := uint(followingID_uint64)

	if followerID == followingID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot follow yourself"})
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		follow := model.Follower{
			FollowerID:  followerID,
			FollowingID: followingID,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&follow).Error; err != nil {
			return err
		}

		if tx.RowsAffected > 0 {
			tx.Model(&model.User{}).Where("id = ?", followerID).UpdateColumn("following_count", gorm.Expr("following_count + 1"))
			tx.Model(&model.User{}).Where("id = ?", followingID).UpdateColumn("followers_count", gorm.Expr("followers_count + 1"))
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow user", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Followed successfully"})
}

// === UnfollowUser 取消关注一个用户 (使用您的代码) ===
func (h *UserHandler) UnfollowUser(c *gin.Context) {
	followerID_interface, _ := c.Get("userID")
	followerID := followerID_interface.(uint)
	followingID_str := c.Param("id")
	followingID_uint64, _ := strconv.ParseUint(followingID_str, 10, 32)
	followingID := uint(followingID_uint64)

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&model.Follower{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			tx.Model(&model.User{}).Where("id = ?", followerID).UpdateColumn("following_count", gorm.Expr("GREATEST(0, following_count - 1)"))
			tx.Model(&model.User{}).Where("id = ?", followingID).UpdateColumn("followers_count", gorm.Expr("GREATEST(0, followers_count - 1)"))
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfollow user", "details": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Unfollowed successfully"})
}
