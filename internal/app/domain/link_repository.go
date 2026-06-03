package domain

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LinkRepository interface {
	Create(ctx context.Context, link *Link) (int, error)
	GetByShortCode(ctx context.Context, shortCode string) (*Link, error)
	List(ctx context.Context, limit, offset int) ([]*Link, int, error)
	Delete(ctx context.Context, shortCode string) error
	IncrementVisits(ctx context.Context, shortCode string) error
}

type linkRepository struct {
	pool *pgxpool.Pool
}

func NewLinkRepository(pool *pgxpool.Pool) LinkRepository {
	return &linkRepository{pool: pool}
}

func (r *linkRepository) Create(ctx context.Context, link *Link) (int, error) {
	const query = `
		INSERT INTO links (short_code, original_url, created_at, visits)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	link.CreatedAt = time.Now()

	err := r.pool.QueryRow(ctx, query,
		link.ShortCode,
		link.OriginalURL,
		link.CreatedAt,
		link.Visits,
	).Scan(&link.ID)

	if err != nil {
		return -1, err
	}
	return link.ID, nil
}

func (r *linkRepository) GetByShortCode(ctx context.Context, shortCode string) (*Link, error) {
	const query = `
		SELECT id, short_code, original_url, created_at, visits
		FROM links
		WHERE short_code = $1`

	var link Link
	err := r.pool.QueryRow(ctx, query, shortCode).Scan(
		&link.ID,
		&link.ShortCode,
		&link.OriginalURL,
		&link.CreatedAt,
		&link.Visits,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &link, nil
}

func (r *linkRepository) List(ctx context.Context, limit, offset int) ([]*Link, int, error) {
	const query = `
		SELECT id, short_code, original_url, created_at, visits
		FROM links
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var links []*Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.ID, &l.ShortCode, &l.OriginalURL, &l.CreatedAt, &l.Visits); err != nil {
			return nil, 0, err
		}
		links = append(links, &l)
	}

	var total int
	r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM links").Scan(&total)

	return links, total, nil
}

func (r *linkRepository) Delete(ctx context.Context, shortCode string) error {
	const query = `DELETE FROM links WHERE short_code = $1`
	result, err := r.pool.Exec(ctx, query, shortCode)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *linkRepository) IncrementVisits(ctx context.Context, shortCode string) error {
	const query = `
		UPDATE links 
		SET visits = visits + 1 
		WHERE short_code = $1`
	_, err := r.pool.Exec(ctx, query, shortCode)
	return err
}
