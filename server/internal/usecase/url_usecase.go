package usecase

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/url"
	"strings"
	"time"

	"bitly-url/internal/cache"
	"bitly-url/internal/entity"
	"bitly-url/internal/metrics"
	"bitly-url/internal/pkg/errors"
	"bitly-url/internal/repository"

	"github.com/google/uuid"
)

const (
	cacheTTL       = 1 * time.Hour
	cacheKeyPrefix = "short:"
	maxURLSize     = 2048
	shortCodeLen   = 6
)

type URLUseCase struct {
	repo       repository.URLRepository
	cache      cache.Cache
	clickQueue chan *entity.Click
}

func NewURLUseCase(repo repository.URLRepository, c cache.Cache) *URLUseCase {
	uc := &URLUseCase{
		repo:       repo,
		cache:      c,
		clickQueue: make(chan *entity.Click, 10000),
	}
	go uc.clickWorker()
	return uc
}

func (uc *URLUseCase) Shorten(ctx context.Context, original string) (*entity.URL, error) {
	if err := uc.validateOriginal(original); err != nil {
		return nil, err
	}

	short, err := uc.generateUniqueShort(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	url := &entity.URL{
		ID:        uuid.New(),
		Short:     short,
		Original:  original,
		Clicks:    0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.repo.Save(ctx, url); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, errors.ErrShortCode
		}
		slog.Error("failed to save url", "error", err)
		return nil, errors.ErrInternal
	}

	uc.cache.Set(ctx, cacheKeyPrefix+short, original, cacheTTL)
	return url, nil
}

func (uc *URLUseCase) FindByShort(ctx context.Context, short string) (*entity.URL, error) {
	cacheKey := cacheKeyPrefix + short

	if cached, err := uc.cache.Get(ctx, cacheKey); err == nil {
		metrics.CacheHitsTotal.Inc()
		metrics.URLRedirectTotal.Inc()

		if err := uc.validateRedirectTarget(cached); err != nil {
			return nil, err
		}
		uc.cache.Incr(ctx, cacheKey+":clicks")

		return &entity.URL{
			Short:    short,
			Original: cached,
		}, nil
	}
	metrics.CacheMissesTotal.Inc()

	url, err := uc.repo.FindByShort(ctx, short)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	if url.IsExpired() {
		return nil, errors.ErrGone
	}

	if err := uc.validateRedirectTarget(url.Original); err != nil {
		return nil, err
	}

	metrics.URLRedirectTotal.Inc()
	uc.cache.Set(ctx, cacheKey, url.Original, cacheTTL)
	return url, nil
}

func (uc *URLUseCase) TrackClick(shortID uuid.UUID, ip, userAgent, referer string) {
	uc.clickQueue <- &entity.Click{
		ID:        uuid.New(),
		ShortID:   shortID,
		IP:        ip,
		UserAgent: userAgent,
		Referer:   referer,
		Timestamp: time.Now(),
	}
}

func (uc *URLUseCase) FindAll(ctx context.Context, limit, offset int) ([]*entity.URL, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	urls, err := uc.repo.FindAll(ctx, limit, offset)
	if err != nil {
		slog.Error("failed to list urls", "error", err)
		return nil, errors.ErrInternal
	}
	return urls, nil
}

func (uc *URLUseCase) Delete(ctx context.Context, id string) error {
	if err := uc.repo.Delete(ctx, id); err != nil {
		slog.Error("failed to delete url", "error", err)
		return errors.ErrInternal
	}
	return nil
}

func (uc *URLUseCase) clickWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	buf := make([]*entity.Click, 0, 100)

	for {
		select {
		case click := <-uc.clickQueue:
			buf = append(buf, click)
			if len(buf) >= 100 {
				uc.flushClicks(buf)
				buf = buf[:0]
			}
		case <-ticker.C:
			if len(buf) > 0 {
				uc.flushClicks(buf)
				buf = buf[:0]
			}
		}
	}
}

func (uc *URLUseCase) flushClicks(clicks []*entity.Click) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if repo, ok := uc.repo.(interface{ BatchInsert(context.Context, []*entity.Click) error }); ok {
		if err := repo.BatchInsert(ctx, clicks); err != nil {
			slog.Error("failed to batch insert clicks", "error", err)
		}
	}
}

func (uc *URLUseCase) validateOriginal(raw string) error {
	if len(raw) > maxURLSize {
		return fmt.Errorf("url exceeds maximum length of %d characters: %w", maxURLSize, errors.ErrBadRequest)
	}

	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return fmt.Errorf("invalid url: %w", errors.ErrBadRequest)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url scheme must be http or https: %w", errors.ErrBadRequest)
	}

	if err := uc.validateRedirectTarget(raw); err != nil {
		return err
	}

	return nil
}

func (uc *URLUseCase) validateRedirectTarget(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid redirect target: %w", errors.ErrBadRequest)
	}

	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return errors.ErrBadRequest
	}

	privateSuffixes := []string{
		"localhost",
		"127.0.0.1",
		"0.0.0.0",
		"10.",
		"172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.",
		"172.24.", "172.25.", "172.26.", "172.27.",
		"172.28.", "172.29.", "172.30.", "172.31.",
		"192.168.",
		"169.254.",
		"::1",
		"fc00:", "fd00:",
	}

	for _, suffix := range privateSuffixes {
		if strings.HasPrefix(hostname, suffix) {
			return errors.ErrForbidden
		}
	}

	ips, err := net.LookupIP(hostname)
	if err == nil {
		for _, ip := range ips {
			if ip.IsPrivate() || ip.IsLoopback() || ip.IsUnspecified() {
				return errors.ErrForbidden
			}
		}
	}

	return nil
}

func (uc *URLUseCase) generateUniqueShort(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		code, err := generateShortCode()
		if err != nil {
			return "", err
		}

		exists := true
		cacheKey := cacheKeyPrefix + code
		if _, err := uc.cache.Get(ctx, cacheKey); err != nil {
			exists = false
		} else {
			continue
		}

		if !exists {
			if _, err := uc.repo.FindByShort(ctx, code); err != nil {
				return code, nil
			}
		}
	}

	return "", fmt.Errorf("failed to generate unique short code after 10 attempts: %w", errors.ErrInternal)
}

func generateShortCode() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, shortCodeLen)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		code[i] = charset[n.Int64()]
	}
	return string(code), nil
}
