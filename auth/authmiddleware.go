package auth

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/dmateusp/opengym/demo"
	"github.com/dmateusp/opengym/log"
	"github.com/golang-jwt/jwt/v5"
)

// Parse the JWT from cookies and add claim information to the context
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Unauthenticated path
		if r.URL.EscapedPath() != "/api/auth/me" && strings.HasPrefix(r.URL.EscapedPath(), "/api/auth/google") {
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.EscapedPath(), "/api/demo") {
			if !demo.GetDemoMode() {
				http.Error(w, "Demo mode is not enabled", http.StatusForbidden)
			}
			next.ServeHTTP(w, r)
			return
		}

		var (
			jwtCookieName    = JWTCookie
			issuer           = Issuer
			jwtSigningSecret = signingSecret.Value()
		)

		if demo.GetDemoMode() {
			jwtCookieName = demo.DemoJWTCookie
			issuer = demo.DemoIssuer
			jwtSigningSecret = demo.GetDemoSigningSecret()
		}

		jwtCookie, err := r.Cookie(jwtCookieName)
		if err != nil {
			log.FromCtx(r.Context()).InfoContext(
				r.Context(),
				"Failed to get JWT",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		jwtToken, err := jwt.ParseWithClaims(jwtCookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
			return []byte(jwtSigningSecret), nil
		}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
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
