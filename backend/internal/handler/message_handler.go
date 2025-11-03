// file: internal/handler/message_handler.go

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shuind/language-learner/backend/internal/model" // 确保路径正确
	"gorm.io/gorm"
)

type MessageHandler struct {
	DB *gorm.DB
}

func NewMessageHandler(db *gorm.DB) *MessageHandler {
	return &MessageHandler{DB: db}
}

type SendMessageInput struct {
	RecipientID uint   `json:"recipient_id" binding:"required"`
	Content     string `json:"content" binding:"required"`
}

// SendMessage 发送一条私信
func (h *MessageHandler) SendMessage(c *gin.Context) {
	senderID_interface, _ := c.Get("userID")
	senderID := senderID_interface.(uint)

	var input SendMessageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	/* if senderID == input.RecipientID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能给自己发私信"})
		return
	} */

	message := model.Message{
		SenderID:    senderID,
		RecipientID: input.RecipientID,
		Content:     input.Content,
	}

	if err := h.DB.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送失败"})
		return
	}
	c.JSON(http.StatusCreated, message)
}

// GetConversation 获取与某个用户的对话历史
func (h *MessageHandler) GetConversation(c *gin.Context) {
	myID_interface, _ := c.Get("userID")
	myID := myID_interface.(uint)

	otherUserID, _ := strconv.ParseUint(c.Param("userID"), 10, 32)

	var messages []model.Message
	h.DB.Where("(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		myID, otherUserID, otherUserID, myID).
		Order("created_at ASC").
		Find(&messages)

	c.JSON(http.StatusOK, messages)
}
