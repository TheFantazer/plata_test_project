package frankfurter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

const maxResponseBytes = 64 << 10 // 64 kb

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(httpClient *http.Client, baseURL string) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("HTTP client is required")
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil ||
		parsedURL.Host == "" ||
		(parsedURL.Scheme != "http" && parsedURL.Scheme != "https") ||
		parsedURL.RawQuery != "" ||
		parsedURL.Fragment != "" ||
		parsedURL.User != nil {
		return nil, fmt.Errorf("invalid provider base URL %q", baseURL)
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
	}, nil
}

// StatusError позволяет воркеру отличить временный ответ API
// от ошибки, которую повторять бессмысленно.
type StatusError struct {
	StatusCode int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("rate provider returned HTTP %d", e.StatusCode)
}

func (e *StatusError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode >= http.StatusInternalServerError
}

func (c *Client) GetRate(ctx context.Context, pair domain.Pair) (string, error) {
	if err := pair.Validate(); err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf(
		"%s/v2/rate/%s/%s",
		c.baseURL,
		strings.ToLower(string(pair.Base)),
		strings.ToLower(string(pair.Quote)),
	)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create rate request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request rate: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", &StatusError{StatusCode: response.StatusCode}
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read rate response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return "", fmt.Errorf("rate response exceeds %d bytes", maxResponseBytes)
	}

	var result struct {
		Base  string      `json:"base"`
		Quote string      `json:"quote"`
		Rate  json.Number `json:"rate"`
	}

	decoder := json.NewDecoder(bytes.NewReader(body))

	if err := decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("decode rate response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return "", fmt.Errorf("rate response contains extra data")
	}

	if !strings.EqualFold(result.Base, string(pair.Base)) ||
		!strings.EqualFold(result.Quote, string(pair.Quote)) {
		return "", fmt.Errorf(
			"rate provider returned unexpected pair %s/%s",
			result.Base,
			result.Quote,
		)
	}

	rate, err := strconv.ParseFloat(result.Rate.String(), 64)
	if err != nil || math.IsInf(rate, 0) || math.IsNaN(rate) ||
		rate < 0.0000000001 || rate >= 10000000000 {
		return "", fmt.Errorf("rate provider returned invalid rate %q", result.Rate)
	}

	return result.Rate.String(), nil
}
