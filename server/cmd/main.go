package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bitly-url/internal/cache"
	"bitly-url/internal/config"
	"bitly-url/internal/database"
	"bitly-url/internal/handler"
	"bitly-url/internal/repository/postgres"
	"bitly-url/internal/router"
	"bitly-url/internal/usecase"

	_ "bitly-url/docs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})))

	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	var c cache.Cache
	redisCache, err := cache.NewRedis(cfg.RedisURL)
	if err != nil {
		slog.Warn("redis not available, running without cache", "error", err)
		c = nil
	} else {
		c = redisCache
		defer redisCache.Close()
	}

	urlRepo := postgres.NewURLPostgresRepo(db)
	clickRepo := postgres.NewClickPostgresRepo(db)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	uc := usecase.NewURLUseCase(urlRepo, clickRepo, c, ctx)
	h := handler.NewURLHandler(uc)
	r := router.New(h, db, c)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
