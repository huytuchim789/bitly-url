# Backend Refactor Plan

## Tổng quan

Refactor Go backend với top thư viện Go 2026, giữ nguyên clean architecture.

## Dependencies mới (go.mod)

| Package | Lý do |
|---------|-------|
| `github.com/caarlos0/env/v11` | Config struct tags thay vì os.LookupEnv |
| `github.com/gin-contrib/cors` | CORS middleware chuẩn |
| `github.com/google/uuid` | Gen UUID cho request ID |
| `github.com/stretchr/testify` | Testing framework |
| `github.com/swaggo/swag` | Chuyển từ indirect → direct (cần import) |

## Files cần sửa (7 files)

### 1. `go.mod`
- Thêm: caarlos0/env, gin-contrib/cors, google/uuid, testify
- Chuyển swaggo/swag từ indirect → direct

### 2. `internal/config/config.go`
```go
package config

import "github.com/caarlos0/env/v11"

type Config struct {
    Port        string `env:"PORT" envDefault:"8080"`
    DatabaseURL string `env:"DATABASE_URL" envDefault:"postgres://postgres:postgres@localhost:5432/bitly?sslmode=disable"`
    LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() (*Config, error) {
    cfg := &Config{}
    if err := env.Parse(cfg); err != nil {
        return nil, err
    }
    return cfg, nil
}
```

### 3. `cmd/main.go` — graceful shutdown + slog
```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

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

    repo := postgres.NewURLPostgresRepo(db)
    uc := usecase.NewURLUseCase(repo)
    h := handler.NewURLHandler(uc)
    r := router.New(h)

    srv := &http.Server{
        Addr:    ":" + cfg.Port,
        Handler: r,
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        slog.Info("server starting", "port", cfg.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    <-ctx.Done()
    slog.Info("shutting down...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
```

### 4. `internal/router/router.go` — thêm CORS + middleware + healthz
```go
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
    }

    r.GET("/:short", h.Redirect)

    return r
}
```

## Files cần tạo mới (3 files)

### 5. `internal/middleware/logger.go`
```go
package middleware

import (
    "log/slog"
    "time"

    "github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path

        c.Next()

        slog.Info("request",
            "method", c.Request.Method,
            "path", path,
            "status", c.Writer.Status(),
            "duration", time.Since(start).String(),
            "request_id", c.GetString("request_id"),
        )
    }
}
```

### 6. `internal/middleware/requestid.go`
```go
package middleware

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

func RequestID() gin.HandlerFunc {
    return func(c *gin.Context) {
        id := c.GetHeader("X-Request-ID")
        if id == "" {
            id = uuid.New().String()
        }
        c.Set("request_id", id)
        c.Header("X-Request-ID", id)
        c.Next()
    }
}
```

### 7. `internal/middleware/error.go`
```go
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
            return
        }

        slog.Error("unhandled error", "error", err, "request_id", c.GetString("request_id"))
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
    }
}
```

### 8. `internal/pkg/errors/errors.go`
```go
package errors

import "net/http"

type AppError struct {
    Code    int    `json:"code"`
    Message string `json:"error"`
}

func (e *AppError) Error() string {
    return e.Message
}

var (
    ErrNotFound   = &AppError{Code: http.StatusNotFound, Message: "url not found"}
    ErrBadRequest = &AppError{Code: http.StatusBadRequest, Message: "bad request"}
    ErrInternal   = &AppError{Code: http.StatusInternalServerError, Message: "internal server error"}
)
```

### 9. `internal/handler/url_handler.go` — sử dụng typed errors
```go
package handler

import (
    "net/http"

    "bitly-url/internal/pkg/errors"
    "bitly-url/internal/usecase"

    "github.com/gin-gonic/gin"
    "github.com/go-playground/validator/v10"
)

type URLHandler struct {
    uc *usecase.URLUseCase
}

func NewURLHandler(uc *usecase.URLUseCase) *URLHandler {
    return &URLHandler{uc: uc}
}

type ShortenRequest struct {
    URL string `json:"url" binding:"required,url"`
}

func (h *URLHandler) Shorten(c *gin.Context) {
    var req ShortenRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.Error(errors.ErrBadRequest)
        return
    }

    url, err := h.uc.Shorten(c.Request.Context(), req.URL)
    if err != nil {
        c.Error(err)
        return
    }
    c.JSON(http.StatusCreated, url)
}

func (h *URLHandler) Redirect(c *gin.Context) {
    short := c.Param("short")
    url, err := h.uc.FindByShort(c.Request.Context(), short)
    if err != nil {
        c.Error(err)
        return
    }
    c.Redirect(http.StatusMovedPermanently, url.Original)
}

func (h *URLHandler) List(c *gin.Context) {
    urls, err := h.uc.FindAll(c.Request.Context())
    if err != nil {
        c.Error(err)
        return
    }
    c.JSON(http.StatusOK, urls)
}
```

### 10. `internal/usecase/url_usecase.go` — slog + typed errors
```go
package usecase

import (
    "context"
    "crypto/rand"
    "log/slog"
    "math/big"
    "time"

    "bitly-url/internal/entity"
    "bitly-url/internal/pkg/errors"
    "bitly-url/internal/repository"
)

type URLUseCase struct {
    repo repository.URLRepository
}

func NewURLUseCase(repo repository.URLRepository) *URLUseCase {
    return &URLUseCase{repo: repo}
}

func (uc *URLUseCase) Shorten(ctx context.Context, original string) (*entity.URL, error) {
    short, err := generateShortCode()
    if err != nil {
        return nil, errors.ErrInternal
    }

    url := &entity.URL{
        ID:        short,
        Original:  original,
        Short:     short,
        CreatedAt: time.Now(),
    }

    if err := uc.repo.Save(ctx, url); err != nil {
        slog.Error("failed to save url", "error", err)
        return nil, errors.ErrInternal
    }

    return url, nil
}

func (uc *URLUseCase) FindByShort(ctx context.Context, short string) (*entity.URL, error) {
    url, err := uc.repo.FindByShort(ctx, short)
    if err != nil {
        return nil, errors.ErrNotFound
    }
    return url, nil
}

func (uc *URLUseCase) FindAll(ctx context.Context) ([]*entity.URL, error) {
    urls, err := uc.repo.FindAll(ctx)
    if err != nil {
        slog.Error("failed to list urls", "error", err)
        return nil, errors.ErrInternal
    }
    return urls, nil
}

func generateShortCode() (string, error) {
    const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    code := make([]byte, 6)
    for i := range code {
        n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        if err != nil {
            return "", err
        }
        code[i] = charset[n.Int64()]
    }
    return string(code), nil
}
```

### 11. `server/Makefile`
```makefile
.PHONY: run build swag sqlc test lint

run:
	go run ./cmd/main.go

build:
	go build -o bin/server ./cmd/main.go

swag:
	swag init -g cmd/main.go --output docs

sqlc:
	sqlc generate

test:
	go test ./... -v -count=1

lint:
	golangci-lint run

tidy:
	go mod tidy

all: swag sqlc build
```

### 12. `server/.env.example`
```
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/bitly?sslmode=disable
LOG_LEVEL=info
```

### 13. `server/Dockerfile` — update Go version
```dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

## Thứ tự thực hiện

1. Sửa `go.mod` — thêm deps, chạy `go mod tidy`
2. Tạo `internal/pkg/errors/errors.go`
3. Tạo `internal/middleware/logger.go`, `requestid.go`, `error.go`
4. Sửa `internal/config/config.go`
5. Sửa `internal/handler/url_handler.go`
6. Sửa `internal/usecase/url_usecase.go`
7. Sửa `internal/router/router.go`
8. Sửa `cmd/main.go`
9. Tạo `server/Makefile`, `server/.env.example`
10. Sửa `server/Dockerfile`
11. Chạy `go mod tidy`
