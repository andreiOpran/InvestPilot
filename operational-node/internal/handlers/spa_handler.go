package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SPAFallbackHandler serves the frontend index.html for unknown non-API routes
func SPAFallbackHandler(indexPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// if request was meant for API endpoint, return 404 json
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}

		// otherwise, serve SPA index.html so React Router can take over
		c.File(indexPath)
	}
}
