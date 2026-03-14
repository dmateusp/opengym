package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
	"github.com/oapi-codegen/nullable"
)

func TestGetApiGamesIdReimbursements_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/reimbursements", nil)
	w := httptest.NewRecorder()

	srv.GetApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetApiGamesIdReimbursements_GameNotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	r := httptest.NewRequest(http.MethodGet, "/api/games/missing/reimbursements", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdReimbursements(w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetApiGamesIdReimbursements_ForbiddenForNonOrganizer(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	otherID := dbtesting.UpsertTestUser(t, sqlDB, "other@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{})

	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/reimbursements", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(otherID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestGetApiGamesIdReimbursements_OrganizerSeesParticipants(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{})
	reimbursedAt := staticClock.Now().Add(-10 * time.Minute)
	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 participantID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now(),
		ConfirmedAt:            sql.NullTime{},
		Guests:                 sql.NullInt64{},
		ReimbursementReference: sql.NullString{String: "A1b2", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert participant: %v", err)
	}
	if _, err := sqlDB.Exec(`update game_participants set reimbursed_at = ? where game_id = ? and user_id = ?`,
		reimbursedAt, "g1", participantID); err != nil {
		t.Fatalf("failed to set reimbursed_at: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/reimbursements", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var entries []api.GameReimbursementEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Participant.Id != strconv.FormatInt(participantID, 10) {
		t.Fatalf("expected participant id %d, got %s", participantID, entries[0].Participant.Id)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{4}$`).MatchString(entries[0].ReimbursementReference) {
		t.Fatalf("expected reimbursement reference to be a 4-character alphanumeric value, got %q", entries[0].ReimbursementReference)
	}
	if !entries[0].ReimbursedAt.IsSpecified() || entries[0].ReimbursedAt.IsNull() {
		t.Fatalf("expected reimbursed_at to be set, got %+v", entries[0].ReimbursedAt)
	}
	if !entries[0].ReimbursedAt.MustGet().Equal(reimbursedAt) {
		t.Fatalf("expected reimbursed_at %v, got %v", reimbursedAt, entries[0].ReimbursedAt.MustGet())
	}
	if entries[0].AmountOwedCents != 0 {
		t.Fatalf("expected amount_owed_cents to be 0 for free game, got %d", entries[0].AmountOwedCents)
	}
}

func TestGetApiGamesIdReimbursements_NotGoingParticipantsExcluded(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	goingID := dbtesting.UpsertTestUser(t, sqlDB, "going@example.com")
	notGoingID := dbtesting.UpsertTestUser(t, sqlDB, "notgoing@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{})
	for _, uid := range []int64{goingID, notGoingID} {
		going := uid == goingID
		if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
			UserID:                 uid,
			GameID:                 "g1",
			Going:                  sql.NullBool{Valid: true, Bool: going},
			GoingUpdatedAt:         staticClock.Now(),
			ReimbursementReference: sql.NullString{String: map[bool]string{true: "C3d4", false: "E5f6"}[going], Valid: true},
		}); err != nil {
			t.Fatalf("failed to insert participant: %v", err)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/reimbursements", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var entries []api.GameReimbursementEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (only going), got %d", len(entries))
	}
	if entries[0].Participant.Id != strconv.FormatInt(goingID, 10) {
		t.Fatalf("expected going participant %d, got %s", goingID, entries[0].Participant.Id)
	}
}

func TestGetApiGamesIdReimbursements_AmountOwedExcludesWaitlistAndIncludesGuests(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	p1ID := dbtesting.UpsertTestUser(t, sqlDB, "p1@example.com")
	p2ID := dbtesting.UpsertTestUser(t, sqlDB, "p2@example.com")
	p3WaitlistedID := dbtesting.UpsertTestUser(t, sqlDB, "p3@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{})
	if _, err := sqlDB.Exec(`update games set total_price_cents = ?, max_players = ?, game_spots_left = ? where id = ?`, 1000, 3, 3, "g1"); err != nil {
		t.Fatalf("failed to update game pricing/capacity: %v", err)
	}

	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 p1ID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now().Add(1 * time.Minute),
		Guests:                 sql.NullInt64{Valid: true, Int64: 1},
		ReimbursementReference: sql.NullString{String: "A1b2", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert participant 1: %v", err)
	}

	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 p2ID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now().Add(2 * time.Minute),
		Guests:                 sql.NullInt64{Valid: true, Int64: 0},
		ReimbursementReference: sql.NullString{String: "C3d4", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert participant 2: %v", err)
	}

	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 p3WaitlistedID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now().Add(3 * time.Minute),
		Guests:                 sql.NullInt64{Valid: true, Int64: 0},
		ReimbursementReference: sql.NullString{String: "E5f6", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert waitlisted participant: %v", err)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/games/g1/reimbursements", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.GetApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var entries []api.GameReimbursementEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (waitlisted excluded), got %d", len(entries))
	}

	amountByParticipant := map[string]int64{}
	for _, entry := range entries {
		amountByParticipant[entry.Participant.Id] = entry.AmountOwedCents
	}

	if _, found := amountByParticipant[strconv.FormatInt(p3WaitlistedID, 10)]; found {
		t.Fatalf("waitlisted participant should not be included in reimbursements")
	}

	if got := amountByParticipant[strconv.FormatInt(p1ID, 10)]; got != 667 {
		t.Fatalf("expected participant 1 amount owed to be 667, got %d", got)
	}
	if got := amountByParticipant[strconv.FormatInt(p2ID, 10)]; got != 334 {
		t.Fatalf("expected participant 2 amount owed to be 334, got %d", got)
	}
}

func TestPutApiGamesIdReimbursements_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	body := []byte(`{"reimbursed_at":null}`)
	r := httptest.NewRequest(http.MethodPut, "/api/games/g1/reimbursements", bytes.NewReader(body))
	w := httptest.NewRecorder()

	srv.PutApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPutApiGamesIdReimbursements_GameNotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	userID := dbtesting.UpsertTestUser(t, sqlDB, "user@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	body := []byte(`{"reimbursed_at":null}`)
	r := httptest.NewRequest(http.MethodPut, "/api/games/missing/reimbursements", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.PutApiGamesIdReimbursements(w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPutApiGamesIdReimbursements_OrganizerUpdatesParticipant(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{Valid: false})
	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 participantID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now(),
		ConfirmedAt:            sql.NullTime{},
		Guests:                 sql.NullInt64{},
		ReimbursementReference: sql.NullString{String: "G7h8", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert participant: %v", err)
	}

	receivedAt := staticClock.Now().Add(-10 * time.Minute)
	req := api.UpdateReimbursementRequest{}
	if err := req.FromUpdateReimbursementRequest0(api.UpdateReimbursementRequest0{
		ParticipantId:           strconv.FormatInt(participantID, 10),
		ReimbursementReceivedAt: nullable.NewNullableWithValue(receivedAt),
	}); err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPut, "/api/games/g1/reimbursements", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()

	srv.PutApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.ReimbursementRecord
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ParticipantId != strconv.FormatInt(participantID, 10) {
		t.Fatalf("expected participant id %d, got %s", participantID, resp.ParticipantId)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{4}$`).MatchString(resp.ReimbursementReference) {
		t.Fatalf("expected reimbursement reference to be a 4-character alphanumeric value, got %q", resp.ReimbursementReference)
	}

	stored := sql.NullTime{}
	if err := sqlDB.QueryRow(`select reimbursement_received_at from game_participants where game_id = ? and user_id = ?`, "g1", participantID).Scan(&stored); err != nil {
		t.Fatalf("failed to query participant: %v", err)
	}
	if !stored.Valid || !stored.Time.Equal(receivedAt) {
		t.Fatalf("expected reimbursement_received_at %v, got %+v", receivedAt, stored)
	}
}

func TestPutApiGamesIdReimbursements_ParticipantUpdatesSelf(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true})
	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 participantID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now(),
		ConfirmedAt:            sql.NullTime{},
		Guests:                 sql.NullInt64{},
		ReimbursementReference: sql.NullString{String: "J9k0", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert participant: %v", err)
	}

	reimbursedAt := staticClock.Now().Add(-5 * time.Minute)
	req := api.UpdateReimbursementRequest{}
	if err := req.FromUpdateReimbursementRequest1(api.UpdateReimbursementRequest1{
		ReimbursedAt: nullable.NewNullableWithValue(reimbursedAt),
	}); err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPut, "/api/games/g1/reimbursements", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PutApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	stored := sql.NullTime{}
	if err := sqlDB.QueryRow(`select reimbursed_at from game_participants where game_id = ? and user_id = ?`, "g1", participantID).Scan(&stored); err != nil {
		t.Fatalf("failed to query participant: %v", err)
	}
	if !stored.Valid || !stored.Time.Equal(reimbursedAt) {
		t.Fatalf("expected reimbursed_at %v, got %+v", reimbursedAt, stored)
	}
}

func TestPutApiGamesIdReimbursements_ParticipantCannotSetParticipantId(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	participantID := dbtesting.UpsertTestUser(t, sqlDB, "participant@example.com")
	otherParticipantID := dbtesting.UpsertTestUser(t, sqlDB, "other@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true})
	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 participantID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now(),
		ConfirmedAt:            sql.NullTime{},
		Guests:                 sql.NullInt64{},
		ReimbursementReference: sql.NullString{String: "L1m2", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert participant: %v", err)
	}
	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:                 otherParticipantID,
		GameID:                 "g1",
		Going:                  sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt:         staticClock.Now(),
		ConfirmedAt:            sql.NullTime{},
		Guests:                 sql.NullInt64{},
		ReimbursementReference: sql.NullString{String: "N3p4", Valid: true},
	}); err != nil {
		t.Fatalf("failed to insert other participant: %v", err)
	}

	req := api.UpdateReimbursementRequest{}
	if err := req.FromUpdateReimbursementRequest0(api.UpdateReimbursementRequest0{
		ParticipantId:           strconv.FormatInt(otherParticipantID, 10),
		ReimbursementReceivedAt: nullable.NewNullNullable[time.Time](),
	}); err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPut, "/api/games/g1/reimbursements", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(participantID)}))
	w := httptest.NewRecorder()

	srv.PutApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusBadRequest, w.Code, w.Body.String())
	}
}

func TestPutApiGamesIdReimbursements_NonParticipantForbidden(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	staticClock := clock.StaticClock{Time: time.Now()}
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "organizer@example.com")
	nonParticipantID := dbtesting.UpsertTestUser(t, sqlDB, "nonparticipant@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createGame(t, querier, "g1", organizerID, sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true})

	req := api.UpdateReimbursementRequest{}
	if err := req.FromUpdateReimbursementRequest1(api.UpdateReimbursementRequest1{
		ReimbursedAt: nullable.NewNullNullable[time.Time](),
	}); err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	body, _ := json.Marshal(req)

	r := httptest.NewRequest(http.MethodPut, "/api/games/g1/reimbursements", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(nonParticipantID)}))
	w := httptest.NewRecorder()

	srv.PutApiGamesIdReimbursements(w, r, "g1")

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusForbidden, w.Code, w.Body.String())
	}
}
