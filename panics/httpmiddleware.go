package panics

import (
	"log/slog"
	"net/http"

	"github.com/dmateusp/opengym/log"
)

// Catches panics so they don't crash the http server.
// Logs the errors instead.
func PanicsCatcherMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger := log.FromCtx(r.Context())
				logger.Error("Panic recovered", slog.Any("error", err))
				w.WriteHeader(http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
