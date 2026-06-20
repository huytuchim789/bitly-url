package postgres

import (
	"context"
	"fmt"

	"bitly-url/internal/entity"

	"github.com/jackc/pgx/v5/pgxpool"
)

type URLPostgresRepo struct {
	db *pgxpool.Pool
}

func NewURLPostgresRepo(db *pgxpool.Pool) *URLPostgresRepo {
	return &URLPostgresRepo{db: db}
}

func (r *URLPostgresRepo) Save(ctx context.Context, url *entity.URL) error {
	query := `INSERT INTO urls (id, original, short, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, url.ID, url.Original, url.Short, url.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to save url: %w", err)
	}
	return nil
}

func (r *URLPostgresRepo) FindByID(ctx context.Context, id string) (*entity.URL, error) {
	query := `SELECT id, original, short, created_at FROM urls WHERE id = $1`
	row := r.db.QueryRow(ctx, query, id)

	var url entity.URL
	err := row.Scan(&url.ID, &url.Original, &url.Short, &url.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("url not found: %w", err)
	}
	return &url, nil
}

func (r *URLPostgresRepo) FindByShort(ctx context.Context, short string) (*entity.URL, error) {
	return r.FindByID(ctx, short)
}

func (r *URLPostgresRepo) FindAll(ctx context.Context) ([]*entity.URL, error) {
	query := `SELECT id, original, short, created_at FROM urls ORDER BY created_at DESC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list urls: %w", err)
	}
	defer rows.Close()

	var urls []*entity.URL
	for rows.Next() {
		var url entity.URL
		if err := rows.Scan(&url.ID, &url.Original, &url.Short, &url.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan url: %w", err)
		}
		urls = append(urls, &url)
	}
	return urls, nil
}

func (r *URLPostgresRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM urls WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete url: %w", err)
	}
	return nil
}
