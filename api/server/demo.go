package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/demo"
	"github.com/golang-jwt/jwt/v5"
)

func (s *server) GetApiDemoUsers(w http.ResponseWriter, r *http.Request) {
	dbUsers, err := s.querier.ListDemoUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	users := make([]api.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		var user api.User
		user.FromDb(dbUser.User)
		users[i] = user
	}
	json.NewEncoder(w).Encode(users)
}
func (s *server) PostApiDemoUsersUserIdImpersonate(w http.ResponseWriter, r *http.Request, userId string) {
	parsedUserId, err := strconv.Atoi(userId)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse user id: %s", err.Error()), http.StatusBadRequest)
		return
	}
	dbUser, err := s.querier.UserGetById(r.Context(), int64(parsedUserId))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get user: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if !dbUser.User.IsDemo {
		http.Error(w, "user was not found or it was not a demo user", http.StatusNotFound)
		return
	}

	now := time.Now()

	jwtToken := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    demo.DemoIssuer,
			Subject:   userId,
			ExpiresAt: jwt.NewNumericDate(now.Add(4 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	)
	signedJwt, err := jwtToken.SignedString([]byte(demo.GetDemoSigningSecret()))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign jwt token: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     demo.DemoJWTCookie,
			Value:    signedJwt,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int((4 * time.Hour).Seconds()),
		},
	)

	var user api.User
	user.FromDb(dbUser.User)
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
