package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/TheFantazer/plata_test_project/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateOrGetActive(ctx context.Context, pair domain.Pair) (int64, error) {
	const insertQuery = `
		INSERT INTO quote_updates (base_currency, quote_currency)
		VALUES ($1, $2)
		ON CONFLICT (base_currency, quote_currency)
			WHERE status IN ('pending', 'processing')
		DO NOTHING 
		RETURNING id
	`

	const findActiveQuery = `
		SELECT id
		FROM quote_updates
		WHERE base_currency = $1
		  AND quote_currency = $2
		  AND status IN ('pending', 'processing')
	`

	const maxAttempts = 5

	for attempt := 0; attempt < maxAttempts; attempt++ {
		var id int64

		err := r.pool.QueryRow(
			ctx,
			insertQuery,
			pair.Base,
			pair.Quote,
		).Scan(&id)

		if err == nil {
			return id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("insert quote update: %w", err)
		}

		err = r.pool.QueryRow(
			ctx,
			findActiveQuery,
			pair.Base,
			pair.Quote,
		).Scan(&id)

		if err == nil {
			return id, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("find active quote update: %w", err)
		}

		if err := ctx.Err(); err != nil {
			return 0, err
		}
	}

	return 0, fmt.Errorf("create or find active quote update: exceeded %d attempts", maxAttempts)
}
