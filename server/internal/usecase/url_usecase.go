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
