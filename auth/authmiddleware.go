package auth

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/dmateusp/opengym/log"
	"github.com/golang-jwt/jwt/v5"
)

// Parse the JWT from cookies and add claim information to the context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Unauthenticated path
		if strings.HasPrefix(r.URL.Path, "/api/auth") {
			next.ServeHTTP(w, r)
			return
		}

		jwtCookie, err := r.Cookie(JTWCookie)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		jwtToken, err := jwt.ParseWithClaims(jwtCookie.Value, jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
			return []byte(*signingSecret), nil
		}, jwt.WithIssuer(Issuer), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
		if err != nil {
			log.FromCtx(r.Context()).InfoContext(
				r.Context(),
				"Failed to parse JWT with claims",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sub, err := jwtToken.Claims.GetSubject()
		if err != nil {
			log.FromCtx(r.Context()).InfoContext(
				r.Context(),
				"Failed to get subject from JWT claims",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userId, err := strconv.Atoi(sub)
		if err != nil {
			log.FromCtx(r.Context()).InfoContext(
				r.Context(),
				"Failed to convert subject to user ID",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithAuthInfo(
			r.Context(),
			AuthInfo{UserId: userId},
		)))

	})
}
