package queue

import (
	"context"

	"bitly-url/internal/entity"
)

type ClickQueue interface {
	Publish(ctx context.Context, click *entity.Click) error
	Consume(ctx context.Context) (<-chan *entity.Click, error)
	Close() error
}
