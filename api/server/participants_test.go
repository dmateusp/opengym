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

	// Verify game_spots_left decreased by 3, waitlist remains unchanged
	var gameSpotsLeft, waitlistSpotsLeft int64
	row := sqlDB.QueryRow(`select game_spots_left, waitlist_spots_left from games where id = ?`, "gspots")
	if err := row.Scan(&gameSpotsLeft, &waitlistSpotsLeft); err != nil {
		t.Fatalf("failed to load game row: %v", err)
	}
	if gameSpotsLeft != 2 {
		t.Fatalf("expected game_spots_left=2, got %d", gameSpotsLeft)
	}
	if waitlistSpotsLeft != 2 {
		t.Fatalf("expected waitlist_spots_left=2, got %d", waitlistSpotsLeft)
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

	var gameSpotsLeft, waitlistSpotsLeft int64
	row := sqlDB.QueryRow(`select game_spots_left, waitlist_spots_left from games where id = ?`, "gfree")
	if err := row.Scan(&gameSpotsLeft, &waitlistSpotsLeft); err != nil {
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
