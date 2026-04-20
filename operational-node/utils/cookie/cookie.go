package cookie

import (
	"net/http"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/gin-gonic/gin"
)

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
