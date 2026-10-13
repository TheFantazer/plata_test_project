package service

import (
	"context"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

type QuoteRepository interface {
	CreateOrGetActive(ctx context.Context, pair domain.Pair) (int64, error)
}

type QuoteService struct {
	repository QuoteRepository
}

func NewQuoteService(repository QuoteRepository) *QuoteService {
	return &QuoteService{repository: repository}
}

func (s *QuoteService) RequestUpdate(ctx context.Context, rawPair string) (int64, error) {
	pair, err := domain.ParsePair(rawPair)
	if err != nil {
		return 0, err
	}

	return s.repository.CreateOrGetActive(ctx, pair)
}
