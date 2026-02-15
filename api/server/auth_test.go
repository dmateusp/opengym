package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
)

func TestPostApiAuthLogout_Success(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a request with an authenticated user
	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))

	// Set an existing JWT cookie to simulate an authenticated session
	r.AddCookie(&http.Cookie{
		Name:     auth.JWTCookie,
		Value:    "test-jwt-token",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	w := httptest.NewRecorder()
	srv.PostApiAuthLogout(w, r)

	// Verify the response status is 204 No Content
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify the JWT cookie was cleared
	cookies := w.Result().Cookies()
	jwtCookieCleared := false
	for _, cookie := range cookies {
		if cookie.Name == auth.JWTCookie {
			if cookie.MaxAge == -1 && cookie.Value == "" {
				jwtCookieCleared = true
			} else {
				t.Errorf("Expected JWT cookie to be cleared (MaxAge=-1, Value=empty), got MaxAge=%d, Value=%s", cookie.MaxAge, cookie.Value)
			}
			break
		}
	}
	if !jwtCookieCleared {
		t.Error("Expected JWT cookie to be present and cleared, but it was not found or not properly cleared")
	}
}

func TestPostApiAuthLogout_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a request without authentication
	r := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)

	w := httptest.NewRecorder()
	srv.PostApiAuthLogout(w, r)

	// Verify the response status is 401 Unauthorized
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	// Verify the response body contains the error message
	if w.Body.String() != "Unauthorized\n" {
		t.Errorf("Expected error message 'Unauthorized', got %s", w.Body.String())
	}
}
