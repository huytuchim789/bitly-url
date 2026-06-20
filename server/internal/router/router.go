package router

import (
	"bitly-url/internal/handler"
	"bitly-url/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(h *handler.URLHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.RequestID())
	r.Use(middleware.ErrorHandler())
	r.Use(cors.Default())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	api := r.Group("/api")
	{
		api.POST("/shorten", h.Shorten)
			api.GET("/urls", h.List)
		api.GET("/urls/:short", h.GetByShort)
	}

	r.GET("/:short", h.Redirect)

	return r
}
