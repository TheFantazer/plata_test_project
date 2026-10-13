package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	MXN Currency = "MXN"
)

var ErrInvalidPair = errors.New("invalid currency pair")

type Pair struct {
	Base  Currency
	Quote Currency
}

func ParsePair(raw string) (Pair, error) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return Pair{}, fmt.Errorf("%w: expected format Curr1/Curr2", ErrInvalidPair)
	}

	pair := Pair{
		Base:  Currency(parts[0]),
		Quote: Currency(parts[1]),
	}
	if err := pair.Validate(); err != nil {
		return Pair{}, err
	}

	return pair, nil
}

func (p Pair) Validate() error {
	if !supportedCurrency(p.Base) || !supportedCurrency(p.Quote) {
		return fmt.Errorf(
			"%w: supported currencies are USD, EUR and MXN",
			ErrInvalidPair,
		)
	}
	if p.Base == p.Quote {
		return fmt.Errorf("%w: currencies must differ", ErrInvalidPair)
	}
	return nil
}

func (p Pair) String() string {
	return string(p.Base) + "/" + string(p.Quote)
}

func supportedCurrency(currency Currency) bool {
	switch currency {
	case USD, EUR, MXN:
		return true
	default:
		return false
	}
}

type UpdateStatus string

const (
	StatusPending    UpdateStatus = "pending"
	StatusProcessing UpdateStatus = "processing"
	StatusCompleted  UpdateStatus = "completed"
	StatusFailed     UpdateStatus = "failed"
)

type QuoteUpdate struct {
	ID           int64
	Pair         Pair
	Status       UpdateStatus
	Price        *string
	CreatedAt    time.Time
	FetchedAt    *time.Time
	ErrorMessage *string
}
