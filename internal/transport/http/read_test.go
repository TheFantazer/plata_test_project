package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

type readQuoteService struct {
	update      domain.QuoteUpdate
	err         error
	getByIDCall int
	latestCall  int
	id          int64
	pair        string
}

var _ QuoteService = (*readQuoteService)(nil)

func (s *readQuoteService) RequestUpdate(_ context.Context, _ string) (int64, error) {
	return 0, nil
}

func (s *readQuoteService) GetUpdate(_ context.Context, id int64) (domain.QuoteUpdate, error) {
	s.getByIDCall++
	s.id = id
	return s.update, s.err
}

func (s *readQuoteService) GetLatest(_ context.Context, pair string) (domain.QuoteUpdate, error) {
	s.latestCall++
	s.pair = pair
	return s.update, s.err
}

func TestGetQuoteUpdate(t *testing.T) {
	price := "20.5250000000"
	fetchedAt := time.Date(2026, 10, 14, 9, 20, 42, 0, time.UTC)
	pair := domain.Pair{Base: domain.EUR, Quote: domain.MXN}

	tests := []struct {
		name       string
		path       string
		update     domain.QuoteUpdate
		err        error
		wantStatus int
		wantBody   string
		wantCalls  int
	}{
		{
			name:       "completed update",
			path:       "/v1/quote-updates/42",
			update:     domain.QuoteUpdate{ID: 42, Pair: pair, Status: domain.StatusCompleted, Price: &price, FetchedAt: &fetchedAt},
			wantStatus: http.StatusOK,
			wantBody:   `{"id":42,"pair":"EUR/MXN","status":"completed","price":"20.5250000000","fetched_at":"2026-10-14T09:20:42Z"}` + "\n",
			wantCalls:  1,
		},
		{
			name:       "pending update has no price",
			path:       "/v1/quote-updates/42",
			update:     domain.QuoteUpdate{ID: 42, Pair: pair, Status: domain.StatusPending},
			wantStatus: http.StatusOK,
			wantBody:   `{"id":42,"pair":"EUR/MXN","status":"pending"}` + "\n",
			wantCalls:  1,
		},
		{
			name:       "invalid ID",
			path:       "/v1/quote-updates/abc",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid update ID"}` + "\n",
		},
		{
			name:       "missing update",
			path:       "/v1/quote-updates/42",
			err:        domain.ErrQuoteNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"quote not found"}` + "\n",
			wantCalls:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &readQuoteService{update: tt.update, err: tt.err}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			NewRouter(NewHandler(service)).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
			if service.getByIDCall != tt.wantCalls {
				t.Errorf("GetUpdate calls = %d, want %d", service.getByIDCall, tt.wantCalls)
			}
			if tt.wantCalls == 1 && service.id != 42 {
				t.Errorf("ID passed to service = %d, want 42", service.id)
			}
		})
	}
}

func TestGetLatestQuote(t *testing.T) {
	price := "20.5250000000"
	fetchedAt := time.Date(2026, 10, 14, 9, 20, 42, 0, time.UTC)
	completed := domain.QuoteUpdate{
		ID: 42, Pair: domain.Pair{Base: domain.EUR, Quote: domain.MXN},
		Status: domain.StatusCompleted, Price: &price, FetchedAt: &fetchedAt,
	}

	tests := []struct {
		name       string
		path       string
		err        error
		wantStatus int
		wantBody   string
		wantPair   string
	}{
		{
			name:       "last completed quote",
			path:       "/v1/quotes/latest?pair=EUR%2FMXN",
			wantStatus: http.StatusOK,
			wantBody:   `{"id":42,"pair":"EUR/MXN","status":"completed","price":"20.5250000000","fetched_at":"2026-10-14T09:20:42Z"}` + "\n",
			wantPair:   "EUR/MXN",
		},
		{
			name:       "no completed quote yet",
			path:       "/v1/quotes/latest?pair=EUR%2FMXN",
			err:        domain.ErrQuoteNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"quote not found"}` + "\n",
			wantPair:   "EUR/MXN",
		},
		{
			name:       "invalid pair",
			path:       "/v1/quotes/latest?pair=EUR%2FEUR",
			err:        domain.ErrInvalidPair,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":"invalid currency pair"}` + "\n",
			wantPair:   "EUR/EUR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &readQuoteService{update: completed, err: tt.err}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)

			NewRouter(NewHandler(service)).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
			if service.latestCall != 1 {
				t.Errorf("GetLatest calls = %d, want 1", service.latestCall)
			}
			if service.pair != tt.wantPair {
				t.Errorf("pair passed to service = %q, want %q", service.pair, tt.wantPair)
			}
		})
	}
}

func TestRouterErrorsJSON(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
		wantAllow  string
	}{
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":"route not found"}` + "\n",
		},
		{
			name:       "wrong method on create route",
			method:     http.MethodGet,
			path:       "/v1/quote-updates",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"error":"method not allowed"}` + "\n",
			wantAllow:  "POST",
		},
		{
			name:       "wrong method on read route",
			method:     http.MethodPost,
			path:       "/v1/quote-updates/42",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   `{"error":"method not allowed"}` + "\n",
			wantAllow:  "GET, HEAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			NewRouter(NewHandler(&readQuoteService{})).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if response.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
			if response.Header().Get("Allow") != tt.wantAllow {
				t.Errorf("Allow = %q, want %q", response.Header().Get("Allow"), tt.wantAllow)
			}
		})
	}
}
