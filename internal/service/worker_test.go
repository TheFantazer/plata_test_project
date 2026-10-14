package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/TheFantazer/plata_test_project/internal/domain"
	"github.com/TheFantazer/plata_test_project/internal/provider/frankfurter"
)

type rateResult struct {
	price string
	err   error
}

type sequenceRateProvider struct {
	results []rateResult
	calls   int
}

func (p *sequenceRateProvider) GetRate(_ context.Context, _ domain.Pair) (string, error) {
	result := p.results[p.calls]
	p.calls++
	return result.price, result.err
}

func TestFetchRateWithRetry(t *testing.T) {
	tests := []struct {
		name      string
		results   []rateResult
		wantPrice string
		wantCalls int
		wantError bool
	}{
		{
			name: "retries rate limit",
			results: []rateResult{
				{err: &frankfurter.StatusError{StatusCode: http.StatusTooManyRequests}},
				{price: "18.5"},
			},
			wantPrice: "18.5",
			wantCalls: 2,
		},
		{
			name: "retries server error",
			results: []rateResult{
				{err: &frankfurter.StatusError{StatusCode: http.StatusBadGateway}},
				{err: &frankfurter.StatusError{StatusCode: http.StatusServiceUnavailable}},
				{price: "18.5"},
			},
			wantPrice: "18.5",
			wantCalls: 3,
		},
		{
			name: "does not retry not found",
			results: []rateResult{
				{err: &frankfurter.StatusError{StatusCode: http.StatusNotFound}},
			},
			wantCalls: 1,
			wantError: true,
		},
		{
			name: "stops after three attempts",
			results: []rateResult{
				{err: &frankfurter.StatusError{StatusCode: http.StatusTooManyRequests}},
				{err: &frankfurter.StatusError{StatusCode: http.StatusTooManyRequests}},
				{err: &frankfurter.StatusError{StatusCode: http.StatusTooManyRequests}},
			},
			wantCalls: 3,
			wantError: true,
		},
		{
			name: "retries timeout",
			results: []rateResult{
				{err: context.DeadlineExceeded},
				{price: "18.5"},
			},
			wantPrice: "18.5",
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &sequenceRateProvider{results: tt.results}
			worker := NewWorker(nil, provider)
			worker.retryDelay = 0

			price, err := worker.fetchRateWithRetry(context.Background(), domain.Pair{
				Base: domain.EUR, Quote: domain.MXN,
			})
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %t", err, tt.wantError)
			}
			if price != tt.wantPrice {
				t.Errorf("price = %q, want %q", price, tt.wantPrice)
			}
			if provider.calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", provider.calls, tt.wantCalls)
			}
		})
	}
}

func TestFetchRateWithRetryStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := &cancellingRateProvider{cancel: cancel}
	worker := NewWorker(nil, provider)
	worker.retryDelay = 0

	_, err := worker.fetchRateWithRetry(ctx, domain.Pair{Base: domain.EUR, Quote: domain.MXN})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if provider.calls != 1 {
		t.Fatalf("calls = %d, want 1", provider.calls)
	}
}

type cancellingRateProvider struct {
	cancel context.CancelFunc
	calls  int
}

func (p *cancellingRateProvider) GetRate(_ context.Context, _ domain.Pair) (string, error) {
	p.calls++
	p.cancel()
	return "", &frankfurter.StatusError{StatusCode: http.StatusTooManyRequests}
}
