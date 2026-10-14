package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

func (r *Repository) GetByID(ctx context.Context, id int64) (domain.QuoteUpdate, error) {
	const query = `
		SELECT id, base_currency, quote_currency, status,
		       price::text, created_at, fetched_at, error_message
		FROM quote_updates
		WHERE id = $1
	`

	return scanQuoteUpdate(r.pool.QueryRow(ctx, query, id))
}

func (r *Repository) GetLatestCompleted(ctx context.Context, pair domain.Pair) (domain.QuoteUpdate, error) {
	const query = `
		SELECT id, base_currency, quote_currency, status,
		       price::text, created_at, fetched_at, error_message
		FROM quote_updates
		WHERE base_currency = $1
		  AND quote_currency = $2
		  AND status = 'completed'
		ORDER BY fetched_at DESC, id DESC
		LIMIT 1
	`

	return scanQuoteUpdate(r.pool.QueryRow(ctx, query, pair.Base, pair.Quote))
}

func scanQuoteUpdate(row pgx.Row) (domain.QuoteUpdate, error) {
	var update domain.QuoteUpdate
	var base, quote, status string

	err := row.Scan(
		&update.ID,
		&base,
		&quote,
		&status,
		&update.Price,
		&update.CreatedAt,
		&update.FetchedAt,
		&update.ErrorMessage,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.QuoteUpdate{}, domain.ErrQuoteNotFound
	}
	if err != nil {
		return domain.QuoteUpdate{}, fmt.Errorf("read quote update: %w", err)
	}

	update.Pair = domain.Pair{
		Base:  domain.Currency(base),
		Quote: domain.Currency(quote),
	}
	update.Status = domain.UpdateStatus(status)

	return update, nil
}
