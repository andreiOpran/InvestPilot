package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds standard http protection headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		// prevent mime-sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		// enable xss protection
		c.Header("X-XSS-Protection", "1; mode=block")

		c.Next()
	}
}
