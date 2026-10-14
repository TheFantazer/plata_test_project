package http

import stdhttp "net/http"

func NewRouter(handler *Handler) stdhttp.Handler {
	mux := stdhttp.NewServeMux()

	mux.HandleFunc("GET /healthz", healthHandler)

	registerRoute(mux, stdhttp.MethodPost, "/v1/quote-updates", handler.CreateQuoteUpdate)
	registerRoute(mux, stdhttp.MethodGet, "/v1/quote-updates/{id}", handler.GetQuoteUpdate)
	registerRoute(mux, stdhttp.MethodGet, "/v1/quotes/latest", handler.GetLatestQuote)

	mux.HandleFunc("/", func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		writeError(w, stdhttp.StatusNotFound, "route not found")
	})

	return mux
}

func healthHandler(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	w.WriteHeader(stdhttp.StatusOK)
}

func registerRoute(mux *stdhttp.ServeMux, method, path string, handler stdhttp.HandlerFunc) {
	mux.HandleFunc(method+" "+path, handler)
	mux.HandleFunc(path, func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.Header().Set("Allow", allowedMethods(method))
		writeError(w, stdhttp.StatusMethodNotAllowed, "method not allowed")
	})
}

func allowedMethods(method string) string {
	if method == stdhttp.MethodGet {
		return "GET, HEAD"
	}
	return method
}
