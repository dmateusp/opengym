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
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
)

func TestPostApiGamesIdParticipants_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
	staticClock := clock.StaticClock{Time: time.Now()}

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
	staticClock := clock.StaticClock{Time: time.Now()}

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
	staticClock := clock.StaticClock{Time: time.Now()}

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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

	staticClock := clock.StaticClock{Time: time.Now()}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := dbtesting.SetupTestDB(t)
			defer sqlDB.Close()

			organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
			participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

			querier := db.New(sqlDB)
			srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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

func TestPostApiGamesIdParticipants_ConfirmParticipation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Published game so non-organizer can interact
	createGame(t, querier, "g1", organizerID, sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true})

	confirm := api.True
	req := api.UpdateGameParticipationRequest{Status: api.Going, Confirmed: &confirm}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Confirmed_at should be set (non-null) and recent
	var confirmedAt sql.NullTime
	row := sqlDB.QueryRow(`select confirmed_at from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
	if err := row.Scan(&confirmedAt); err != nil {
		t.Fatalf("failed to load participant row: %v", err)
	}
	if !confirmedAt.Valid {
		t.Fatalf("expected confirmed_at to be set, got NULL")
	}
	if time.Since(confirmedAt.Time) > 5*time.Second {
		t.Fatalf("expected confirmed_at to be recent, got %v", confirmedAt.Time)
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
		MaxPlayers:         50,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      50,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}
}

func TestGetApiGamesIdParticipants_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
	staticClock := clock.StaticClock{Time: time.Now()}

	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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

	staticClock := clock.StaticClock{Time: time.Now()}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB := dbtesting.SetupTestDB(t)
			defer sqlDB.Close()

			organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
			participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

			querier := db.New(sqlDB)
			srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      2,
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
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user2),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// User 3: going (should be waitlisted because max is 2)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user3),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// User 4: not going
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user4),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: false, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
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

func TestGetApiGamesIdParticipants_OrganizerPriorityEvenWhenLate(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	userEarly := dbtesting.UpsertTestUser(t, sqlDB, "early@example.com")
	userLate := dbtesting.UpsertTestUser(t, sqlDB, "late@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      2,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Fill both player slots before the organizer marks as going.
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(userEarly),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(userLate),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Organizer joins after the other two but should still be prioritized into the main list.

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(organizerID),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

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

	if participants[0].User.Id != strconv.FormatInt(organizerID, 10) {
		t.Fatalf("expected organizer first, got user %s", participants[0].User.Id)
	}
	status, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status != api.Going {
		t.Fatalf("organizer should be going, got %v", status)
	}

	if participants[1].User.Id != strconv.FormatInt(userEarly, 10) {
		t.Fatalf("expected early participant second, got user %s", participants[1].User.Id)
	}
	status, _ = participants[1].Status.AsParticipationStatusUpdate()
	if status != api.Going {
		t.Fatalf("early participant should be going, got %v", status)
	}

	if participants[2].User.Id != strconv.FormatInt(userLate, 10) {
		t.Fatalf("expected late participant third, got user %s", participants[2].User.Id)
	}
	waitlistStatus, err := participants[2].Status.AsParticipationStatus1()
	if err != nil || waitlistStatus != api.Waitlisted {
		t.Fatalf("late participant should be waitlisted, got %v (err %v)", waitlistStatus, err)
	}
}

func TestGetApiGamesIdParticipants_NonGoingDoesNotConsumeSlot(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	going1 := dbtesting.UpsertTestUser(t, sqlDB, "going1@example.com")
	notGoing := dbtesting.UpsertTestUser(t, sqlDB, "notgoing@example.com")
	going2 := dbtesting.UpsertTestUser(t, sqlDB, "going2@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      2,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(going1),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(notGoing),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: false, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(going2),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

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

	status0, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status0 != api.Going {
		t.Fatalf("first participant should be going, got %v", status0)
	}
	status1, _ := participants[1].Status.AsParticipationStatusUpdate()
	if status1 != api.NotGoing {
		t.Fatalf("second participant should be not_going, got %v", status1)
	}
	status2, _ := participants[2].Status.AsParticipationStatusUpdate()
	if status2 != api.Going {
		t.Fatalf("third participant should still be going (not waitlisted), got %v", status2)
	}
}

func TestGetApiGamesIdParticipants_OrganizerNotGoingDoesNotBlockSlots(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	going1 := dbtesting.UpsertTestUser(t, sqlDB, "going1@example.com")
	going2 := dbtesting.UpsertTestUser(t, sqlDB, "going2@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      2,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Organizer marks not going; should not consume a slot or change waitlist math.
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(organizerID),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: false, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(going1),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(going2),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

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

	// Organizer is first by priority but not going.
	status0, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status0 != api.NotGoing {
		t.Fatalf("organizer should be not_going, got %v", status0)
	}

	// Both other users should occupy the two main slots (not waitlisted).
	status1, _ := participants[1].Status.AsParticipationStatusUpdate()
	if status1 != api.Going {
		t.Fatalf("second participant should be going, got %v", status1)
	}
	status2, _ := participants[2].Status.AsParticipationStatusUpdate()
	if status2 != api.Going {
		t.Fatalf("third participant should be going, got %v", status2)
	}
}

func TestGetApiGamesIdParticipants_NoWaitlistCapacityStillReturnsWaitlisted(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2 := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3 := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
		MaxPlayers:         1,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      1,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user2),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user3),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

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

	mainStatus, _ := participants[0].Status.AsParticipationStatusUpdate()
	if mainStatus != api.Going {
		t.Fatalf("first participant should be going (main slot), got %v", mainStatus)
	}
	wl1, err := participants[1].Status.AsParticipationStatus1()
	if err != nil || wl1 != api.Waitlisted {
		t.Fatalf("second participant should be waitlisted despite zero waitlist size, got %v (err %v)", wl1, err)
	}
	wl2, err := participants[2].Status.AsParticipationStatus1()
	if err != nil || wl2 != api.Waitlisted {
		t.Fatalf("third participant should be waitlisted despite zero waitlist size, got %v (err %v)", wl2, err)
	}
}

func TestGetApiGamesIdParticipants_LargeCapacityAllGoing(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with high capacity so everyone fits
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
		MaxPlayers:         100,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      100,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Create 3 users going
	for i := 1; i <= 3; i++ {
		userID := dbtesting.UpsertTestUser(t, sqlDB, "user"+strconv.Itoa(i)+"@example.com")
		querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
			GoingUpdatedAt: staticClock.Now(),
			UserID:         int64(userID),
			GameID:         "g1",
			Going:          sql.NullBool{Bool: true, Valid: true},
			ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
		})

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

	// All should be "going"
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
func TestGetApiGamesIdParticipants_GuestsCountTowardsMaxPlayers(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2 := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3 := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with max 4 players
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
		MaxPlayers:         4,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      4,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// User 1: going with 2 guests (3 total)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
		Guests:         sql.NullInt64{Int64: 2, Valid: true},
	})

	// User 2: going with 1 guest (2 total, should fit in main list since 3+2=5 > 4)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user2),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
		Guests:         sql.NullInt64{Int64: 1, Valid: true},
	})

	// User 3: going with no guests (should be waitlisted because 3+2+1=6 > 4)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user3),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

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

	// User 1: going (3 slots with 2 guests)
	if participants[0].User.Id != strconv.FormatInt(user1, 10) {
		t.Errorf("expected first participant to be user1, got %s", participants[0].User.Id)
	}
	status1, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status1 != api.Going {
		t.Errorf("expected user1 status to be going, got %v", status1)
	}

	// User 2: waitlisted (2 slots with 1 guest, would exceed max of 4: 3+2 > 4)
	if participants[1].User.Id != strconv.FormatInt(user2, 10) {
		t.Errorf("expected second participant to be user2, got %s", participants[1].User.Id)
	}
	status2, err := participants[1].Status.AsParticipationStatus1()
	if err != nil || status2 != api.Waitlisted {
		t.Errorf("expected user2 status to be waitlisted, got %v (err %v)", status2, err)
	}

	// User 3: going (1 slot with no guests: 3+1=4 <= 4, even though user2 is waitlisted)
	if participants[2].User.Id != strconv.FormatInt(user3, 10) {
		t.Errorf("expected third participant to be user3, got %s", participants[2].User.Id)
	}
	status3, _ := participants[2].Status.AsParticipationStatusUpdate()
	if status3 != api.Going {
		t.Errorf("expected user3 status to be going, got %v", status3)
	}
}

func TestPostApiGamesIdParticipants_GroupTooLargeForGoingListGoesToWaitlist(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with 5 max players, 3 waitlist spots
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
		MaxPlayers:         5,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      5,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Fill 3 spots in the going list (leaving 2 spots free)
	// Use the POST endpoint to properly update the spots counters
	user1Guests := int(2)
	req1 := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &user1Guests}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body1))
	r1 = r1.WithContext(auth.WithAuthInfo(r1.Context(), auth.AuthInfo{UserId: int(user1)}))
	w1 := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(w1, r1, "g1")
	if w1.Code != http.StatusOK {
		t.Fatalf("failed to add user1: status %d, body: %s", w1.Code, w1.Body.String())
	}

	// Now try to join with 2 guests (3 people total)
	// Should go to waitlist since group size (3) > remaining going spots (2)
	guests := int(2)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify the game spots were updated correctly
	game, err := querier.GameGetById(context.Background(), "g1")
	if err != nil {
		t.Fatalf("failed to get game: %v", err)
	}

	// Going list should still have 2 spots left (unchanged)
	if game.GameSpotsLeft != 2 {
		t.Errorf("expected game spots left to be 2, got %d", game.GameSpotsLeft)
	}

	// Verify the participant is in the database with correct guests
	var guestsCol sql.NullInt64
	row := sqlDB.QueryRow(`select guests from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
	if err := row.Scan(&guestsCol); err != nil {
		t.Fatalf("failed to load participant row: %v", err)
	}
	if !guestsCol.Valid || guestsCol.Int64 != 2 {
		t.Fatalf("expected guests=2, got %+v", guestsCol)
	}

	// Now fetch the participants list and verify the new participant is waitlisted
	r = httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()

	srv.GetApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var participants []api.ParticipantWithUser
	if err := json.NewDecoder(w.Body).Decode(&participants); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}

	// First participant (user1) should be going
	if participants[0].User.Id != strconv.FormatInt(user1, 10) {
		t.Errorf("expected first participant to be user1, got %s", participants[0].User.Id)
	}
	status1, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status1 != api.Going {
		t.Errorf("expected user1 status to be going, got %v", status1)
	}
	if participants[0].Guests != 2 {
		t.Errorf("expected user1 to have 2 guests, got %d", participants[0].Guests)
	}

	// Second participant (new participant) should be waitlisted
	if participants[1].User.Id != strconv.FormatInt(participantID, 10) {
		t.Errorf("expected second participant to be participant, got %s", participants[1].User.Id)
	}
	status2, err := participants[1].Status.AsParticipationStatus1()
	if err != nil || status2 != api.Waitlisted {
		t.Errorf("expected participant status to be waitlisted, got %v (err %v)", status2, err)
	}
	if participants[1].Guests != 2 {
		t.Errorf("expected participant to have 2 guests, got %d", participants[1].Guests)
	}
}

func TestPostApiGamesIdParticipants_GuestsExceedsLimit(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with max 2 guests per player
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
		MaxPlayers:         10,
		MaxGuestsPerPlayer: 2,
		GameSpotsLeft:      10,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Try to join with 3 guests (exceeds limit of 2)
	guests := int(3)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	// Verify no participant was created
	var guests_col sql.NullInt64
	row := sqlDB.QueryRow(`select guests from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
	if err := row.Scan(&guests_col); err == nil {
		t.Fatalf("expected no row to be created, but found guests=%v", guests_col)
	}
}

func TestPostApiGamesIdParticipants_GuestsWithinLimit(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with max 2 guests per player
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
		MaxPlayers:         10,
		MaxGuestsPerPlayer: 2,
		GameSpotsLeft:      10,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Join with 2 guests (at limit)
	guests := int(2)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
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
		t.Fatalf("expected status going, got %s", status)
	}

	// Verify guests were saved
	var guests_col sql.NullInt64
	row := sqlDB.QueryRow(`select guests from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
	if err := row.Scan(&guests_col); err != nil {
		t.Fatalf("failed to load participant row: %v", err)
	}
	if !guests_col.Valid || guests_col.Int64 != 2 {
		t.Fatalf("expected guests=2, got %+v", guests_col)
	}
}

func TestPostApiGamesIdParticipants_UpdateGuestsChangesQueueOrder(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2 := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

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
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      2,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// User 1: going first at time T0
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: time.Now().Add(-2 * time.Second), // T0 - earlier
		UserID:         int64(user1),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// User 2: going second at time T1 (slightly later)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: time.Now().Add(-1 * time.Second), // T1 - later
		UserID:         int64(user2),
		GameID:         "g1",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Now user1 updates their participation to add a guest (should trigger going_updated_at to update)
	// This moves user1 to the back of the queue, so they should now be waitlisted
	guests := int(1)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(user1)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Get participants list
	// After update, order should be:
	// 1. User2 (going_updated_at earlier) - fills 1 slot, stays going
	// 2. User1 (going_updated_at newer due to guest update) - would fill 2 slots, but only 1 left, so waitlisted
	r = httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()

	srv.GetApiGamesIdParticipants(w, r, "g1")

	var participants []api.ParticipantWithUser
	if err := json.NewDecoder(w.Body).Decode(&participants); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(participants))
	}

	// User 2: should still be first (older going_updated_at) and going
	if participants[0].User.Id != strconv.FormatInt(user2, 10) {
		t.Errorf("expected first participant to be user2, got %s", participants[0].User.Id)
	}
	status2, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status2 != api.Going {
		t.Errorf("expected user2 status to be going, got %v", status2)
	}

	// User 1: should now be second (newer going_updated_at due to guest update) and waitlisted
	if participants[1].User.Id != strconv.FormatInt(user1, 10) {
		t.Errorf("expected second participant to be user1, got %s", participants[1].User.Id)
	}
	status1, err := participants[1].Status.AsParticipationStatus1()
	if err != nil || status1 != api.Waitlisted {
		t.Errorf("expected user1 status to be waitlisted, got %v (err %v)", status1, err)
	}
}

func TestPostApiGamesIdParticipants_GoingReducesGameSpotsLeft(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with 5 spots and 2 waitlist spots
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gspots",
		OrganizerID:        organizerID,
		Name:               "Spots Test",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         5,
		MaxGuestsPerPlayer: 2,
		GameSpotsLeft:      5,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	guests := int(2)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/gspots/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "gspots")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Verify game_spots_left decreased by 3
	var gameSpotsLeft int64
	row := sqlDB.QueryRow(`select game_spots_left from games where id = ?`, "gspots")
	if err := row.Scan(&gameSpotsLeft); err != nil {
		t.Fatalf("failed to load game row: %v", err)
	}
	if gameSpotsLeft != 2 {
		t.Fatalf("expected game_spots_left=2, got %d", gameSpotsLeft)
	}
}

func TestPostApiGamesIdParticipants_NotGoingFreesGameSpotsLeft(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	userA := dbtesting.UpsertTestUser(t, sqlDB, "a@example.com")
	userB := dbtesting.UpsertTestUser(t, sqlDB, "b@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with 3 spots, fill them with A(1 guest) and B
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gfree",
		OrganizerID:        organizerID,
		Name:               "Spots Free Test",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         3,
		MaxGuestsPerPlayer: 2,
		GameSpotsLeft:      3,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// A joins with no guests (uses 1 spot) via server
	initReqA := api.UpdateGameParticipationRequest{Status: api.Going}
	initBodyA, _ := json.Marshal(initReqA)
	rA := httptest.NewRequest(http.MethodPost, "/api/games/gfree/participants", bytes.NewReader(initBodyA))
	rA = rA.WithContext(auth.WithAuthInfo(rA.Context(), auth.AuthInfo{UserId: int(userA)}))
	wA := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wA, rA, "gfree")
	if wA.Code != http.StatusOK {
		t.Fatalf("expected status %d for A join, got %d", http.StatusOK, wA.Code)
	}

	// B joins (uses 1 spot) via server
	initReqB := api.UpdateGameParticipationRequest{Status: api.Going}
	initBodyB, _ := json.Marshal(initReqB)
	rB := httptest.NewRequest(http.MethodPost, "/api/games/gfree/participants", bytes.NewReader(initBodyB))
	rB = rB.WithContext(auth.WithAuthInfo(rB.Context(), auth.AuthInfo{UserId: int(userB)}))
	wB := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wB, rB, "gfree")
	if wB.Code != http.StatusOK {
		t.Fatalf("expected status %d for B join, got %d", http.StatusOK, wB.Code)
	}

	// A changes to not_going; this should free 1 spot in the main list
	statusReq := api.NotGoing
	req := api.UpdateGameParticipationRequest{Status: statusReq}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/gfree/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userA)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "gfree")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var gameSpotsLeft int64
	row := sqlDB.QueryRow(`select game_spots_left from games where id = ?`, "gfree")
	if err := row.Scan(&gameSpotsLeft); err != nil {
		t.Fatalf("failed to load game row: %v", err)
	}
	if gameSpotsLeft != 2 {
		t.Fatalf("expected game_spots_left to increase to 2, got %d", gameSpotsLeft)
	}
}

func TestOrganizerLateJoinFullGamePushesLastToWaitlist(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	userEarly := dbtesting.UpsertTestUser(t, sqlDB, "early@example.com")
	userLate := dbtesting.UpsertTestUser(t, sqlDB, "late@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Full main (2), waitlist capacity=1
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gorgpush",
		OrganizerID:        organizerID,
		Name:               "Organizer Push",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         2,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      0,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Fill main list
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now().Add(-2 * time.Second),
		UserID:         int64(userEarly),
		GameID:         "gorgpush",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now().Add(-1 * time.Second),
		UserID:         int64(userLate),
		GameID:         "gorgpush",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Organizer joins late via POST
	req := api.UpdateGameParticipationRequest{Status: api.Going}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/games/gorgpush/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(w, r, "gorgpush")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Listing should show organizer prioritized and late user pushed to waitlist
	r = httptest.NewRequest(http.MethodGet, "/api/games/gorgpush/participants", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.GetApiGamesIdParticipants(w, r, "gorgpush")
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
	if participants[0].User.Id != strconv.FormatInt(organizerID, 10) {
		t.Fatalf("expected organizer first, got %s", participants[0].User.Id)
	}
	status, _ := participants[0].Status.AsParticipationStatusUpdate()
	if status != api.Going {
		t.Fatalf("organizer should be going, got %v", status)
	}
	status, _ = participants[1].Status.AsParticipationStatusUpdate()
	if status != api.Going {
		t.Fatalf("second should be going, got %v", status)
	}
	_, err = participants[2].Status.AsParticipationStatus1()
	if err != nil {
		t.Fatalf("third should be waitlisted, got parse error: %v", err)
	}
}

func TestPostApiGamesIdParticipants_NotGoingClearsGuests(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with guests allowed
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
		MaxPlayers:         10,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      10,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// First, join with 3 guests
	guests := int(3)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify guests were saved
	var guestsCol sql.NullInt64
	row := sqlDB.QueryRow(`select guests from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
	if err := row.Scan(&guestsCol); err != nil {
		t.Fatalf("failed to load participant row: %v", err)
	}
	if !guestsCol.Valid || guestsCol.Int64 != 3 {
		t.Fatalf("expected guests=3, got %+v", guestsCol)
	}

	// Now change to not going (with guests still in request, but they should be cleared)
	guestsInRequest := int(2)
	req = api.UpdateGameParticipationRequest{Status: api.NotGoing, Guests: &guestsInRequest}
	body, _ = json.Marshal(req)

	r = httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w = httptest.NewRecorder()

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
		t.Fatalf("expected status not_going, got %s", status)
	}

	// Verify guests were cleared (set to 0)
	row = sqlDB.QueryRow(`select guests from game_participants where user_id = ? and game_id = ?`, participantID, "g1")
	if err := row.Scan(&guestsCol); err != nil {
		t.Fatalf("failed to load participant row: %v", err)
	}
	if !guestsCol.Valid || guestsCol.Int64 != 0 {
		t.Fatalf("expected guests to be cleared (0), got %+v", guestsCol)
	}
}

func TestPostApiGamesIdParticipants_NonOrganizerFitsInMainList(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create game with 2 max players and 2 spots left
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gmainlist",
		OrganizerID:        organizerID,
		Name:               "Main List Test",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         2,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      2,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Non-organizer joins with 1 guest (group of 2)
	guests := int(1)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/gmainlist/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "gmainlist")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameParticipation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Response should indicate "going"
	status, err := resp.Status.AsParticipationStatusUpdate()
	if err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}
	if status != api.Going {
		t.Fatalf("expected status going, got %v", status)
	}

	// Game spots left should be decremented
	game, err := querier.GameGetById(context.Background(), "gmainlist")
	if err != nil {
		t.Fatalf("failed to get game: %v", err)
	}
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected game spots left to be 0, got %d", game.GameSpotsLeft)
	}
}

func TestPostApiGamesIdParticipants_NonOrganizerWaitlistedWhenFull(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create game with 1 max player and 0 spots left
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gfull",
		OrganizerID:        organizerID,
		Name:               "Full Game Test",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         1,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      0,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// First user already in the game
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now().Add(-time.Second),
		UserID:         int64(user1),
		GameID:         "gfull",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Second user (non-organizer) tries to join - should be waitlisted
	req := api.UpdateGameParticipationRequest{Status: api.Going}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/gfull/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "gfull")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameParticipation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Response should indicate "waitlisted"
	_, err = resp.Status.AsParticipationStatus1()
	if err != nil {
		t.Fatalf("expected status waitlisted, got parse error: %v", err)
	}

	// Game spots should remain 0
	game, err := querier.GameGetById(context.Background(), "gfull")
	if err != nil {
		t.Fatalf("failed to get game: %v", err)
	}
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected game spots left to remain 0, got %d", game.GameSpotsLeft)
	}
}

func TestPostApiGamesIdParticipants_OrganizerAlwaysGetsGoingStatus(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create game with 1 max player (full) and 0 spots left
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gorgfull",
		OrganizerID:        organizerID,
		Name:               "Organizer Full Test",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         1,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      0,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// First user already in the game
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now().Add(-time.Second),
		UserID:         int64(user1),
		GameID:         "gorgfull",
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Organizer tries to join - should always get "going" even when full
	req := api.UpdateGameParticipationRequest{Status: api.Going}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/gorgfull/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "gorgfull")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameParticipation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Response should indicate "going" for organizer
	status, err := resp.Status.AsParticipationStatusUpdate()
	if err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}
	if status != api.Going {
		t.Fatalf("expected organizer status going, got %v", status)
	}

	// Game spots should remain 0 (organizer joining without space doesn't decrement)
	game, err := querier.GameGetById(context.Background(), "gorgfull")
	if err != nil {
		t.Fatalf("failed to get game: %v", err)
	}
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected game spots left to remain 0 when organizer joins full game, got %d", game.GameSpotsLeft)
	}
}

func TestPostApiGamesIdParticipants_OrganizerWithGuestsFitsInMainList(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create game with 3 max players and 3 spots left
	_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "gorgmain",
		OrganizerID:        organizerID,
		Name:               "Organizer Main Test",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         3,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      3,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// Organizer joins with 2 guests (group of 3)
	guests := int(2)
	req := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPost, "/api/games/gorgmain/participants", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.PostApiGamesIdParticipants(w, r, "gorgmain")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameParticipation
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Response should indicate "going"
	status, err := resp.Status.AsParticipationStatusUpdate()
	if err != nil {
		t.Fatalf("failed to parse status: %v", err)
	}
	if status != api.Going {
		t.Fatalf("expected status going, got %v", status)
	}

	// Game spots left should be decremented by 3
	game, err := querier.GameGetById(context.Background(), "gorgmain")
	if err != nil {
		t.Fatalf("failed to get game: %v", err)
	}
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected game spots left to be 0 (3-3), got %d", game.GameSpotsLeft)
	}
}
func TestPostApiGamesIdParticipants_WaitlistedPromotedWhenMainListLeaves(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2 := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3 := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with max 3 players
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
		MaxPlayers:         3,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      3,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// User1 joins with 2 guests (group of 3, fills main list completely)
	guests1 := int(2)
	req1 := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guests1}
	body1, _ := json.Marshal(req1)
	r1 := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body1))
	r1 = r1.WithContext(auth.WithAuthInfo(r1.Context(), auth.AuthInfo{UserId: int(user1)}))
	w1 := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(w1, r1, "g1")
	if w1.Code != http.StatusOK {
		t.Fatalf("user1 failed to join: status %d", w1.Code)
	}

	// Verify GameSpotsLeft is 0 after user1 joins
	game, err := querier.GameGetById(context.Background(), "g1")
	if err != nil {
		t.Fatalf("failed to get game after user1 join: %v", err)
	}
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected game spots left to be 0 after user1 (3 people) joins max 3, got %d", game.GameSpotsLeft)
	}

	// User2 joins (should go to waitlist since no spots left)
	req2 := api.UpdateGameParticipationRequest{Status: api.Going}
	body2, _ := json.Marshal(req2)
	r2 := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body2))
	r2 = r2.WithContext(auth.WithAuthInfo(r2.Context(), auth.AuthInfo{UserId: int(user2)}))
	w2 := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(w2, r2, "g1")
	if w2.Code != http.StatusOK {
		t.Fatalf("user2 failed to join: status %d", w2.Code)
	}

	// Verify user2 is waitlisted
	var resp2 api.GameParticipation
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode user2 response: %v", err)
	}
	status2, err := resp2.Status.AsParticipationStatus1()
	if err != nil || status2 != api.Waitlisted {
		t.Fatalf("expected user2 to be waitlisted, got status: %v (err: %v)", status2, err)
	}

	// User3 joins (should also go to waitlist since user2 is already there)
	req3 := api.UpdateGameParticipationRequest{Status: api.Going}
	body3, _ := json.Marshal(req3)
	r3 := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(body3))
	r3 = r3.WithContext(auth.WithAuthInfo(r3.Context(), auth.AuthInfo{UserId: int(user3)}))
	w3 := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(w3, r3, "g1")
	if w3.Code != http.StatusOK {
		t.Fatalf("user3 failed to join: status %d", w3.Code)
	}

	// Verify user3 is waitlisted
	var resp3 api.GameParticipation
	if err := json.NewDecoder(w3.Body).Decode(&resp3); err != nil {
		t.Fatalf("failed to decode user3 response: %v", err)
	}
	status3, err := resp3.Status.AsParticipationStatus1()
	if err != nil || status3 != api.Waitlisted {
		t.Fatalf("expected user3 to be waitlisted, got status: %v (err: %v)", status3, err)
	}

	// GameSpotsLeft should still be 0
	game, err = querier.GameGetById(context.Background(), "g1")
	if err != nil {
		t.Fatalf("failed to get game after users join: %v", err)
	}
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected game spots left to be 0 after waitlist filled, got %d", game.GameSpotsLeft)
	}

	// Now user1 leaves the main list
	reqLeave := api.UpdateGameParticipationRequest{Status: api.NotGoing}
	bodyLeave, _ := json.Marshal(reqLeave)
	rLeave := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyLeave))
	rLeave = rLeave.WithContext(auth.WithAuthInfo(rLeave.Context(), auth.AuthInfo{UserId: int(user1)}))
	wLeave := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wLeave, rLeave, "g1")
	if wLeave.Code != http.StatusOK {
		t.Fatalf("user1 failed to leave: status %d", wLeave.Code)
	}

	// The critical check: GameSpotsLeft should be 1 (freed 3 spots, consumed 2 by promotions: user2(1) + user3(1))
	// Not 3 (which would be wrong - it would mean user2 and user3 weren't promoted)
	// Not 0 (which would be wrong - it would mean all 3 freed spots weren't available)
	game, err = querier.GameGetById(context.Background(), "g1")
	if err != nil {
		t.Fatalf("failed to get game after user1 leaves: %v", err)
	}
	if game.GameSpotsLeft != 1 {
		t.Fatalf("BUG: expected game spots left to be 1 after user1 leaves (3 freed - 2 for user2&user3 promotion), got %d", game.GameSpotsLeft)
	}

	// Verify by checking GET participants that user2 and user3 are now in the main list
	rGet := httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	rGet = rGet.WithContext(auth.WithAuthInfo(rGet.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	wGet := httptest.NewRecorder()
	srv.GetApiGamesIdParticipants(wGet, rGet, "g1")

	if wGet.Code != http.StatusOK {
		t.Fatalf("failed to get participants: status %d", wGet.Code)
	}

	var participants []api.ParticipantWithUser
	if err := json.NewDecoder(wGet.Body).Decode(&participants); err != nil {
		t.Fatalf("failed to decode participants: %v", err)
	}

	if len(participants) != 3 {
		t.Fatalf("expected 3 participants, got %d", len(participants))
	}

	// Find user2 in the list and verify they're in the main list now
	var user2Entry *api.ParticipantWithUser
	for i := range participants {
		if participants[i].User.Id == strconv.FormatInt(user2, 10) {
			user2Entry = &participants[i]
			break
		}
	}

	if user2Entry == nil {
		t.Fatalf("user2 not found in participants list")
	}

	user2Status, err := user2Entry.Status.AsParticipationStatusUpdate()
	if err != nil || user2Status != api.Going {
		t.Fatalf("expected user2 to be in main list (going), got status: %v (err: %v)", user2Status, err)
	}

	// Verify user3 is also in the main list now (promoted because there's capacity)
	var user3Entry *api.ParticipantWithUser
	for i := range participants {
		if participants[i].User.Id == strconv.FormatInt(user3, 10) {
			user3Entry = &participants[i]
			break
		}
	}

	if user3Entry == nil {
		t.Fatalf("user3 not found in participants list")
	}

	user3Status, err := user3Entry.Status.AsParticipationStatusUpdate()
	if err != nil || user3Status != api.Going {
		t.Fatalf("expected user3 to be promoted to main list (going), got status: %v (err: %v)", user3Status, err)
	}
}
func TestPostApiGamesIdParticipants_OrganizerJoinsLateDemotesNonFittingGroup(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	userA := dbtesting.UpsertTestUser(t, sqlDB, "userA@example.com")
	userB := dbtesting.UpsertTestUser(t, sqlDB, "userB@example.com")
	userC := dbtesting.UpsertTestUser(t, sqlDB, "userC@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game with max 5 players
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
		MaxPlayers:         5,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      5,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	// User A joins with 2 guests (3 people total, 2 spots left)
	guestsA := int(2)
	reqA := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guestsA}
	bodyA, _ := json.Marshal(reqA)
	rA := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyA))
	rA = rA.WithContext(auth.WithAuthInfo(rA.Context(), auth.AuthInfo{UserId: int(userA)}))
	wA := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wA, rA, "g1")
	if wA.Code != http.StatusOK {
		t.Fatalf("userA failed to join: status %d", wA.Code)
	}

	game, _ := querier.GameGetById(context.Background(), "g1")
	if game.GameSpotsLeft != 2 {
		t.Fatalf("expected spots left 2 after userA, got %d", game.GameSpotsLeft)
	}

	// User B joins with 1 guest (2 people total, fills remaining capacity)
	guestsB := int(1)
	reqB := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guestsB}
	bodyB, _ := json.Marshal(reqB)
	rB := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyB))
	rB = rB.WithContext(auth.WithAuthInfo(rB.Context(), auth.AuthInfo{UserId: int(userB)}))
	wB := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wB, rB, "g1")
	if wB.Code != http.StatusOK {
		t.Fatalf("userB failed to join: status %d", wB.Code)
	}

	game, _ = querier.GameGetById(context.Background(), "g1")
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected spots left 0 after userB, got %d", game.GameSpotsLeft)
	}

	// User C joins (1 person, no guests) - goes to waitlist (no space)
	reqC := api.UpdateGameParticipationRequest{Status: api.Going}
	bodyC, _ := json.Marshal(reqC)
	rC := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyC))
	rC = rC.WithContext(auth.WithAuthInfo(rC.Context(), auth.AuthInfo{UserId: int(userC)}))
	wC := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wC, rC, "g1")
	if wC.Code != http.StatusOK {
		t.Fatalf("userC failed to join: status %d", wC.Code)
	}

	var respC api.GameParticipation
	json.NewDecoder(wC.Body).Decode(&respC)
	statusC, _ := respC.Status.AsParticipationStatus1()
	if statusC != api.Waitlisted {
		t.Fatalf("expected userC to be waitlisted, got going")
	}

	// Now organizer joins with 1 guest (2 people total)
	// The organizer has priority, so they should go to main list
	// This should demote userB (2 people) to the waitlist
	// and keep userC on the waitlist
	guestsOrg := int(1)
	reqOrg := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guestsOrg}
	bodyOrg, _ := json.Marshal(reqOrg)
	rOrg := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyOrg))
	rOrg = rOrg.WithContext(auth.WithAuthInfo(rOrg.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	wOrg := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wOrg, rOrg, "g1")
	if wOrg.Code != http.StatusOK {
		t.Fatalf("organizer failed to join: status %d", wOrg.Code)
	}

	var respOrg api.GameParticipation
	json.NewDecoder(wOrg.Body).Decode(&respOrg)
	statusOrg, _ := respOrg.Status.AsParticipationStatusUpdate()
	if statusOrg != api.Going {
		t.Fatalf("expected organizer to go to main list (going), but got status: %v", statusOrg)
	}

	// Check GameSpotsLeft - should still be 0 since:
	// - Organizer (2) + UserA (3) = 5 total (full)
	game, _ = querier.GameGetById(context.Background(), "g1")
	if game.GameSpotsLeft != 0 {
		t.Fatalf("expected spots left 0 after organizer joins, got %d", game.GameSpotsLeft)
	}

	// Verify the main list now consists of: Organizer + UserA
	// And userB is demoted to waitlist
	rGet := httptest.NewRequest(http.MethodGet, "/api/games/g1/participants", nil)
	rGet = rGet.WithContext(auth.WithAuthInfo(rGet.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	wGet := httptest.NewRecorder()
	srv.GetApiGamesIdParticipants(wGet, rGet, "g1")

	var participants []api.ParticipantWithUser
	json.NewDecoder(wGet.Body).Decode(&participants)

	// Find statuses
	statuses := make(map[int64]api.ParticipationStatus)
	for _, p := range participants {
		uid, _ := strconv.ParseInt(p.User.Id, 10, 64)
		statuses[uid] = p.Status
	}

	// Organizer should be going
	orgStatus, _ := statuses[organizerID].AsParticipationStatusUpdate()
	if orgStatus != api.Going {
		t.Fatalf("expected organizer to be going")
	}

	// UserA should be going
	aStatus, _ := statuses[userA].AsParticipationStatusUpdate()
	if aStatus != api.Going {
		t.Fatalf("expected userA to be going, got status: %v", aStatus)
	}

	// UserB should be on waitlist (demoted because organizer took a spot)
	bStatus, _ := statuses[userB].AsParticipationStatus1()
	if bStatus != api.Waitlisted {
		t.Fatalf("expected userB to be waitlisted (demoted by organizer priority), got status: going")
	}

	// UserC should be on waitlist
	cStatus, _ := statuses[userC].AsParticipationStatus1()
	if cStatus != api.Waitlisted {
		t.Fatalf("expected userC to be on waitlist")
	}
}

func TestPostApiGamesIdParticipants_OrganizerJoinsLateRecalculatesGameSpots(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	userA := dbtesting.UpsertTestUser(t, sqlDB, "userA@example.com")
	userB := dbtesting.UpsertTestUser(t, sqlDB, "userB@example.com")
	userC := dbtesting.UpsertTestUser(t, sqlDB, "userC@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create game with max 5 players
	querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:                 "g1",
		OrganizerID:        organizerID,
		Name:               "Test Game",
		PublishedAt:        sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
		Description:        sql.NullString{},
		TotalPriceCents:    0,
		Location:           sql.NullString{},
		StartsAt:           sql.NullTime{},
		DurationMinutes:    60,
		MaxPlayers:         5,
		MaxGuestsPerPlayer: 5,
		GameSpotsLeft:      5,
	})

	// UserA joins with 2 guests (3 people total) → GameSpotsLeft = 2
	guestsA := int(2)
	reqA := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guestsA}
	bodyA, _ := json.Marshal(reqA)
	rA := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyA))
	rA = rA.WithContext(auth.WithAuthInfo(rA.Context(), auth.AuthInfo{UserId: int(userA)}))
	wA := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wA, rA, "g1")

	game, _ := querier.GameGetById(context.Background(), "g1")
	if game.GameSpotsLeft != 2 {
		t.Fatalf("expected GameSpotsLeft=2 after userA, got %d", game.GameSpotsLeft)
	}

	// UserB joins with 2 guests (3 people total)
	// 3 > 2 (GameSpotsLeft), so UserB goes to waitlist
	guestsB := int(2)
	reqB := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guestsB}
	bodyB, _ := json.Marshal(reqB)
	rB := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyB))
	rB = rB.WithContext(auth.WithAuthInfo(rB.Context(), auth.AuthInfo{UserId: int(userB)}))
	wB := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wB, rB, "g1")

	var respB api.GameParticipation
	json.NewDecoder(wB.Body).Decode(&respB)
	statusB, _ := respB.Status.AsParticipationStatus1()
	if statusB != api.Waitlisted {
		t.Fatalf("expected userB on waitlist since 3 > 2 GameSpotsLeft")
	}

	game, _ = querier.GameGetById(context.Background(), "g1")
	if game.GameSpotsLeft != 2 {
		t.Fatalf("expected GameSpotsLeft=2 still (no change), got %d", game.GameSpotsLeft)
	}

	// UserC tries to join (1 person) with GameSpotsLeft=2
	// UserC should fit in main list
	reqC := api.UpdateGameParticipationRequest{Status: api.Going}
	bodyC, _ := json.Marshal(reqC)
	rC := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyC))
	rC = rC.WithContext(auth.WithAuthInfo(rC.Context(), auth.AuthInfo{UserId: int(userC)}))
	wC := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wC, rC, "g1")

	var respC api.GameParticipation
	json.NewDecoder(wC.Body).Decode(&respC)
	statusC, _ := respC.Status.AsParticipationStatusUpdate()
	if statusC != api.Going {
		t.Fatalf("expected userC to fit in main list")
	}

	game, _ = querier.GameGetById(context.Background(), "g1")
	// UserA (3) + UserC (1) = 4, so GameSpotsLeft should be 1
	if game.GameSpotsLeft != 1 {
		t.Fatalf("expected GameSpotsLeft=1 after userC joins, got %d", game.GameSpotsLeft)
	}

	// NOW: Organizer joins with 1 guest (2 people)
	// GameSpotsLeft = 1, but Organizer needs 2 spots
	// However, Organizer has priority, so they get in
	// After Organizer joins: Org (2) + UserA (3) = 5 (full, at capacity)
	// UserC stays in main list, UserB stays on waitlist
	// GameSpotsLeft should be recalculated to 0
	guestsOrg := int(1)
	reqOrg := api.UpdateGameParticipationRequest{Status: api.Going, Guests: &guestsOrg}
	bodyOrg, _ := json.Marshal(reqOrg)
	rOrg := httptest.NewRequest(http.MethodPost, "/api/games/g1/participants", bytes.NewReader(bodyOrg))
	rOrg = rOrg.WithContext(auth.WithAuthInfo(rOrg.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	wOrg := httptest.NewRecorder()
	srv.PostApiGamesIdParticipants(wOrg, rOrg, "g1")

	var respOrg api.GameParticipation
	json.NewDecoder(wOrg.Body).Decode(&respOrg)
	statusOrg, _ := respOrg.Status.AsParticipationStatusUpdate()
	if statusOrg != api.Going {
		t.Fatalf("expected organizer to be marked as going")
	}

	game, _ = querier.GameGetById(context.Background(), "g1")
	// CRITICAL CHECK: GameSpotsLeft should be 0 because:
	// Main list: Org (2) + UserA (3) = 5, at capacity
	if game.GameSpotsLeft != 0 {
		t.Fatalf("BUG: GameSpotsLeft should be 0 after organizer joins (Org 2 + UserA 3 + UserC 1 = 6? wait that's >5). "+
			"Actually: Org (2) + UserA (3) = 5, so GameSpotsLeft=0. But got %d", game.GameSpotsLeft)
	}
}
