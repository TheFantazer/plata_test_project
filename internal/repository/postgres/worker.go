package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

func (r *Repository) RequeueProcessing(ctx context.Context) (int64, error) {
	const query = `
		UPDATE quote_updates
		SET status = 'pending',
		    price = NULL,
		    fetched_at = NULL,
		    error_message = NULL
		WHERE status = 'processing'
	`

	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("requeue processing quote updates: %w", err)
	}

	return tag.RowsAffected(), nil
}

func (r *Repository) ClaimPending(ctx context.Context) (domain.QuoteUpdate, bool, error) {
	const query = `
		WITH next_update AS (
			SELECT id
			FROM quote_updates
			WHERE status = 'pending'
			ORDER BY created_at, id
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE quote_updates AS q
		SET status = 'processing'
		FROM next_update
		WHERE q.id = next_update.id
		RETURNING q.id, q.base_currency, q.quote_currency
	`

	var update domain.QuoteUpdate
	var base, quote string

	err := r.pool.QueryRow(ctx, query).Scan(&update.ID, &base, &quote)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.QuoteUpdate{}, false, nil
	}
	if err != nil {
		return domain.QuoteUpdate{}, false, fmt.Errorf("claim pending quote update: %w", err)
	}

	update.Pair = domain.Pair{
		Base:  domain.Currency(base),
		Quote: domain.Currency(quote),
	}
	update.Status = domain.StatusProcessing

	return update, true, nil
}

func (r *Repository) Complete(ctx context.Context, id int64, price string, fetchedAt time.Time) error {
	const query = `
		UPDATE quote_updates
		SET status = 'completed',
		    price = $2,
		    fetched_at = $3,
		    error_message = NULL
		WHERE id = $1
		  AND status = 'processing'
	`

	tag, err := r.pool.Exec(ctx, query, id, price, fetchedAt)
	if err != nil {
		return fmt.Errorf("complete quote update %d: %w", id, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("complete quote update %d: expected one processing row", id)
	}

	return nil
}

func (r *Repository) Fail(ctx context.Context, id int64, message string) error {
	const query = `
		UPDATE quote_updates
		SET status = 'failed',
		    price = NULL,
		    fetched_at = NULL,
		    error_message = $2
		WHERE id = $1
		  AND status = 'processing'
	`

	tag, err := r.pool.Exec(ctx, query, id, message)
	if err != nil {
		return fmt.Errorf("fail quote update %d: %w", id, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("fail quote update %d: expected one processing row", id)
	}

	return nil
}
