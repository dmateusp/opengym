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

func TestGetApiGamesIdParticipants_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	w := httptest.NewRecorder()

	srv.GetApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetApiGamesIdParticipants_GameNotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	r := httptest.NewRequest(http.MethodGet, "/api/games/missing/participants", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdParticipants(w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetApiGamesIdParticipants_NonOrganizerRestrictions(t *testing.T) {
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

			r := httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
			r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
			w := httptest.NewRecorder()

			srv.GetApiGamesIdParticipants(w, r, "g1")

			if w.Code != tc.expectCode {
				t.Fatalf("expected status %d, got %d", tc.expectCode, w.Code)
			}
		})
	}
}

func TestGetApiGamesIdParticipants_StatusComputation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	// Create a game with max 2 players
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "g1",
		OrganizerID:        organizerID,
		Name:               "Test Game",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         2,
		MaxWaitlistSize:    -1,
		MaxGuestsPerPlayer: -1,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Create 4 users with different participation statuses
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2 := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3 := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")
	user4 := dbtesting.UpsertTestUser(t, sqlDB, "user4@example.com")

	// User 1 & 2: going (should be in main list)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:    user1,
		GameID:    "g1",
		Going:     sql.NullBool{Bool: true, Valid: true},
		Confirmed: sql.NullBool{Bool: true, Valid: true},
	})
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:    user2,
		GameID:    "g1",
		Going:     sql.NullBool{Bool: true, Valid: true},
		Confirmed: sql.NullBool{Bool: true, Valid: true},
	})
	time.Sleep(10 * time.Millisecond)

	// User 3: going (should be waitlisted because max is 2)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:    user3,
		GameID:    "g1",
		Going:     sql.NullBool{Bool: true, Valid: true},
		Confirmed: sql.NullBool{Bool: true, Valid: true},
	})
	time.Sleep(10 * time.Millisecond)

	// User 4: not going
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:    user4,
		GameID:    "g1",
		Going:     sql.NullBool{Bool: false, Valid: true},
		Confirmed: sql.NullBool{Bool: true, Valid: true},
	})

	// Get participants list
	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var participants []api.ParticipantWithUser
	if err := json.NewDecoder(w.Body).Decode(&participants); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(participants) != 4 {
		t.Fatalf("expected 4 participants, got %d", len(participants))
	}

	// Verify the order (oldest update first) and statuses
	// User 1 (going) - updated first, so first in main list
	if participants[0].User.Id != strconv.FormatInt(user1, 10) {
		t.Errorf("expected first participant to be user1, got %s", participants[0].User.Id)
	}
	status1, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status1 != api.Going {
		t.Errorf("expected user1 status to be going, got %v", status1)
	}

	// User 2 (going) - updated second, so second in main list
	if participants[1].User.Id != strconv.FormatInt(user2, 10) {
		t.Errorf("expected second participant to be user2, got %s", participants[1].User.Id)
	}
	status2, _ := participants[1].Status.AsParticipationStatusUpdate()
	if status2 != api.Going {
		t.Errorf("expected user2 status to be going, got %v", status2)
	}

	// User 3 (waitlisted) - updated third, exceeds max of 2
	if participants[2].User.Id != strconv.FormatInt(user3, 10) {
		t.Errorf("expected third participant to be user3, got %s", participants[2].User.Id)
	}
	status3, err := participants[2].Status.AsParticipationStatus1()
	if err != nil || status3 != api.Waitlisted {
		t.Errorf("expected user3 status to be waitlisted, got %v, err %v", status3, err)
	}

	// User 4 (not_going) - updated last
	if participants[3].User.Id != strconv.FormatInt(user4, 10) {
		t.Errorf("expected fourth participant to be user4, got %s", participants[3].User.Id)
	}
	status, _ := participants[3].Status.AsParticipationStatusUpdate()
	if status != api.NotGoing {
		t.Errorf("expected user4 status to be not_going, got %v", status)
	}
}

func TestGetApiGamesIdParticipants_UnlimitedPlayers(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

	// Create a game with unlimited players
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "g1",
		OrganizerID:        organizerID,
		Name:               "Test Game",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         -1, // Unlimited
		MaxWaitlistSize:    0,
		MaxGuestsPerPlayer: -1,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Create 3 users going
	for i := 1; i <= 3; i++ {
		userID := dbtesting.UpsertTestUser(t, sqlDB, "user"+strconv.Itoa(i)+"@example.com")
		querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
			UserID:    userID,
			GameID:    "g1",
			Going:     sql.NullBool{Bool: true, Valid: true},
			Confirmed: sql.NullBool{Bool: true, Valid: true},
		})
		time.Sleep(10 * time.Millisecond)
	}

	// Get participants list
	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var participants []api.ParticipantWithUser
	if err := json.NewDecoder(w.Body).Decode(&participants); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(participants) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(participants))
	}

	// All should be "going" with unlimited players
	for i, p := range participants {
		status, err := p.Status.AsParticipationStatusUpdate()
		if err != nil {
			t.Errorf("participant %d: failed to parse status: %v", i, err)
		}
		if status != api.Going {
			t.Errorf("participant %d: expected status going, got %v", i, status)
		}
	}
}
