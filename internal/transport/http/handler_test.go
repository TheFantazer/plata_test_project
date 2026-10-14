package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

type stubQuoteService struct {
	id    int64
	err   error
	pair  string
	calls int
}

func (s *stubQuoteService) RequestUpdate(_ context.Context, pair string) (int64, error) {
	s.calls++
	s.pair = pair
	return s.id, s.err
}

func (s *stubQuoteService) GetUpdate(_ context.Context, _ int64) (domain.QuoteUpdate, error) {
	return domain.QuoteUpdate{}, nil
}

func (s *stubQuoteService) GetLatest(_ context.Context, _ string) (domain.QuoteUpdate, error) {
	return domain.QuoteUpdate{}, nil
}

func TestCreateQuoteUpdate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantBody   string
		wantCalls  int
		wantPair   string
	}{
		{
			name:       "accepted update",
			body:       `{"pair":"EUR/MXN"}`,
			wantStatus: http.StatusAccepted,
			wantBody:   `{"id":42}` + "\n",
			wantCalls:  1,
			wantPair:   "EUR/MXN",
		},
		{
			name:       "invalid pair",
			body:       `{"pair":"EUR/EUR"}`,
			serviceErr: domain.ErrInvalidPair,
			wantStatus: http.StatusBadRequest,
			wantCalls:  1,
			wantPair:   "EUR/EUR",
		},
		{
			name:       "unknown field",
			body:       `{"pair":"EUR/MXN","extra":true}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "second JSON value",
			body:       `{"pair":"EUR/MXN"}{"pair":"USD/MXN"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "oversized body",
			body:       `{"pair":"EUR/MXN"}` + strings.Repeat(" ", maxRequestBytes),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "repository failure",
			body:       `{"pair":"EUR/MXN"}`,
			serviceErr: errors.New("database unavailable"),
			wantStatus: http.StatusInternalServerError,
			wantCalls:  1,
			wantPair:   "EUR/MXN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &stubQuoteService{id: 42, err: tt.serviceErr}
			request := httptest.NewRequest(http.MethodPost, "/v1/quote-updates", strings.NewReader(tt.body))
			response := httptest.NewRecorder()

			NewRouter(NewHandler(service)).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && response.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", response.Body.String(), tt.wantBody)
			}
			if service.calls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", service.calls, tt.wantCalls)
			}
			if service.pair != tt.wantPair {
				t.Errorf("pair = %q, want %q", service.pair, tt.wantPair)
			}
			if tt.wantStatus == http.StatusAccepted && response.Header().Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", response.Header().Get("Content-Type"))
			}
		})
	}
}
