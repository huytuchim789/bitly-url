package middleware

import (
	"strconv"
	"time"

	"bitly-url/internal/cache"
	"bitly-url/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	cache   cache.Cache
	limit   int
	window  time.Duration
	enabled bool
}

func NewRateLimiter(c cache.Cache, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		cache:   c,
		limit:   limit,
		window:  window,
		enabled: c != nil,
	}
}

func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.enabled {
			c.Next()
			return
		}

		ip := c.ClientIP()
		key := "ratelimit:" + c.FullPath() + ":" + ip

		count, err := rl.cache.Incr(c.Request.Context(), key)
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			rl.cache.Expire(c.Request.Context(), key, rl.window)
		}

		if count > int64(rl.limit) {
			c.Error(errors.ErrRateLimited)
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(rl.limit-int(count)))
		c.Next()
	}
}
