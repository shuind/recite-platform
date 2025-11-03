package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware 是一个“严格”的认证中间件。
// 如果 Token 无效或不存在，它会中断请求并返回 401 错误。
// 用于保护需要强制登录的接口（如发帖、点赞、修改个人信息等）。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtKey := []byte(os.Getenv("JWT_SECRET"))
		if len(jwtKey) == 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "JWT secret not configured"})
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}
		tokenString := parts[1]

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		userIDStr := claims.Subject
		userID64, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			return
		}
		
		c.Set("userID", uint(userID64))
		c.Next()
	}
}


// ===================================================================
// ===================== 【核心新增代码在这里】 ==========================
// ===================================================================

// AuthUserMiddleware 是一个“可选”的认证中间件。
// 它会尝试解析 Token，如果成功就设置 userID；如果失败或没有 Token，它也直接放行。
// 用于公开接口，以便这些接口也能知道当前访问者是否是登录用户。
func AuthUserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		jwtKey := []byte(os.Getenv("JWT_SECRET"))
		authHeader := c.GetHeader("Authorization")

		// 如果没有 Authorization 头，或者格式不正确，直接调用 c.Next() 放行
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}
		tokenString := parts[1]

		// 尝试解析和验证 token
		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return jwtKey, nil
		})

		// 如果 token 无效或过期，也直接放行
		if err != nil || !token.Valid {
			c.Next()
			return
		}

		// 只有在 token 完全有效的情况下，才解析 userID 并存入 context
		userIDStr := claims.Subject
		userID64, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.Next() // userID 格式错误，也放行
			return
		}
		
		// 成功解析后，设置 userID
		c.Set("userID", uint(userID64))
		
		// 无论如何，都继续处理请求
		c.Next()
	}
}