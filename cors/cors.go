package cors

import (
	"flag"
	"net/http"
	"strings"
)

var allowedOriginPrefix = flag.String("cors.allowed-origin-prefix", "http://localhost:", "CORS allowed origin prefix")

// CORSMiddleware adds CORS headers to allow requests from the frontend
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow localhost origins for development
		if origin != "" && strings.HasPrefix(origin, *allowedOriginPrefix) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		// Handle preflight requests - always respond to OPTIONS
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
