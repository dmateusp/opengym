package server

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/db"
)

var (
	gameIDLength = flag.Int("game.id-length", 4, "Length of the game ID, keep it short for easier sharing, but not too short to avoid collisions. 4 = 62^4 possibilities.")
)

const maxGameIDAttempts = 3

func (srv *server) PostApiGames(w http.ResponseWriter, r *http.Request) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req api.CreateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	var game db.Game
	var err error

	// Try up to maxGameIDAttempts times to create a game with a unique ID
	for attempt := range maxGameIDAttempts {
		gameID := srv.randomAlphanumericGenerator.Generate(*gameIDLength)

		params := db.GameCreateParams{
			ID:          gameID,
			OrganizerID: int64(authInfo.UserId),
			Name:        req.Name,
		}

		if req.Description != nil {
			params.Description.String = *req.Description
			params.Description.Valid = true
		}

		if req.TotalPriceCents != nil {
			params.TotalPriceCents = int64(*req.TotalPriceCents)
		}

		if req.Location != nil {
			params.Location.String = *req.Location
			params.Location.Valid = true
		}

		if req.StartsAt != nil {
			params.StartsAt.Time = *req.StartsAt
			params.StartsAt.Valid = true
		}

		if req.DurationMinutes != nil {
			params.DurationMinutes = int64(*req.DurationMinutes)
		}

		if req.MaxPlayers != nil {
			params.MaxPlayers = int64(*req.MaxPlayers)
		}

		if req.MaxGuestsPerPlayer != nil {
			params.MaxGuestsPerPlayer = int64(*req.MaxGuestsPerPlayer)
		}

		if req.MaxWaitlistSize != nil {
			params.MaxWaitlistSize = int64(*req.MaxWaitlistSize)
		}

		game, err = srv.querier.GameCreate(r.Context(), params)
		if err == nil {
			// Successfully created, break out of retry loop
			break
		}

		// Check if this is a constraint violation (duplicate ID)
		if attempt < maxGameIDAttempts-1 && isConstraintError(err) {
			// Try again with a new ID
			continue
		}

		// For other errors or last attempt, return error
		http.Error(w, fmt.Sprintf("failed to create game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	var apiGame api.Game
	apiGame.FromDb(game)
	err = json.NewEncoder(w).Encode(apiGame)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func isConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed: games.id")
}

func (srv *server) PatchApiGamesId(w http.ResponseWriter, r *http.Request, id string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	game, err := srv.querier.GameGetById(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if game.OrganizerID != int64(authInfo.UserId) {
		http.Error(w, "forbidden: you are not the organizer of this game", http.StatusForbidden)
		return
	}

	var req api.UpdateGameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	params := db.GameUpdateParams{
		ID:   id,
		Name: game.Name,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}

	if req.Description != nil {
		params.Description.String = *req.Description
		params.Description.Valid = true
	}

	if req.Publish != nil && *req.Publish {
		params.PublishedAt.Time = time.Now()
		params.PublishedAt.Valid = true
	}

	if req.TotalPriceCents != nil {
		params.TotalPriceCents = int64(*req.TotalPriceCents)
	}

	if req.Location != nil {
		params.Location.String = *req.Location
		params.Location.Valid = true
	}

	if req.StartsAt != nil {
		params.StartsAt.Time = *req.StartsAt
		params.StartsAt.Valid = true
	}

	if req.DurationMinutes != nil {
		params.DurationMinutes = int64(*req.DurationMinutes)
	}

	if req.MaxPlayers != nil {
		params.MaxPlayers = int64(*req.MaxPlayers)
	}

	if req.MaxGuestsPerPlayer != nil {
		params.MaxGuestsPerPlayer = int64(*req.MaxGuestsPerPlayer)
	}

	if req.MaxWaitlistSize != nil {
		params.MaxWaitlistSize = int64(*req.MaxWaitlistSize)
	}

	err = srv.querier.GameUpdate(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	updatedGame, err := srv.querier.GameGetById(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve updated game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	var apiGame api.Game
	apiGame.FromDb(updatedGame)
	err = json.NewEncoder(w).Encode(apiGame)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
