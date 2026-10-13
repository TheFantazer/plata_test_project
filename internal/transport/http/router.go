package http

import stdhttp "net/http"

func NewRouter(handler *Handler) stdhttp.Handler {
	mux := stdhttp.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusOK)
	})

	mux.HandleFunc("POST /v1/quote-updates", handler.CreateQuoteUpdate)

	return mux
}
