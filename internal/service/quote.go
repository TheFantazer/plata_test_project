package service

import (
	"context"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

type QuoteRepository interface {
	CreateOrGetActive(ctx context.Context, pair domain.Pair) (int64, error)
	GetByID(ctx context.Context, id int64) (domain.QuoteUpdate, error)
	GetLatestCompleted(ctx context.Context, pair domain.Pair) (domain.QuoteUpdate, error)
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

func (s *QuoteService) GetUpdate(ctx context.Context, id int64) (domain.QuoteUpdate, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *QuoteService) GetLatest(
	ctx context.Context,
	rawPair string,
) (domain.QuoteUpdate, error) {
	pair, err := domain.ParsePair(rawPair)
	if err != nil {
		return domain.QuoteUpdate{}, err
	}

	return s.repository.GetLatestCompleted(ctx, pair)
}
