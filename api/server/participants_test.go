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

func TestPostApiGamesIdParticipants_ConfirmParticipation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

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
		UserID:      int64(user1),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(user2),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)

	// User 3: going (should be waitlisted because max is 2)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(user3),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)

	// User 4: not going
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(user4),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: false, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
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

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	userEarly := dbtesting.UpsertTestUser(t, sqlDB, "early@example.com")
	userLate := dbtesting.UpsertTestUser(t, sqlDB, "late@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

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

	// Fill both player slots before the organizer marks as going.
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(userEarly),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(userLate),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})

	// Organizer joins after the other two but should still be prioritized into the main list.
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(organizerID),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
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

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	going1 := dbtesting.UpsertTestUser(t, sqlDB, "going1@example.com")
	notGoing := dbtesting.UpsertTestUser(t, sqlDB, "notgoing@example.com")
	going2 := dbtesting.UpsertTestUser(t, sqlDB, "going2@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

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

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(going1),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(notGoing),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: false, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(going2),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
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

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	going1 := dbtesting.UpsertTestUser(t, sqlDB, "going1@example.com")
	going2 := dbtesting.UpsertTestUser(t, sqlDB, "going2@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

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

	// Organizer marks not going; should not consume a slot or change waitlist math.
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(organizerID),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: false, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(going1),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(going2),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
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

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	user1 := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2 := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3 := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator())

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
		MaxWaitlistSize:    0,
		MaxGuestsPerPlayer: -1,
	})
	if err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(user1),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(user2),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
	})
	time.Sleep(10 * time.Millisecond)
	querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:      int64(user3),
		GameID:      "g1",
		Going:       sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
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
			UserID:      int64(userID),
			GameID:      "g1",
			Going:       sql.NullBool{Bool: true, Valid: true},
			ConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
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
