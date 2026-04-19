package auth

import (
	"net/http"
	"strings"

	"file-upload/backend/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const ContextUserIDKey = "userID"
const ContextUsernameKey = "username"
const ContextIsAdminKey = "isAdmin"

func Middleware(secret string, database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing authorization header"})
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer"))
		if tokenString == "" || tokenString == header {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid authorization header"})
			return
		}

		claims, err := ParseToken(secret, tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token"})
			return
		}

		var user models.User
		if err := database.Select("id", "username", "is_admin").First(&user, "id = ?", claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "user not found"})
			return
		}

		c.Set(ContextUserIDKey, user.ID)
		c.Set(ContextUsernameKey, user.Username)
		c.Set(ContextIsAdminKey, user.IsAdmin)
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, _ := c.Get(ContextIsAdminKey)
		admin, ok := isAdmin.(bool)
		if !ok || !admin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "admin access required"})
			return
		}
		c.Next()
	}
}

