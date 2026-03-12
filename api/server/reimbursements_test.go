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

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
	"github.com/oapi-codegen/nullable"
)

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
		UserID:         participantID,
		GameID:         "g1",
		Going:          sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt: staticClock.Now(),
		ConfirmedAt:    sql.NullTime{},
		Guests:         sql.NullInt64{},
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
		UserID:         participantID,
		GameID:         "g1",
		Going:          sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt: staticClock.Now(),
		ConfirmedAt:    sql.NullTime{},
		Guests:         sql.NullInt64{},
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
		UserID:         participantID,
		GameID:         "g1",
		Going:          sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt: staticClock.Now(),
		ConfirmedAt:    sql.NullTime{},
		Guests:         sql.NullInt64{},
	}); err != nil {
		t.Fatalf("failed to insert participant: %v", err)
	}
	if err := querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		UserID:         otherParticipantID,
		GameID:         "g1",
		Going:          sql.NullBool{Valid: true, Bool: true},
		GoingUpdatedAt: staticClock.Now(),
		ConfirmedAt:    sql.NullTime{},
		Guests:         sql.NullInt64{},
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
