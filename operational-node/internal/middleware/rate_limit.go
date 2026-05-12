package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/andreiOpran/licenta/operational-node/internal/config"
	"github.com/andreiOpran/licenta/operational-node/utils/realip"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// client holds the rate limiter and last time the IP was seen
type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = make(map[string]*client)
)

// init starts background garbage collector when application boots
func init() {
	go cleanupOldClients()
}

func cleanupOldClients() {
	for {
		time.Sleep(time.Minute) // GC cycle interval

		mu.Lock()
		for ip, c := range clients {
			if time.Since(c.lastSeen) > 3*time.Minute {
				delete(clients, ip)
			}
		}
		mu.Unlock()
	}
}

func IPRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := realip.Get(c)

		mu.Lock()

		// if the IP is new, initialize token bucket for it
		if _, found := clients[ip]; !found {
			clients[ip] = &client{
				limiter: rate.NewLimiter(rate.Limit(config.Env.RateLimitRPS), config.Env.RateLimitBurst),
			}
		}

		// update last seen timestamp so GC doesn't delete active users
		clients[ip].lastSeen = time.Now()

		// attempt to consume token
		allow := clients[ip].limiter.Allow()
		mu.Unlock()

		if !allow {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": ErrTooManyRequests,
			})
			return
		}

		c.Next()
	}
}
