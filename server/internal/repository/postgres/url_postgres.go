package postgres

import (
	"context"
	"fmt"

	"bitly-url/internal/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type URLPostgresRepo struct {
	db *pgxpool.Pool
}

func NewURLPostgresRepo(db *pgxpool.Pool) *URLPostgresRepo {
	return &URLPostgresRepo{db: db}
}

func (r *URLPostgresRepo) Save(ctx context.Context, url *entity.URL) error {
	query := `INSERT INTO urls (id, short, original, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.Exec(ctx, query,
		url.ID, url.Short, url.Original, url.ExpiresAt, url.CreatedAt, url.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to save url: %w", err)
	}
	return nil
}

func (r *URLPostgresRepo) FindByID(ctx context.Context, id string) (*entity.URL, error) {
	query := `SELECT id, short, original, clicks, expires_at, created_at, updated_at
		FROM urls WHERE id = $1`
	return r.scanURL(r.db.QueryRow(ctx, query, id))
}

func (r *URLPostgresRepo) FindByShort(ctx context.Context, short string) (*entity.URL, error) {
	query := `SELECT id, short, original, clicks, expires_at, created_at, updated_at
		FROM urls WHERE short = $1`
	return r.scanURL(r.db.QueryRow(ctx, query, short))
}

func (r *URLPostgresRepo) FindAll(ctx context.Context, limit, offset int) ([]*entity.URL, error) {
	query := `SELECT id, short, original, clicks, expires_at, created_at, updated_at
		FROM urls ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list urls: %w", err)
	}
	defer rows.Close()

	urls := make([]*entity.URL, 0)
	for rows.Next() {
		url, err := r.scanURL(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan url: %w", err)
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func (r *URLPostgresRepo) IncrementClicks(ctx context.Context, id string) error {
	query := `UPDATE urls SET clicks = clicks + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *URLPostgresRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM urls WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *URLPostgresRepo) scanURL(row pgx.Row) (*entity.URL, error) {
	var url entity.URL
	err := row.Scan(&url.ID, &url.Short, &url.Original, &url.Clicks, &url.ExpiresAt, &url.CreatedAt, &url.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("url not found: %w", err)
		}
		return nil, fmt.Errorf("failed to scan url: %w", err)
	}
	return &url, nil
}

type ClickPostgresRepo struct {
	db *pgxpool.Pool
}

func NewClickPostgresRepo(db *pgxpool.Pool) *ClickPostgresRepo {
	return &ClickPostgresRepo{db: db}
}

func (r *ClickPostgresRepo) BatchInsert(ctx context.Context, clicks []*entity.Click) error {
	if len(clicks) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `INSERT INTO clicks (id, short_id, ip, user_agent, referer, country, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	for _, c := range clicks {
		batch.Queue(query, c.ID, c.ShortID, c.IP, c.UserAgent, c.Referer, c.Country, c.Timestamp)
		if c.ID == uuid.Nil {
			c.ID = uuid.New()
		}
	}

	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range clicks {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed to insert click: %w", err)
		}
	}
	return nil
}
