package middleware

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/gin-gonic/gin"
)

func InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := c.GetHeader("X-Internal-Secret")
		if secret == "" || secret != config.Env.InternalEndpointsSecret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
