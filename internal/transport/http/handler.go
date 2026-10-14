package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	stdhttp "net/http"
	"strconv"
	"time"

	"github.com/TheFantazer/plata_test_project/internal/domain"
)

const maxRequestBytes = 4 << 10 //4 Kib

type QuoteService interface {
	RequestUpdate(ctx context.Context, pair string) (int64, error)
	GetUpdate(ctx context.Context, id int64) (domain.QuoteUpdate, error)
	GetLatest(ctx context.Context, pair string) (domain.QuoteUpdate, error)
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

type quoteUpdateResponse struct {
	ID           int64               `json:"id"`
	Pair         string              `json:"pair"`
	Status       domain.UpdateStatus `json:"status"`
	Price        *string             `json:"price,omitempty"`
	FetchedAt    *time.Time          `json:"fetched_at,omitempty"`
	ErrorMessage *string             `json:"error_message,omitempty"`
}

func (h *Handler) CreateQuoteUpdate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	r.Body = stdhttp.MaxBytesReader(w, r.Body, maxRequestBytes)
	var request createQuoteUpdateRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSONBodyError(w, err)
		return
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSONBodyError(w, err)
		return
	}

	id, err := h.service.RequestUpdate(r.Context(), request.Pair)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPair) {
			stdhttp.Error(w, err.Error(), stdhttp.StatusBadRequest)
			return
		}

		log.Printf("create quote update: %v", err)
		stdhttp.Error(w, "internal server error", stdhttp.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(stdhttp.StatusAccepted)
	_ = json.NewEncoder(w).Encode(createQuoteUpdateResponse{ID: id})
}

func writeJSONBodyError(w stdhttp.ResponseWriter, err error) {
	var sizeError *stdhttp.MaxBytesError
	if errors.As(err, &sizeError) {
		stdhttp.Error(w, "request body too large", stdhttp.StatusRequestEntityTooLarge)
		return
	}

	stdhttp.Error(w, "invalid JSON body", stdhttp.StatusBadRequest)
}

func (h *Handler) GetQuoteUpdate(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		stdhttp.Error(w, "invalid update ID", stdhttp.StatusBadRequest)
		return
	}

	update, err := h.service.GetUpdate(r.Context(), id)
	if err != nil {
		writeReadError(w, err, "get quote update")
		return
	}

	writeQuoteUpdate(w, update)
}

func (h *Handler) GetLatestQuote(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	update, err := h.service.GetLatest(r.Context(), r.URL.Query().Get("pair"))
	if err != nil {
		writeReadError(w, err, "get latest quote")
		return
	}

	writeQuoteUpdate(w, update)
}

func writeReadError(w stdhttp.ResponseWriter, err error, operation string) {
	switch {
	case errors.Is(err, domain.ErrInvalidPair):
		stdhttp.Error(w, err.Error(), stdhttp.StatusBadRequest)
	case errors.Is(err, domain.ErrQuoteNotFound):
		stdhttp.Error(w, "quote not found", stdhttp.StatusNotFound)
	default:
		log.Printf("%s: %v", operation, err)
		stdhttp.Error(w, "internal server error", stdhttp.StatusInternalServerError)
	}
}

func writeQuoteUpdate(w stdhttp.ResponseWriter, update domain.QuoteUpdate) {
	response := quoteUpdateResponse{
		ID:           update.ID,
		Pair:         update.Pair.String(),
		Status:       update.Status,
		Price:        update.Price,
		FetchedAt:    update.FetchedAt,
		ErrorMessage: update.ErrorMessage,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
