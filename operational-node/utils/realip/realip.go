package realip

import "github.com/gin-gonic/gin"

// Get returns CF-Connecting-IP when behind Cloudflare, falls back to gin ClientIP.
func Get(c *gin.Context) string {
	if cf := c.GetHeader("CF-Connecting-IP"); cf != "" {
		return cf
	}
	return c.ClientIP()
}
