package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	servertesting "github.com/dmateusp/opengym/api/server/testing"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/db"
	dbtesting "github.com/dmateusp/opengym/db/testing"
	"github.com/dmateusp/opengym/ptr"
)

func TestPostApiGames_Success(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	now := time.Now()
	req := api.CreateGameRequest{
		Name:               "Sunday Morning Volleyball",
		Description:        ptr.Ptr("Indoor volleyball - all levels welcome!"),
		TotalPriceCents:    ptr.Ptr(int64(1500)),
		Location:           ptr.Ptr("123 Main St, Downtown"),
		StartsAt:           &now,
		DurationMinutes:    ptr.Ptr(int64(120)),
		MaxPlayers:         ptr.Ptr(int64(12)),
		MaxWaitlistSize:    ptr.Ptr(int64(5)),
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

	var response api.Game
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response matches OpenAPI schema
	if response.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, response.Name)
	}
	if response.OrganizerId != int(testUserID) {
		t.Errorf("Expected organizer ID %d, got %d", testUserID, response.OrganizerId)
	}
	if response.Description == nil || *response.Description != *req.Description {
		t.Errorf("Expected description %s, got %v", *req.Description, response.Description)
	}
	if response.TotalPriceCents == nil || *response.TotalPriceCents != int64(*req.TotalPriceCents) {
		t.Errorf("Expected price %d, got %v", *req.TotalPriceCents, response.TotalPriceCents)
	}

	// Required fields per OpenAPI spec
	if response.Id == "" {
		t.Error("Expected id to be set")
	}
	if response.CreatedAt.IsZero() {
		t.Error("Expected createdAt to be set")
	}
	if response.UpdatedAt.IsZero() {
		t.Error("Expected updatedAt to be set")
	}
}

func TestPostApiGames_MinimalRequest(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

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

	var response api.Game
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Name != req.Name {
		t.Errorf("Expected name %s, got %s", req.Name, response.Name)
	}
}

func TestPostApiGames_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

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

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader([]byte("invalid json")))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()

	srv.PostApiGames(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestPostApiGames_IDClashRetry(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)

	srv := server.NewServer(querier, servertesting.NewTestAlphanumericGenerator("foo", "bar"))

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

	var response api.Game
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Id == "" {
		t.Error("Expected id to be set")
	}

	if response.Id != "bar" {
		t.Error("Expected id to be 'bar'")
	}
}

// TestPostApiGames_DatabaseError is skipped when using real database
// Database errors are better tested through integration tests

func TestPatchApiGamesId_Success(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	// Create a game first
	createReq := api.CreateGameRequest{
		Name: "Old Name",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.Game
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Id

	// Now update it
	newName := "Updated Name"
	newDescription := "Updated description"
	publish := true

	updateReq := api.UpdateGameRequest{
		Name:        &newName,
		Description: &newDescription,
		Publish:     &publish,
	}

	body, _ = json.Marshal(updateReq)
	r = httptest.NewRequest(http.MethodPatch, "/api/games/"+gameID, bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w = httptest.NewRecorder()

	srv.PatchApiGamesId(w, r, gameID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response api.Game
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response matches OpenAPI schema and updates
	if response.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, response.Name)
	}
	if response.Description == nil || *response.Description != newDescription {
		t.Errorf("Expected description %s, got %v", newDescription, response.Description)
	}
	if response.PublishedAt == nil {
		t.Error("Expected publishedAt to be set")
	}
}

func TestPatchApiGamesId_PartialUpdate(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	// Create a game first
	createReq := api.CreateGameRequest{
		Name: "Old Name",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.Game
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Id

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

	var response api.Game
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Name != newName {
		t.Errorf("Expected name %s, got %s", newName, response.Name)
	}
}

func TestPatchApiGamesId_Unauthorized(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

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

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

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

	// Create organizer user and their game
	organizerID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	createReq := api.CreateGameRequest{
		Name: "Test Game",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(organizerID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.Game
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Id

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

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	// Create a game first
	createReq := api.CreateGameRequest{
		Name: "Test Game",
	}
	body, _ := json.Marshal(createReq)
	r := httptest.NewRequest(http.MethodPost, "/api/games", bytes.NewReader(body))
	r = r.WithContext(auth.WithAuthInfo(r.Context(), auth.AuthInfo{UserId: int(testUserID)}))
	w := httptest.NewRecorder()
	srv.PostApiGames(w, r)

	var createdGame api.Game
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Id

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

	testUserID := dbtesting.CreateTestUser(t, sqlDB)
	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

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

	var createdGame api.Game
	json.NewDecoder(w.Body).Decode(&createdGame)
	gameID := createdGame.Id

	// Now retrieve it
	r = httptest.NewRequest(http.MethodGet, "/api/games/"+gameID, nil)
	w = httptest.NewRecorder()

	srv.GetApiGamesId(w, r, gameID)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var retrievedGame api.Game
	if err := json.NewDecoder(w.Body).Decode(&retrievedGame); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify retrieved game matches created game
	if retrievedGame.Id != createdGame.Id {
		t.Errorf("Expected id %s, got %s", createdGame.Id, retrievedGame.Id)
	}
	if retrievedGame.Name != createdGame.Name {
		t.Errorf("Expected name %s, got %s", createdGame.Name, retrievedGame.Name)
	}
	if retrievedGame.OrganizerId != createdGame.OrganizerId {
		t.Errorf("Expected organizer ID %d, got %d", createdGame.OrganizerId, retrievedGame.OrganizerId)
	}
	if retrievedGame.Description == nil || *retrievedGame.Description != *createdGame.Description {
		t.Errorf("Expected description %v, got %v", createdGame.Description, retrievedGame.Description)
	}
}

func TestGetApiGamesId_NotFound(t *testing.T) {
	sqlDB := dbtesting.SetupTestDB(t)
	defer sqlDB.Close()

	querier := db.New(sqlDB)
	srv := server.NewServer(querier, server.NewRandomAlphanumericGenerator())

	// Try to retrieve a non-existent game
	r := httptest.NewRequest(http.MethodGet, "/api/games/nonexistent", nil)
	w := httptest.NewRecorder()

	srv.GetApiGamesId(w, r, "nonexistent")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}
