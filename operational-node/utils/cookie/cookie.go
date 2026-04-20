package cookie

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/gin-gonic/gin"
)

// SetHttpOnly sets a given value as a HttpOnly cookie on a given path
func SetHttpOnly(c *gin.Context, name, value string, maxAge int, path string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		value,
		maxAge,
		path,                    // restricted to this path only
		config.Env.CookieDomain, // domain
		config.Env.CookieSecure, // secure
		true,                    // httpOnly
	)
}

// Clear clears a cookie by name from a given path
func Clear(c *gin.Context, name, path string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		name,
		"",
		-1, // MaxAge <0 -  deletes cookie instantly
		path,
		config.Env.CookieDomain, // domain
		config.Env.CookieSecure, // secure
		true,                    // httpOnly
	)
}
