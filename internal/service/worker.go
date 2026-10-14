package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

type WorkerRepository interface {
	ClaimPending(ctx context.Context) (domain.QuoteUpdate, bool, error)
	Complete(ctx context.Context, id int64, price string, fetchedAt time.Time) error
	Fail(ctx context.Context, id int64, message string) error
}

type RateProvider interface {
	GetRate(ctx context.Context, pair domain.Pair) (string, error)
}

type Worker struct {
	repository WorkerRepository
	provider   RateProvider
	retryDelay time.Duration
}

func NewWorker(repository WorkerRepository, provider RateProvider) *Worker {
	return &Worker{
		repository: repository,
		provider:   provider,
		retryDelay: time.Second,
	}
}

const maxFetchAttempts = 3

func (w *Worker) fetchRateWithRetry(ctx context.Context, pair domain.Pair) (string, error) {
	for attempt := 1; attempt <= maxFetchAttempts; attempt++ {
		requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		price, err := w.provider.GetRate(requestCtx, pair)
		cancel()
		if err == nil {
			return price, nil
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		if attempt == maxFetchAttempts || !isRetryableProviderError(err) {
			return "", err
		}

		delay := w.retryDelay * time.Duration(1<<(attempt-1))
		log.Printf("rate request for %s failed (attempt %d/%d): %v; retrying in %s",
			pair, attempt, maxFetchAttempts, err, delay)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return "", ctx.Err()
		case <-timer.C:
		}
	}

	return "", errors.New("rate request exhausted retries")
}

func isRetryableProviderError(err error) bool {
	var retryable interface{ Retryable() bool }
	if errors.As(err, &retryable) {
		return retryable.Retryable()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return nil
		}

		update, found, err := w.repository.ClaimPending(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("claim quote update: %w", err)
		}

		if !found {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				continue
			}
		}

		price, fetchErr := w.fetchRateWithRetry(ctx, update.Pair)

		if ctx.Err() != nil {
			return nil
		}

		if fetchErr != nil {
			if err := w.repository.Fail(ctx, update.ID, fetchErr.Error()); err != nil {
				return fmt.Errorf("save failure for update %d: %w", update.ID, err)
			}

			log.Printf("quote update %d failed: %v", update.ID, fetchErr)
			continue
		}

		if err := w.repository.Complete(
			ctx,
			update.ID,
			price,
			time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("save quote update %d: %w", update.ID, err)
		}

		log.Printf("quote update %d completed: %s = %s", update.ID, update.Pair, price)
	}
}
