package frankfurter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

func TestGetRate(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		wantPrice  string
		wantStatus int
		wantError  bool
	}{
		{
			name:      "preserves decimal rate",
			status:    http.StatusOK,
			body:      `{"base":"EUR","quote":"MXN","rate":18.1234567890}`,
			wantPrice: "18.1234567890",
		},
		{
			name:       "rate limit",
			status:     http.StatusTooManyRequests,
			wantStatus: http.StatusTooManyRequests,
			wantError:  true,
		},
		{
			name:      "wrong pair",
			status:    http.StatusOK,
			body:      `{"base":"USD","quote":"MXN","rate":18.5}`,
			wantError: true,
		},
		{
			name:      "nonpositive rate",
			status:    http.StatusOK,
			body:      `{"base":"EUR","quote":"MXN","rate":0}`,
			wantError: true,
		},
		{
			name:      "oversized response",
			status:    http.StatusOK,
			body:      strings.Repeat("x", maxResponseBytes+1),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/v2/rate/eur/mxn" {
					t.Errorf("request = %s %s, want GET /v2/rate/eur/mxn", r.Method, r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(server.Client(), server.URL)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			price, err := client.GetRate(context.Background(), domain.Pair{Base: domain.EUR, Quote: domain.MXN})
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %t", err, tt.wantError)
			}
			if price != tt.wantPrice {
				t.Errorf("price = %q, want %q", price, tt.wantPrice)
			}
			if tt.wantStatus != 0 {
				var statusErr *StatusError
				if !errors.As(err, &statusErr) || statusErr.StatusCode != tt.wantStatus {
					t.Errorf("error = %v, want StatusError with status %d", err, tt.wantStatus)
				}
			}
		})
	}
}
