package httpserver

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll || origin == "" || originAllowed(origin, allowedOrigins) {
			if origin != "" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
			} else if allowAll {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func originAllowed(origin string, allowed []string) bool {
	trimmed := strings.TrimSpace(origin)
	for _, item := range allowed {
		if strings.TrimSpace(item) == trimmed {
			return true
		}
	}
	return false
}

