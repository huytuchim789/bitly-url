package middleware

import (
	"log/slog"
	"net/http"

	"bitly-url/internal/pkg/errors"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		if appErr, ok := err.(*errors.AppError); ok {
			c.JSON(appErr.Code, gin.H{"error": appErr.Message})

			if appErr.Code >= http.StatusInternalServerError {
				slog.Error("server error",
					"error", appErr.Message,
					"request_id", c.GetString("request_id"),
					"path", c.Request.URL.Path,
				)
			}
			return
		}

		slog.Error("unhandled error",
			"error", err,
			"request_id", c.GetString("request_id"),
			"path", c.Request.URL.Path,
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}
