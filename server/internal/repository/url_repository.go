package repository

import (
	"context"

	"bitly-url/internal/entity"
)

type URLRepository interface {
	Save(ctx context.Context, url *entity.URL) error
	FindByID(ctx context.Context, id string) (*entity.URL, error)
	FindByShort(ctx context.Context, short string) (*entity.URL, error)
	FindAll(ctx context.Context) ([]*entity.URL, error)
	Delete(ctx context.Context, id string) error
}
