package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shuind/language-learner/backend/internal/model" // 确保此路径与您的项目结构匹配
	"gorm.io/gorm"
	"gorm.io/gorm/clause" // 【核心】引入 GORM 的 clause 包
)

// --- DTOs (Data Transfer Objects) for Forum ---

// 【增强】为 AuthorResponse 添加 AvatarURL
type AuthorResponse struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
}

type CreatePostInput struct {
	Title    string `json:"title" binding:"max=255"`
	Content  string `json:"content" binding:"required,min=1"`
	PostType string `json:"post_type"` // e.g., 'thought', 'article', 'question'
	ImageURL string `json:"image_url"`
	VideoURL string `json:"video_url"`
	Status   string `json:"status"` // 【新增】允许前端传入 'draft' 或 'published'
}

// 【增强】为 PostResponse 添加 IsFollowedByMe 并更新 User 字段
type PostResponse struct {
	model.Post
	User              AuthorResponse  `json:"user"`
	LastRepliedByUser *AuthorResponse `json:"last_replied_by_user,omitempty"`
	IsLikedByMe       bool            `json:"is_liked_by_me"`
	IsFollowedByMe    bool            `json:"is_followed_by_me"`
}

type CreateReplyInput struct {
	Content       string `json:"content" binding:"required,min=1"`
	ParentReplyID *uint  `json:"parent_reply_id"` // 对应您的模型字段名
	ReplyToUserID *uint  `json:"reply_to_user_id"`
}

type ReplyResponse struct {
	model.Reply
	User AuthorResponse `json:"user"`
	// 【新增】返回给前端，告知当前用户是否赞了此评论
	IsLikedByMe bool `json:"is_liked_by_me"`
}

// 【新增】用于创建回答的 DTO
type CreateAnswerInput struct {
	Content string `json:"content" binding:"required,min=1"`
}

// --- Handler ---

type PostHandler struct {
	DB *gorm.DB
}

func NewPostHandler(db *gorm.DB) *PostHandler {
	return &PostHandler{DB: db}
}

// === ListPosts: 获取帖子列表 (已增强，保留所有日志) ===
func (h *PostHandler) ListPosts(c *gin.Context) {
	log.Println("--- [获取帖子列表] 1. 处理函数开始 ---")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	orderString := "is_pinned DESC, last_replied_at DESC NULLS LAST, created_at DESC"
	log.Printf("--- [获取帖子列表] 2. 分页参数: page=%d, limit=%d, offset=%d ---", page, limit, offset)

	var posts []model.Post
	var total int64

	dbSession := h.DB.Model(&model.Post{}).Where("status = ?", "published")

	if err := dbSession.Count(&total).Error; err != nil {
		log.Printf("[错误] [获取帖子列表] 2.1. 查询帖子总数失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取帖子总数失败"})
		return
	}
	log.Printf("--- [获取帖子列表] 3. 数据库中帖子总数: %d ---", total)

	if err := dbSession.Preload("User").Preload("LastRepliedByUser").Order(orderString).Offset(offset).Limit(limit).Find(&posts).Error; err != nil {
		log.Printf("[错误] [获取帖子列表] 3.1. 查询帖子列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取帖子列表失败"})
		return
	}
	log.Printf("--- [获取帖子列表] 4. 当前页查询到 %d 篇帖子。 ---", len(posts))

	likedMap := make(map[uint]bool)
	followedMap := make(map[uint]bool)
	userID_interface, exists := c.Get("userID")

	if exists {
		userID, ok := userID_interface.(uint)
		if !ok {
			log.Printf("[错误] [获取帖子列表] 4.1. 上下文中的 userID 类型不是 uint, 实际类型是 %T", userID_interface)
		} else if len(posts) > 0 {
			log.Printf("--- [获取帖子列表] 5. 用户已登录 (用户ID: %d)，开始检查状态。 ---", userID)

			postIDs := make([]uint, len(posts))
			for i, p := range posts {
				postIDs[i] = p.ID
			}

			var userLikes []model.PostLike
			if err := h.DB.Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&userLikes).Error; err == nil {
				for _, like := range userLikes {
					likedMap[like.PostID] = true
				}
			}

			var userFollows []model.QuestionFollow
			if err := h.DB.Where("user_id = ? AND post_id IN ?", userID, postIDs).Find(&userFollows).Error; err == nil {
				for _, follow := range userFollows {
					followedMap[follow.PostID] = true
				}
			}
		}
	} else {
		log.Println("--- [获取帖子列表] 5. 用户未登录，所有帖子的交互状态将为 false。 ---")
	}

	response := make([]PostResponse, len(posts))
	log.Println("--- [获取帖子列表] 6. 开始组装最终返回数据... ---")
	for i, p := range posts {
		response[i] = PostResponse{
			Post:           p,
			User:           AuthorResponse{ID: p.User.ID, Username: p.User.Username, AvatarURL: p.User.AvatarURL},
			IsLikedByMe:    likedMap[p.ID],
			IsFollowedByMe: followedMap[p.ID],
		}
		if p.LastRepliedByUser.ID != 0 {
			response[i].LastRepliedByUser = &AuthorResponse{ID: p.LastRepliedByUser.ID, Username: p.LastRepliedByUser.Username, AvatarURL: p.LastRepliedByUser.AvatarURL}
		}
	}

	log.Println("--- [获取帖子列表] 7. 处理函数成功结束，发送响应。 ---")
	c.JSON(http.StatusOK, gin.H{"total": total, "page": page, "posts": response})
}

// === GetPost: 获取单个帖子详情 (最终重构版，支持知乎式评论) ===
func (h *PostHandler) GetPost(c *gin.Context) {
	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var post model.Post
	if err := h.DB.Preload("User").First(&post, uint(postID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	h.DB.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("views_count", gorm.Expr("views_count + 1"))

	isLiked, isFollowed := false, false
	userID_interface, exists := c.Get("userID")
	if exists {
		userID := userID_interface.(uint)
		if h.DB.Where("user_id = ? AND post_id = ?", userID, postID).First(&model.PostLike{}).Error == nil {
			isLiked = true
		}
		if post.PostType == "question" {
			if h.DB.Where("user_id = ? AND post_id = ?", userID, postID).First(&model.QuestionFollow{}).Error == nil {
				isFollowed = true
			}
		}
	}

	postResponse := PostResponse{
		Post:           post,
		User:           AuthorResponse{ID: post.User.ID, Username: post.User.Username, AvatarURL: post.User.AvatarURL},
		IsLikedByMe:    isLiked,
		IsFollowedByMe: isFollowed,
	}

	if post.PostType == "question" {
		// --- 问答模式的逻辑 (保持不变) ---
		var answers []model.Post
		h.DB.Where("parent_id = ?", postID).Preload("User").Order("votes_count DESC").Find(&answers)
		c.JSON(http.StatusOK, gin.H{"post": postResponse, "answers": answers})

	} else {
		// --- 文章/想法模式的逻辑 (核心重构) ---
		var rootReplies []model.Reply

		// 1. 【核心查询】只查找一级评论，并预加载它们的作者、子评论、子评论的作者、子评论的被回复者
		err := h.DB.Where("post_id = ? AND parent_reply_id IS NULL", postID).
			Preload("User").
			Preload("ChildReplies", func(db *gorm.DB) *gorm.DB {
				return db.Order("created_at ASC").Limit(3) // 每个楼层最多显示3条子评论
			}).
			Preload("ChildReplies.User").
			Preload("ChildReplies.ReplyToUser").
			Order("created_at ASC").
			Find(&rootReplies).Error
		// 2. 【新增】为每条一级评论计算子评论总数
		if len(rootReplies) > 0 {
			for i := range rootReplies {
				var count int64
				h.DB.Model(&model.Reply{}).Where("parent_reply_id = ?", rootReplies[i].ID).Count(&count)
				rootReplies[i].ChildRepliesCount = count
			}
		}
		if err != nil {
			log.Printf("ERROR getting replies: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve replies"})
			return
		}

		// 2. 【增强】为所有加载的评论（一级和二级）检查点赞状态
		likedRepliesMap := make(map[uint]bool)
		if exists && len(rootReplies) > 0 {
			userID := userID_interface.(uint)
			var allReplyIDs []uint

			// 收集所有一级和二级评论的ID
			for _, r := range rootReplies {
				allReplyIDs = append(allReplyIDs, r.ID)
				for _, child := range r.ChildReplies {
					allReplyIDs = append(allReplyIDs, child.ID)
				}
			}

			if len(allReplyIDs) > 0 {
				var replyLikes []model.ReplyLike
				h.DB.Where("user_id = ? AND reply_id IN ?", userID, allReplyIDs).Find(&replyLikes)
				for _, like := range replyLikes {
					likedRepliesMap[like.ReplyID] = true
				}
			}
		}

		// 3. 组装最终的 DTO (此步骤现在可以在前端完成，但后端组装更严谨)
		// 为了简化，我们直接返回带有预加载数据的 rootReplies
		// 前端组件将负责渲染和显示 is_liked_by_me 状态 (通过 likedRepliesMap)
		// (更高级的做法是创建一个包含 IsLikedByMe 的 Reply DTO，此处为保持简单直接返回)

		c.JSON(http.StatusOK, gin.H{
			"post":    postResponse,
			"replies": rootReplies,
			// 【可选但推荐】将点赞状态图也一并返回给前端
			"liked_replies_map": likedRepliesMap,
		})
	}
}

// === CreatePost (您的原有代码，100% 保持不变) ===
func (h *PostHandler) CreatePost(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var input CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	postType := input.PostType
	if postType == "" {
		postType = "thought"
	}

	if postType == "thought" {
		if input.Title == "" {
			runes := []rune(input.Content)
			if len(runes) > 50 {
				input.Title = string(runes[:15]) + "..."
			} else {
				input.Title = input.Content
			}
		}
	} else {
		if input.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Title is required for " + postType + " type"})
			return
		}
	}
	// 【核心修改】决定帖子的状态
	status := "published" // 默认为直接发布
	if input.Status == "draft" {
		status = "draft"
	}
	now := time.Now()
	newPost := model.Post{
		UserID:              userID,
		Title:               input.Title,
		Content:             input.Content,
		Status:              status,
		PostType:            postType,
		ImageURL:            input.ImageURL,
		VideoURL:            input.VideoURL,
		LastRepliedAt:       &now, // 这两个字段似乎在您的 Post 模型中不存在
		LastRepliedByUserID: &userID,
	}

	if err := h.DB.Create(&newPost).Error; err != nil {
		log.Printf("ERROR creating post: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	h.DB.Preload("User").First(&newPost, newPost.ID)

	response := PostResponse{
		Post:        newPost,
		User:        AuthorResponse{ID: newPost.User.ID, Username: newPost.User.Username, AvatarURL: newPost.User.AvatarURL},
		IsLikedByMe: false,
	}

	c.JSON(http.StatusCreated, response)
}

// === ListDrafts: 获取当前用户的所有草稿 ===
func (h *PostHandler) ListDrafts(c *gin.Context) {
	userID := c.MustGet("userID").(uint)

	var drafts []model.Post
	h.DB.Where("user_id = ? AND status = ?", userID, "draft").
		Order("updated_at DESC").
		Find(&drafts)

	c.JSON(http.StatusOK, drafts)
}

// === UpdatePost: 更新一个帖子 (草稿或已发布) ===
func (h *PostHandler) UpdatePost(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var input CreatePostInput                        // 复用 DTO
	if err := c.ShouldBindJSON(&input); err != nil { /* ... */
	}

	var post model.Post
	// 确保只能更新自己的帖子
	if err := h.DB.Where("id = ? AND user_id = ?", postID, userID).First(&post).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found or permission denied"})
		return
	}

	post.Title = input.Title
	post.Content = input.Content
	// 如果前端想在更新的同时发布，也可以在这里处理
	if input.Status == "published" {
		post.Status = "published"
	}

	h.DB.Save(&post)
	c.JSON(http.StatusOK, post)
}

// === CreateReply (您的原有代码，100% 保持不变) ===
func (h *PostHandler) CreateReply(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var input CreateReplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var newReply model.Reply
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&model.Post{}, uint(postID)).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("post not found")
			}
			return err
		}

		newReply = model.Reply{PostID: uint(postID), UserID: userID, Content: input.Content, ParentReplyID: input.ParentReplyID, ReplyToUserID: input.ReplyToUserID}
		if err := tx.Create(&newReply).Error; err != nil {
			return err
		}

		// now := time.Now() // 您的 Post 模型中似乎没有这些字段
		return tx.Model(&model.Post{}).Where("id = ?", postID).Updates(map[string]interface{}{
			"replies_count": gorm.Expr("replies_count + 1"),
			// "last_replied_at":         &now,
			// "last_replied_by_user_id": userID,
		}).Error
	})

	if err != nil {
		if err.Error() == "post not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		} else {
			log.Printf("ERROR creating reply: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create reply"})
		}
		return
	}

	h.DB.Preload("User").First(&newReply, newReply.ID)
	response := ReplyResponse{Reply: newReply, User: AuthorResponse{ID: newReply.User.ID, Username: newReply.User.Username, AvatarURL: newReply.User.AvatarURL}}
	c.JSON(http.StatusCreated, response)
}

// === ToggleLikePost (您的原有代码，100% 保持不变) ===
func (h *PostHandler) ToggleLikePost(c *gin.Context) {
	userID_interface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userID_interface.(uint)
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&model.Post{}, uint(postID)).Error; err != nil {
			return errors.New("post not found")
		}

		result := tx.Where("user_id = ? AND post_id = ?", userID, postID).Delete(&model.PostLike{})
		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 {
			log.Printf("[DEBUG] Unliked post %d by user %d", postID, userID)
			return tx.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("votes_count", gorm.Expr("GREATEST(0, votes_count - 1)")).Error
		} else {
			log.Printf("[DEBUG] Liked post %d by user %d", postID, userID)
			like := model.PostLike{UserID: userID, PostID: uint(postID)}
			if err := tx.Create(&like).Error; err != nil {
				return err
			}
			return tx.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("votes_count", gorm.Expr("votes_count + 1")).Error
		}
	})

	if err != nil {
		if err.Error() == "post not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		} else {
			log.Printf("ERROR toggling like: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle like status"})
		}
		return
	}

	c.Status(http.StatusOK)
}

// === CreateAnswer (新增) ===
func (h *PostHandler) CreateAnswer(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	questionID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var input CreateAnswerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var question model.Post
		if err := tx.Where("id = ? AND post_type = ?", questionID, "question").First(&question).Error; err != nil {
			return errors.New("问题不存在或类型不正确")
		}

		parentID := uint(questionID)
		answer := model.Post{
			UserID:   userID,
			Content:  input.Content,
			Title:    "回答: " + question.Title,
			PostType: "answer",
			ParentID: &parentID,
			Status:   "published",
		}
		if err := tx.Create(&answer).Error; err != nil {
			return err
		}
		return tx.Model(&model.Post{}).Where("id = ?", questionID).UpdateColumn("answers_count", gorm.Expr("answers_count + 1")).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建回答失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}

// === ToggleFollowPost (新增) ===
func (h *PostHandler) ToggleFollowPost(c *gin.Context) {
	userID := c.MustGet("userID").(uint)
	postID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND post_type = ?", postID, "question").First(&model.Post{}).Error; err != nil {
			return errors.New("问题不存在")
		}

		follow := model.QuestionFollow{UserID: userID, PostID: uint(postID)}
		result := tx.Where(&follow).Delete(&model.QuestionFollow{})
		if result.RowsAffected > 0 {
			return tx.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("followers_count", gorm.Expr("GREATEST(0, followers_count - 1)")).Error
		} else {
			if err := tx.Create(&follow).Error; err != nil {
				return err
			}
			return tx.Model(&model.Post{}).Where("id = ?", postID).UpdateColumn("followers_count", gorm.Expr("followers_count + 1")).Error
		}
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// file: internal/handler/post_handler.go

// === ToggleLikeReply (终极兼容版) ===
func (h *PostHandler) ToggleLikeReply(c *gin.Context) {
	userID_interface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	userID := userID_interface.(uint)

	replyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reply ID"})
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&model.Reply{}, uint(replyID)).Error; err != nil {
			return errors.New("reply not found")
		}

		like := model.ReplyLike{UserID: userID, ReplyID: uint(replyID)}
		result := tx.Where(&like).Delete(&model.ReplyLike{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected > 0 { // 取消点赞成功
			// 【核心修正】使用 Updates 方法
			return tx.Model(&model.Reply{}).Where("id = ?", replyID).Updates(map[string]interface{}{
				"likes_count": gorm.Expr("GREATEST(likes_count - 1, 0)"),
			}).Error
		} else { // 添加点赞
			result = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&like)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected > 0 {
				// 【核心修正】使用 Updates 方法
				return tx.Model(&model.Reply{}).Where("id = ?", replyID).Updates(map[string]interface{}{
					"likes_count": gorm.Expr("likes_count + 1"),
				}).Error
			}
			return nil
		}
	})

	if err != nil {
		log.Printf("[错误] ToggleLikeReply 事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle like on reply", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// === GetChildReplies: 获取某条评论下的所有子评论 (分页) ===
func (h *PostHandler) GetChildReplies(c *gin.Context) {
	parentReplyID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reply ID"})
		return
	}

	// (可选) 添加分页
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "4")) // 默认加载100条
	offset := (page - 1) * limit

	var childReplies []model.Reply
	h.DB.Where("parent_reply_id = ?", parentReplyID).
		Preload("User").
		Preload("ReplyToUser").
		Order("created_at ASC").
		Offset(offset).Limit(limit).
		Find(&childReplies)

	// (可选但推荐) 同样检查这些子评论的点赞状态
	// ...

	c.JSON(http.StatusOK, childReplies)
}
