package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
)

func TestPostApiGamesIdParticipants_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	body, _ := json.Marshal(api.UpdateGameParticipationRequest{Status: api.Going})
	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPostApiGamesIdParticipants_InvalidBody(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader([]byte("{invalid")))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostApiGamesIdParticipants_InvalidStatus(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	createGame(t, querier, "g1", userID, sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true})

	body := []byte(`{"status":"maybe"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostApiGamesIdParticipants_GameNotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	body, _ := json.Marshal(api.UpdateGameParticipationRequest{Status: api.Going})
	r := httptest.NewRequest(http.MethodPost, "/api/games/missing/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPostApiGamesIdParticipants_NonOrganizerRestrictions(t *testing.T) {
	cases := []struct {
		name        string
		publishedAt sql.NullTime
		expectCode  int
	}{
		{name: "unpublished", publishedAt: sql.NullTime{Valid: false}, expectCode: http.StatusNotFound},
		{name: "future", publishedAt: sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true}, expectCode: http.StatusNotFound},
		{name: "published", publishedAt: sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true}, expectCode: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := dbtesting.SetupTestDB(t)
			defer sqlDB.Close()

			organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
			participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

			querier := db.New(sqlDB)
			srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

			createGame(t, querier, "g1", organizerID, tc.publishedAt)

			req := api.UpdateGameParticipationRequest{Status: api.Going}
			body, _ := json.Marshal(req)
			r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
			r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
			w := httptest.NewRecorder()

			srv.PostApiGamesIdParticipants(w, r, "g1")

			if w.Code != tc.expectCode {
				t.Fatalf("expected status %d, got %d", tc.expectCode, w.Code)
			}

			if tc.expectCode != http.StatusOK {
				return
			}

			var resp api.GameParticipation
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			status, err := resp.Status.AsParticipationStatusUpdate()
			if err != nil {
				t.Fatalf("failed to parse status: %v", err)
			}

			if status != api.Going {
				t.Fatalf("expected status %s, got %s", api.Going, status)
			}

			if resp.UserId != strconv.FormatInt(participantID, 10) || resp.GameId != "g1" {
				t.Fatalf("unexpected identifiers: game %s user %s", resp.GameId, resp.UserId)
			}

			var going sql.NullBool
			row := sqlDB.QueryRow(`select going from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
			if err := row.Scan(&going); err != nil {
				t.Fatalf("failed to load participant row: %v", err)
			}
			if !going.Valid || !going.Bool {
				t.Fatalf("expected going=true, got %+v", going)
			}
		})
	}
}

func TestPostApiGamesIdParticipants_OrganizerCanUpdateWhenUnpublished(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	createGame(t, querier, "g1", organizerID, sql.NullTime{Valid: false})

	body, _ := json.Marshal(api.UpdateGameParticipationRequest{Status: api.NotGoing})
	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp api.GameParticipation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	status, err := resp.Status.AsParticipationStatusUpdate()
	if err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}
	if status != api.NotGoing {
		t.Fatalf("expected status %s, got %s", api.NotGoing, status)
	}

	var going sql.NullBool
	row := sqlDB.QueryRow(`select going from game_participants where user_id = ? and game_id = ?`, organizerID, "g1")
	if err := row.Scan(&going); err != nil {
		t.Fatalf("failed to load participant row: %v", err)
	}
	if !going.Valid || going.Bool {
		t.Fatalf("expected going=false, got %+v", going)
	}
}

func createGame(t *testing.T, querier *db.Queries, id string, organizerID int64, publishedAt sql.NullTime) {
	t.Helper()

	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 id,
		OrganizerID:        organizerID,
		Name:               "Test Game",
		PublishedAt:        publishedAt,
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         -1,
		MaxWaitlistSize:    0,
		MaxGuestsPerPlayer: -1,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
}
