package server_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	servertesting "github.com/dmateusp/opengym/api/server/testing"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
	"github.com/dmateusp/opengym/ptr"
	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestPostApiGames_Success(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	req := api.CreateGameRequest{
		Name:               "Sunday Morning Volleyball",
		Description:        ptr.Ptr("Indoor volleyball - all levels welcome!"),
		TotalPriceCents:    ptr.Ptr(int64(1500)),
		Location:           ptr.Ptr("123 Main St, Downtown"),
		StartsAt:           ptr.Ptr(staticClock.Now()),
		DurationMinutes:    ptr.Ptr(int64(120)),
		MaxPlayers:         ptr.Ptr(int64(12)),
		MaxGuestsPerPlayer: ptr.Ptr(int64(2)),
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()

	srv.PostApiGames(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response api.GameDetail
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response matches OpenAPI schema
	if response.Game.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, response.Game.Name)
	}
	if response.Game.OrganizerId != int(testUserID) {
		t.Errorf("Expected organizer ID %d, got %d", testUserID, response.Game.OrganizerId)
	}
	if response.Game.Description == nil || *response.Game.Description != *req.Description {
		t.Errorf("Expected description %s, got %v", *req.Description, response.Game.Description)
	}
	if response.Game.TotalPriceCents == nil || *response.Game.TotalPriceCents != int64(*req.TotalPriceCents) {
		t.Errorf("Expected price %d, got %v", *req.TotalPriceCents, response.Game.TotalPriceCents)
	}

	// Required fields per OpenAPI spec
	if response.Game.Id == "" {
		t.Error("Expected id to be set")
	}
	if response.Game.CreatedAt.IsZero() {
		t.Error("Expected createdAt to be set")
	}
	if response.Game.UpdatedAt.IsZero() {
		t.Error("Expected updatedAt to be set")
	}
}

func TestPostApiGames_MinimalRequest(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Only required field per OpenAPI spec
	req := api.CreateGameRequest{
		Name: "Minimal Game",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()

	srv.PostApiGames(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response api.GameDetail
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Game.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, response.Game.Name)
	}
}

func TestPostApiGames_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	req := api.CreateGameRequest{
		Name: "Test Game",
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	// No auth context
	w := httptest.NewRecorder()

	srv.PostApiGames(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPostApiGames_InvalidRequestBody(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader([]byte("invalid json")))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()

	srv.PostApiGames(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostApiGames_NameValidation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	testCases := []struct {
		name           string
		gameName       string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty name",
			gameName:       "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name",
		},
		{
			name:           "name too long",
			gameName:       string(make([]byte, 101)), // 101 characters (max is 100)
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name",
		},
		{
			name:           "name at max length",
			gameName:       string(make([]byte, 100)), // exactly 100 characters
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid short name",
			gameName:       "A",
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := api.CreateGameRequest{
				Name: tc.gameName,
			}

			body, _ := json.Marshal(req)
			r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
			r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
			w := httptest.NewRecorder()

			srv.PostApiGames(w, r)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.expectedError != "" && !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %s", tc.expectedError, w.Body.String())
			}
		})
	}
}

func TestPostApiGames_DescriptionValidation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	testCases := []struct {
		name           string
		description    *string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "description too long",
			description:    ptr.Ptr(string(make([]byte, 1001))), // 1001 characters (max is 1000)
			expectedStatus: http.StatusBadRequest,
			expectedError:  "description",
		},
		{
			name:           "description at max length",
			description:    ptr.Ptr(string(make([]byte, 1000))), // exactly 1000 characters
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "valid description",
			description:    ptr.Ptr("A valid description"),
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "empty description",
			description:    ptr.Ptr(""),
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "nil description",
			description:    nil,
			expectedStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := api.CreateGameRequest{
				Name:        "Valid Game Name",
				Description: tc.description,
			}

			body, _ := json.Marshal(req)
			r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
			r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
			w := httptest.NewRecorder()

			srv.PostApiGames(w, r)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.expectedError != "" && !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %s", tc.expectedError, w.Body.String())
			}
		})
	}
}

func TestPostApiGames_IDClashRetry(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)

	srv := server.NewServer(db.NewQuerierWrapper(querier), servertesting.NewTestAlphanumericGenerator("foo", "bar"), staticClock, sqlDB)

	// Pre-populate the database with a game that has ID "test"
	// to demonstrate that the retry logic works
	params := db.GameCreateParams{
		ID:          "foo",
		OrganizerID: int64(testUserID),
		Name:        "Existing Game",
	}
	_, err := querier.GameCreate(context.Background(), params)
	if err != nil {
		t.Fatalf("Failed to pre-populate game: %v", err)
	}

	req := api.CreateGameRequest{
		Name: "New Game",
	}

	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()

	srv.PostApiGames(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var response api.GameDetail
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Game.Id == "" {
		t.Error("Expected id to be set")
	}

	if response.Game.Id != "bar" {
		t.Error("Expected id to be 'bar'")
	}
}

func TestGetApiGames_DefaultPagination(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	userID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	res, err := sqlDB.Exec(`insert into users (email, name) VALUES (?, ?)`, "other@example.com", "Other")
	if err != nil {
		t.Fatalf("failed to create second user: %v", err)
	}
	otherUserID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("failed to fetch second user id: %v", err)
	}
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	baseTime := staticClock.Now()

	for i := 0; i < 12; i++ {
		_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
			ID:          fmt.Sprintf("g%02d", i),
			OrganizerID: int64(userID),
			Name:        fmt.Sprintf("Game %02d", i),
			PublishedAt: sql.NullTime{Time: baseTime.Add(time.Duration(i) * time.Minute), Valid: true},
		})
		if err != nil {
			t.Fatalf("failed to create game %d: %v", i, err)
		}
	}

	for i := 0; i < 3; i++ {
		_, err := querier.GameCreate(context.Background(), db.GameCreateParams{
			ID:          fmt.Sprintf("o%02d", i),
			OrganizerID: int64(otherUserID),
			Name:        fmt.Sprintf("Other %02d", i),
		})
		if err != nil {
			t.Fatalf("failed to create other user's game %d: %v", i, err)
		}
	}

	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(userID)}))
	w := httptest.NewRecorder()

	srv.GetApiGames(w, r, api.GetApiGamesParams{})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Total != 12 {
		t.Fatalf("expected total 12, got %d", resp.Total)
	}
	if resp.Page != 1 || resp.PageSize != 10 {
		t.Fatalf("unexpected pagination: page %d size %d", resp.Page, resp.PageSize)
	}
	if len(resp.Items) != 10 {
		t.Fatalf("expected 10 items, got %d", len(resp.Items))
	}
	if resp.Items[0].Id != "g11" {
		t.Fatalf("expected first item g11, got %s", resp.Items[0].Id)
	}
	for _, item := range resp.Items {
		if !item.IsOrganizer {
			t.Fatalf("expected isOrganizer true for item %s", item.Id)
		}
		// Verify organizer information is present
		if item.Organizer.Email != openapi_types.Email("john@example.com") {
			t.Fatalf("expected organizer email john@example.com for item %s, got %s", item.Id, item.Organizer.Email)
		}
	}

	page := 2
	pageSize := 5
	params := api.GetApiGamesParams{Page: &page, PageSize: &pageSize}
	r2 := httptest.NewRequest(http.MethodGet, "/api/games?page=2&pageSize=5", nil)
	r2 = r2.WithContext(auth.WithAuthInfo(r2.Context(), auth.AuthInfo{UserId: int(userID)}))
	w2 := httptest.NewRecorder()

	srv.GetApiGames(w2, r2, params)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w2.Code, w2.Body.String())
	}

	var resp2 api.GameListResponse
	if err := json.NewDecoder(w2.Body).Decode(&resp2); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp2.Total != 12 {
		t.Fatalf("expected total 12, got %d", resp2.Total)
	}
	if resp2.Page != 2 || resp2.PageSize != 5 {
		t.Fatalf("unexpected pagination: page %d size %d", resp2.Page, resp2.PageSize)
	}
	if len(resp2.Items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(resp2.Items))
	}
	if resp2.Items[0].Id != "g06" {
		t.Fatalf("expected first item on page 2 to be g06, got %s", resp2.Items[0].Id)
	}
}

func TestGetApiGames_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	w := httptest.NewRecorder()

	srv.GetApiGames(w, r, api.GetApiGamesParams{})

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestGetApiGames_ReturnsOrganizerGames(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	// Create three users
	user1ID := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2ID := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3ID := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	now := staticClock.Now()

	// User1 organizes 2 games
	game1, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game1",
		OrganizerID: int64(user1ID),
		Name:        "User1 Game 1",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game1: %v", err)
	}

	game2, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game2",
		OrganizerID: int64(user1ID),
		Name:        "User1 Game 2",
		PublishedAt: sql.NullTime{Time: now.Add(time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game2: %v", err)
	}

	// User2 organizes 1 game
	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game3",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game3: %v", err)
	}

	// User3 organizes 1 game
	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game4",
		OrganizerID: int64(user3ID),
		Name:        "User3 Game",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game4: %v", err)
	}

	// Request games as user1
	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(user1ID)}))
	w := httptest.NewRecorder()

	srv.GetApiGames(w, r, api.GetApiGamesParams{})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// User1 should see only their 2 organized games
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	// Verify both games belong to user1 and isOrganizer is true
	gameIDs := map[string]bool{}
	for _, item := range resp.Items {
		gameIDs[item.Id] = true
		if !item.IsOrganizer {
			t.Errorf("expected isOrganizer true for game %s", item.Id)
		}
	}

	if !gameIDs[game1.ID] || !gameIDs[game2.ID] {
		t.Errorf("expected games game1 and game2, got %v", gameIDs)
	}
}

func TestGetApiGames_ReturnsParticipantGames(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	// Create three users
	user1ID := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2ID := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3ID := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	now := staticClock.Now()

	// User2 organizes 2 games
	game1, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game1",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 1",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game1: %v", err)
	}

	game2, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game2",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 2",
		PublishedAt: sql.NullTime{Time: now.Add(time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game2: %v", err)
	}

	// User3 organizes 1 game
	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game3",
		OrganizerID: int64(user3ID),
		Name:        "User3 Game",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game3: %v", err)
	}

	// User1 participates in game1 and game2 (but not game3)
	err = querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1ID),
		GameID:         game1.ID,
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: staticClock.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to add user1 to game1: %v", err)
	}

	err = querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1ID),
		GameID:         game2.ID,
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: staticClock.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to add user1 to game2: %v", err)
	}

	// Request games as user1
	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(user1ID)}))
	w := httptest.NewRecorder()

	srv.GetApiGames(w, r, api.GetApiGamesParams{})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// User1 should see only the 2 games they participate in
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	// Verify both games are the ones user1 participates in and isOrganizer is false
	gameIDs := map[string]bool{}
	for _, item := range resp.Items {
		gameIDs[item.Id] = true
		if item.IsOrganizer {
			t.Errorf("expected isOrganizer false for game %s (user1 is participant, not organizer)", item.Id)
		}
	}

	if !gameIDs[game1.ID] || !gameIDs[game2.ID] {
		t.Errorf("expected games game1 and game2, got %v", gameIDs)
	}
}

func TestGetApiGames_ReturnsBothOrganizerAndParticipantGames(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	// Create three users
	user1ID := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2ID := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3ID := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	now := staticClock.Now()

	// User1 organizes 2 games
	game1, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game1",
		OrganizerID: int64(user1ID),
		Name:        "User1 Organized Game 1",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game1: %v", err)
	}

	game2, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game2",
		OrganizerID: int64(user1ID),
		Name:        "User1 Organized Game 2",
		PublishedAt: sql.NullTime{Time: now.Add(time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game2: %v", err)
	}

	// User2 organizes 2 games
	game3, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game3",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 1",
		PublishedAt: sql.NullTime{Time: now.Add(2 * time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game3: %v", err)
	}

	game4, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game4",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 2",
		PublishedAt: sql.NullTime{Time: now.Add(3 * time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game4: %v", err)
	}

	// User3 organizes 1 game
	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game5",
		OrganizerID: int64(user3ID),
		Name:        "User3 Game",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game5: %v", err)
	}

	// User1 also participates in game3 and game4 (organized by user2)
	err = querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1ID),
		GameID:         game3.ID,
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: staticClock.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to add user1 to game3: %v", err)
	}

	err = querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1ID),
		GameID:         game4.ID,
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: staticClock.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to add user1 to game4: %v", err)
	}

	// Request games as user1
	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(user1ID)}))
	w := httptest.NewRecorder()

	srv.GetApiGames(w, r, api.GetApiGamesParams{})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// User1 should see 4 games total: 2 they organize + 2 they participate in
	if resp.Total != 4 {
		t.Errorf("expected total 4, got %d", resp.Total)
	}

	if len(resp.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(resp.Items))
	}

	// Count organizer vs participant games
	organizerGames := map[string]bool{}
	participantGames := map[string]bool{}
	for _, item := range resp.Items {
		if item.IsOrganizer {
			organizerGames[item.Id] = true
		} else {
			participantGames[item.Id] = true
		}
	}

	// Verify user1 sees their 2 organized games with isOrganizer=true
	if len(organizerGames) != 2 {
		t.Errorf("expected 2 organizer games, got %d", len(organizerGames))
	}
	if !organizerGames[game1.ID] || !organizerGames[game2.ID] {
		t.Errorf("expected organizer games game1 and game2, got %v", organizerGames)
	}

	// Verify user1 sees their 2 participant games with isOrganizer=false
	if len(participantGames) != 2 {
		t.Errorf("expected 2 participant games, got %d", len(participantGames))
	}
	if !participantGames[game3.ID] || !participantGames[game4.ID] {
		t.Errorf("expected participant games game3 and game4, got %v", participantGames)
	}
}

func TestGetApiGames_OnlyReturnsUserGames(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	// Create three users
	user1ID := dbtesting.UpsertTestUser(t, sqlDB, "user1@example.com")
	user2ID := dbtesting.UpsertTestUser(t, sqlDB, "user2@example.com")
	user3ID := dbtesting.UpsertTestUser(t, sqlDB, "user3@example.com")

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	now := staticClock.Now()

	// User1 organizes 1 game
	game1, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game1",
		OrganizerID: int64(user1ID),
		Name:        "User1 Game",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game1: %v", err)
	}

	// User2 organizes 3 games
	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game2",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 1",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game2: %v", err)
	}

	game3, err := querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game3",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 2",
		PublishedAt: sql.NullTime{Time: now.Add(time.Minute), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game3: %v", err)
	}

	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game4",
		OrganizerID: int64(user2ID),
		Name:        "User2 Game 3",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game4: %v", err)
	}

	// User3 organizes 2 games
	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game5",
		OrganizerID: int64(user3ID),
		Name:        "User3 Game 1",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game5: %v", err)
	}

	_, err = querier.GameCreate(context.Background(), db.GameCreateParams{
		ID:          "game6",
		OrganizerID: int64(user3ID),
		Name:        "User3 Game 2",
		PublishedAt: sql.NullTime{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to create game6: %v", err)
	}

	// User1 participates in only one of User2's games (game3)
	err = querier.ParticipantsUpsert(context.Background(), db.ParticipantsUpsertParams{
		GoingUpdatedAt: staticClock.Now(),
		UserID:         int64(user1ID),
		GameID:         game3.ID,
		Going:          sql.NullBool{Bool: true, Valid: true},
		ConfirmedAt:    sql.NullTime{Time: staticClock.Now(), Valid: true},
	})
	if err != nil {
		t.Fatalf("failed to add user1 to game3: %v", err)
	}

	// Request games as user1
	r := httptest.NewRequest(http.MethodGet, "/api/games", nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(user1ID)}))
	w := httptest.NewRecorder()

	srv.GetApiGames(w, r, api.GetApiGamesParams{})

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var resp api.GameListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// User1 should see only 2 games: 1 they organize + 1 they participate in
	// They should NOT see user2's other games (game2, game4) or any of user3's games
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}

	if len(resp.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp.Items))
	}

	// Verify correct games are returned
	gameIDs := map[string]bool{}
	for _, item := range resp.Items {
		gameIDs[item.Id] = true
	}

	if !gameIDs[game1.ID] {
		t.Errorf("expected to see game1 (organized by user1)")
	}
	if !gameIDs[game3.ID] {
		t.Errorf("expected to see game3 (user1 participates)")
	}
	if gameIDs["game2"] || gameIDs["game4"] || gameIDs["game5"] || gameIDs["game6"] {
		t.Errorf("user1 should not see games they don't organize or participate in, got %v", gameIDs)
	}
}

// TestPostApiGames_DatabaseError is skipped when using real database
// Database errors are better tested through integration tests

func TestPatchApiGamesId_Success(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game first
	createReq := api.CreateGameRequest{
		Name: "Old Name",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.GameDetail
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Game.Id

	// Now update it
	newName := "Updated Name"
	newDescription := "Updated description"
	publishAt := staticClock.Now()

	updateReq := api.UpdateGameRequest{
		Name:        &newName,
		Description: &newDescription,
		PublishedAt: nullable.NewNullableWithValue(publishAt),
	}

	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+gameID, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w = httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, gameID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response api.GameDetail
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response matches OpenAPI schema and updates
	if response.Game.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, response.Game.Name)
	}
	if response.Game.Description == nil || *response.Game.Description != newDescription {
		t.Errorf("Expected description %s, got %v", newDescription, response.Game.Description)
	}
	if response.Game.PublishedAt == nil {
		t.Error("Expected publishedAt to be set")
	}
}

func TestPatchApiGamesId_PartialUpdate(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game first
	createReq := api.CreateGameRequest{
		Name: "Old Name",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.GameDetail
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Game.Id

	// Now update only the name
	newName := "Updated Name Only"
	req := api.UpdateGameRequest{
		Name: &newName,
	}

	body, _ = json.Marshal(req)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+gameID, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w = httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, gameID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response api.GameDetail
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Game.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, response.Game.Name)
	}
}

func TestPatchApiGamesId_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	req := api.UpdateGameRequest{
		Name: ptr.Ptr("Test"),
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPatch, "/api/games/abc1", bytes.NewReader(body))
	// No auth context
	w := httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, "abc1")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestPatchApiGamesId_NotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	req := api.UpdateGameRequest{
		Name: ptr.Ptr("Test"),
	}
	body, _ := json.Marshal(req)
	r := httptest.NewRequest(http.MethodPatch, "/api/games/notfound", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, "notfound")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestPatchApiGamesId_Forbidden(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	// Create organizer user and their game
	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{
		Name: "Test Game",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.GameDetail
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Game.Id

	// Create another user who tries to update the game
	otherUser, err := sqlDB.Exec(`INSERT INTO users (email, name) VALUES (?, ?)`, "other@example.com", "Other User")
	if err != nil {
		t.Fatalf("Failed to create other user: %v", err)
	}
	otherUserID, _ := otherUser.LastInsertId()

	// Try to update as different user
	req := api.UpdateGameRequest{
		Name: ptr.Ptr("Hacked Name"),
	}
	body, _ = json.Marshal(req)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+gameID, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(otherUserID)}))
	w = httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, gameID)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestPatchApiGamesId_InvalidRequestBody(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game first
	createReq := api.CreateGameRequest{
		Name: "Test Game",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.GameDetail
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Game.Id

	// Try to update with invalid JSON
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+gameID, bytes.NewReader([]byte("invalid json")))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w = httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, gameID)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetApiGamesId_Success(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	testUserID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game first
	createReq := api.CreateGameRequest{
		Name:        "Test Game",
		Description: ptr.Ptr("Test Description"),
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.GameDetail
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Game.Id

	// Now retrieve it
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+gameID, nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w = httptest.NewRecorder()

	srv.GetApiGamesId(w, r, gameID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var gameDetail api.GameDetail
	if err := json.NewDecoder(w.Body).Decode(&gameDetail); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify retrieved game matches created game
	if gameDetail.Game.Id != createdGame.Game.Id {
		t.Errorf("Expected id %s, got %s", createdGame.Game.Id, gameDetail.Game.Id)
	}
	if gameDetail.Game.Name != createdGame.Game.Name {
		t.Errorf("Expected name %s, got %s", createdGame.Game.Name, gameDetail.Game.Name)
	}
	if gameDetail.Game.OrganizerId != createdGame.Game.OrganizerId {
		t.Errorf("Expected organizer ID %d, got %d", createdGame.Game.OrganizerId, gameDetail.Game.OrganizerId)
	}
	if gameDetail.Game.Description == nil || *gameDetail.Game.Description != *createdGame.Game.Description {
		t.Errorf("Expected description %v, got %v", createdGame.Game.Description, gameDetail.Game.Description)
	}

	// Verify organizer information is returned
	if gameDetail.Organizer.Id == "" {
		t.Error("Expected organizer ID to be set")
	}
	if gameDetail.Organizer.Email == "" {
		t.Error("Expected organizer email to be set")
	}
	if gameDetail.Organizer.Email != "john@example.com" {
		t.Errorf("Expected organizer email john@example.com, got %s", gameDetail.Organizer.Email)
	}
	if gameDetail.Organizer.IsDemo {
		t.Error("Expected organizer to not be a demo user (default is false)")
	}
}

func TestGetApiGamesId_NotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Try to retrieve a non-existent game
	r := httptest.NewRequest(http.MethodGet, "/api/games/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.GetApiGamesId(w, r, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestGetApiGamesId_DraftHiddenFromNonOrganizer(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Secret Draft"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	// Unauthenticated user should not see draft
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+created.Game.Id, nil)
	w = httptest.NewRecorder()
	srv.GetApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for draft game, got %d", w.Code)
	}

	// Other authenticated user should not see draft
	res, err := sqlDB.Exec(`insert into users (email, name) values (?, ?)`, "other@example.com", "Other")
	if err != nil {
		t.Fatalf("failed to insert other user: %v", err)
	}
	otherID, _ := res.LastInsertId()
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+created.Game.Id, nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(otherID)}))
	w = httptest.NewRecorder()
	srv.GetApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for draft game to other user, got %d", w.Code)
	}

	// Organizer can see draft
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+created.Game.Id, nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.GetApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for organizer viewing draft, got %d", w.Code)
	}
}

func TestGetApiGamesId_ScheduledHiddenUntilPublished(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Scheduled Game"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	future := staticClock.Now().Add(1 * time.Hour)
	updateReq := api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(future)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when scheduling publish, got %d", w.Code)
	}

	// Non-organizer still should not see until publish time
	res, err := sqlDB.Exec(`insert into users (email, name) values (?, ?)`, "viewer@example.com", "Viewer")
	if err != nil {
		t.Fatalf("failed to insert viewer user: %v", err)
	}
	viewerID, _ := res.LastInsertId()
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+created.Game.Id, nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(viewerID)}))
	w = httptest.NewRecorder()
	srv.GetApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for scheduled game before publish time, got %d", w.Code)
	}

	// Organizer can still view
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+created.Game.Id, nil)
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.GetApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for organizer viewing scheduled game, got %d", w.Code)
	}
}

func TestPatchApiGamesId_PublishPastBecomesNow(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Past Publish"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)
	start := staticClock.Now()
	past := start.Add(-1 * time.Hour)
	updateReq := api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(past)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when publishing with past timestamp, got %d", w.Code)
	}

	var updated api.GameDetail
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Game.PublishedAt == nil {
		t.Fatalf("expected publishedAt to be set")
	}
	if updated.Game.PublishedAt.Before(start) {
		t.Fatalf("expected publishedAt to be >= request time, got %v", updated.Game.PublishedAt)
	}
}

func TestPatchApiGamesId_CannotPublishTwice(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Single Publish"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)
	first := staticClock.Now()
	updateReq := api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(first)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected first publish to succeed, got %d", w.Code)
	}

	second := staticClock.Now().Add(2 * time.Hour)
	updateReq = api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(second)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected second publish attempt to be rejected with 400, got %d", w.Code)
	}
}

func TestPatchApiGamesId_CanRescheduleFuturePublish(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Reschedulable"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	future1 := staticClock.Now().Add(2 * time.Hour)
	updateReq := api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(future1)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected scheduling publish to succeed, got %d", w.Code)
	}

	future2 := staticClock.Now().Add(4 * time.Hour)
	updateReq = api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(future2)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected rescheduling publish to succeed, got %d", w.Code)
	}

	var updated api.GameDetail
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Game.PublishedAt == nil {
		t.Fatalf("expected publishedAt to remain set after reschedule")
	}
	if !updated.Game.PublishedAt.After(future1) {
		t.Fatalf("expected publishedAt to move later than original schedule")
	}
}

func TestPatchApiGamesId_CanClearFuturePublish(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Clearable"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	future := staticClock.Now().Add(2 * time.Hour)
	updateReq := api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(future)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected scheduling publish to succeed, got %d", w.Code)
	}

	updateReq = api.UpdateGameRequest{PublishedAt: nullable.NewNullNullable[time.Time]()}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected clearing scheduled publish to succeed, got %d", w.Code)
	}

	var updated api.GameDetail
	json.NewDecoder(w.Body).Decode(&updated)
	if updated.Game.PublishedAt != nil {
		t.Fatalf("expected publishedAt to be cleared")
	}
}

func TestPatchApiGamesId_CannotClearAfterPublished(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	createReq := api.CreateGameRequest{Name: "Published"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	past := staticClock.Now().Add(-1 * time.Hour)
	updateReq := api.UpdateGameRequest{PublishedAt: nullable.NewNullableWithValue(past)}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusOK {
		t.Fatalf("expected publishing to succeed, got %d", w.Code)
	}

	updateReq = api.UpdateGameRequest{PublishedAt: nullable.NewNullNullable[time.Time]()}
	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w = httptest.NewRecorder()
	srv.PatchApiGamesId(w, r, created.Game.Id)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected clearing after publish to be rejected with 400, got %d", w.Code)
	}
}

func TestPatchApiGamesId_NameValidation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game first
	createReq := api.CreateGameRequest{Name: "Original Game"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	testCases := []struct {
		name           string
		gameName       *string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "empty name",
			gameName:       ptr.Ptr(""),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name",
		},
		{
			name:           "name too long",
			gameName:       ptr.Ptr(string(make([]byte, 101))), // 101 characters (max is 100)
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name",
		},
		{
			name:           "name at max length",
			gameName:       ptr.Ptr(string(make([]byte, 100))), // exactly 100 characters
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid short name",
			gameName:       ptr.Ptr("Updated Name"),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateReq := api.UpdateGameRequest{
				Name: tc.gameName,
			}

			body, _ := json.Marshal(updateReq)
			r := httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
			r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
			w := httptest.NewRecorder()

			srv.PatchApiGamesId(w, r, created.Game.Id)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.expectedError != "" && !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %s", tc.expectedError, w.Body.String())
			}
		})
	}
}

func TestPatchApiGamesId_DescriptionValidation(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()
	staticClock := clock.StaticClock{Time: time.Now()}

	organizerID := dbtesting.UpsertTestUser(t, sqlDB, "john@example.com")
	querier := db.New(sqlDB)
	srv := server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), staticClock, sqlDB)

	// Create a game first
	createReq := api.CreateGameRequest{Name: "Original Game"}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var created api.GameDetail
	json.NewDecoder(w.Body).Decode(&created)

	testCases := []struct {
		name           string
		description    *string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "description too long",
			description:    ptr.Ptr(string(make([]byte, 1001))), // 1001 characters (max is 1000)
			expectedStatus: http.StatusBadRequest,
			expectedError:  "description",
		},
		{
			name:           "description at max length",
			description:    ptr.Ptr(string(make([]byte, 1000))), // exactly 1000 characters
			expectedStatus: http.StatusOK,
		},
		{
			name:           "valid description",
			description:    ptr.Ptr("Updated description"),
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty description",
			description:    ptr.Ptr(""),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			updateReq := api.UpdateGameRequest{
				Description: tc.description,
			}

			body, _ := json.Marshal(updateReq)
			r := httptest.NewRequest(http.MethodPatch, "/api/games/"+created.Game.Id, bytes.NewReader(body))
			r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
			w := httptest.NewRecorder()

			srv.PatchApiGamesId(w, r, created.Game.Id)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.expectedError != "" && !strings.Contains(w.Body.String(), tc.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %s", tc.expectedError, w.Body.String())
			}
		})
	}
}
