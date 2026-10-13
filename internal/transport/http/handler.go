package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	stdhttp "net/http"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

type QuoteService interface {
	RequestUpdate(ctx context.Context, pair string) (int64, error)
}

type Handler struct {
	service QuoteService
}

func NewHandler(service QuoteService) *Handler {
	return &Handler{service: service}
}

type createQuoteUpdateRequest struct {
	Pair string `json:"pair"`
}

type createQuoteUpdateResponse struct {
	ID int64 `json:"id"`
}

func (h *Handler) CreateQuoteUpdate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var request createQuoteUpdateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		stdhttp.Error(w, "invalid JSON body", stdhttp.StatusBadRequest)
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		stdhttp.Error(w, "invalid JSON body", stdhttp.StatusBadRequest)
		return
	}

	id, err := h.service.RequestUpdate(r.Context(), request.Pair)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPair) {
			stdhttp.Error(w, err.Error(), stdhttp.StatusBadRequest)
			return
		}

		stdhttp.Error(w, "internal server error", stdhttp.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(stdhttp.StatusAccepted)

	if err := json.NewEncoder(w).Encode(createQuoteUpdateResponse{ID: id}); err != nil {
		return
	}
}
