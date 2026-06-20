package router

import (
	"net/http"
	pprof "net/http/pprof"
	"time"

	"bitly-url/internal/cache"
	"bitly-url/internal/handler"
	"bitly-url/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(h *handler.URLHandler, pool *pgxpool.Pool, c cache.Cache) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.MetricsMiddleware())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/readyz", func(ctx *gin.Context) {
		reqCtx := ctx.Request.Context()
		if err := pool.Ping(reqCtx); err != nil {
			ctx.JSON(503, gin.H{"status": "not ready", "error": "database ping failed"})
			return
		}
		if c != nil {
			if err := c.Ping(reqCtx); err != nil {
				ctx.JSON(503, gin.H{"status": "not ready", "error": "cache ping failed"})
				return
			}
		}
		ctx.JSON(200, gin.H{"status": "ready"})
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	pprofGroup := r.Group("/debug/pprof")
	{
		pprofGroup.GET("/", gin.WrapH(http.HandlerFunc(pprof.Index)))
		pprofGroup.GET("/cmdline", gin.WrapH(http.HandlerFunc(pprof.Cmdline)))
		pprofGroup.GET("/profile", gin.WrapH(http.HandlerFunc(pprof.Profile)))
		pprofGroup.GET("/symbol", gin.WrapH(http.HandlerFunc(pprof.Symbol)))
		pprofGroup.GET("/trace", gin.WrapH(http.HandlerFunc(pprof.Trace)))
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	rl := middleware.NewRateLimiter(c, 100, time.Minute)
	rlStrict := middleware.NewRateLimiter(c, 10, time.Minute)

	api := r.Group("/api")
	{
		api.POST("/shorten", rlStrict.RateLimit(), h.Shorten)
		api.GET("/urls", rl.RateLimit(), h.List)
		api.GET("/urls/:short", rl.RateLimit(), h.GetByShort)
	}

	r.GET("/:short", rl.RateLimit(), h.Redirect)

	return r
}
