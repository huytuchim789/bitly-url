package middleware

import (
	"strconv"
	"time"

	"bitly-url/internal/metrics"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics.HTTPRequestsActive.Inc()
		start := time.Now()

		c.Next()

		latency := time.Since(start).Milliseconds()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, strconv.Itoa(c.Writer.Status())).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(float64(latency))
		metrics.HTTPRequestsActive.Dec()
	}
}
