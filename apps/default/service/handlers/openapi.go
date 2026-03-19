package handlers

import "net/http"

// EmbeddedSpecHandler serves an embedded OpenAPI document.
func EmbeddedSpecHandler(spec []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(spec)
	})
}
