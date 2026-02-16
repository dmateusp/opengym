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
	"github.com/oapi-codegen/nullable"
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

	// Validate name
	if len(req.Name) == 0 {
		http.Error(w, "name cannot be empty", http.StatusBadRequest)
		return
	}
	if len(req.Name) > 100 {
		http.Error(w, "name cannot exceed 100 characters", http.StatusBadRequest)
		return
	}

	// Validate description
	if req.Description != nil && len(*req.Description) > 1000 {
		http.Error(w, "description cannot exceed 1000 characters", http.StatusBadRequest)
		return
	}

	if req.DurationMinutes != nil && *req.DurationMinutes <= 0 {
		http.Error(w, "duration must be positive", http.StatusBadRequest)
		return
	}

	if req.MaxPlayers != nil && *req.MaxPlayers < 1 {
		http.Error(w, "maxPlayers must be at least 1", http.StatusBadRequest)
		return
	}

	if req.MaxGuestsPerPlayer != nil && *req.MaxGuestsPerPlayer < 0 {
		http.Error(w, "maxGuestsPerPlayer cannot be negative", http.StatusBadRequest)
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
		} else {
			params.MaxPlayers = 100 // Default to 100
		}
		params.GameSpotsLeft = params.MaxPlayers

		if req.MaxGuestsPerPlayer != nil {
			params.MaxGuestsPerPlayer = int64(*req.MaxGuestsPerPlayer)
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

	// Fetch organizer information
	organizerRow, err := srv.querier.UserGetById(r.Context(), game.OrganizerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve organizer: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	gameWithOrganizer := db.GameGetByIdWithOrganizerRow{
		Game: game,
		User: organizerRow.User,
	}

	var apiGameDetail api.GameDetail
	apiGameDetail.FromDb(gameWithOrganizer)
	err = json.NewEncoder(w).Encode(apiGameDetail)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (srv *server) GetApiGames(w http.ResponseWriter, r *http.Request, params api.GetApiGamesParams) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	page := 1
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}

	pageSize := 10
	if params.PageSize != nil {
		pageSize = *params.PageSize
		if pageSize > 25 {
			pageSize = 25
		}
		if pageSize < 1 {
			pageSize = 1
		}
	}

	offset := int64((page - 1) * pageSize)
	limit := int64(pageSize)

	rows, err := srv.querier.GameListByUser(r.Context(), db.GameListByUserParams{
		UserID: int64(authInfo.UserId),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list games: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	total, err := srv.querier.GameCountByUser(r.Context(), int64(authInfo.UserId))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to count games: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	items := make([]api.GameListItem, 0, len(rows))
	for _, row := range rows {
		var item api.GameListItem
		item.Id = row.ID
		item.Name = row.Name
		item.IsOrganizer = row.IsOrganizer
		if row.Location.Valid {
			loc := row.Location.String
			item.Location = &loc
		}
		if row.StartsAt.Valid {
			item.StartsAt = nullable.Nullable[time.Time]{}
			item.StartsAt.Set(row.StartsAt.Time)
		}
		if row.PublishedAt.Valid {
			item.PublishedAt = nullable.Nullable[time.Time]{}
			item.PublishedAt.Set(row.PublishedAt.Time)
		}
		item.UpdatedAt = row.UpdatedAt

		// Map organizer information
		item.Organizer.FromDb(row.User)

		items = append(items, item)
	}

	resp := api.GameListResponse{
		Items:    items,
		Total:    int(total),
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (srv *server) GetPublicApiGamesId(w http.ResponseWriter, r *http.Request, id string) {
	game, err := srv.querier.GameGetPublicInfoById(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	now := srv.clock.Now()
	isPublished := game.PublishedAt.Valid && !game.PublishedAt.Time.After(now)

	resp := api.PublicGameDetail{
		Id:   game.ID,
		Name: game.Name,
	}

	// Set organizer information
	if game.OrganizerName.Valid {
		resp.Organizer.Name = game.OrganizerName.String
	}
	if game.OrganizerPhoto.Valid {
		resp.Organizer.Picture = &game.OrganizerPhoto.String
	}

	if isPublished {
		// Game is published, show spots left and start time
		resp.GameSpotsLeft = &game.GameSpotsLeft
		if game.StartsAt.Valid {
			resp.StartsAt = &game.StartsAt.Time
		}
	} else if game.PublishedAt.Valid {
		// Game is not yet published, show when it will be published
		resp.PublishedAt = &game.PublishedAt.Time
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
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

func (srv *server) GetApiGamesId(w http.ResponseWriter, r *http.Request, id string) {
	game, err := srv.querier.GameGetByIdWithOrganizer(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	authInfo, hasAuth := auth.FromCtx(r.Context())
	isOrganizer := hasAuth && int64(authInfo.UserId) == game.Game.OrganizerID

	// Only organizers can see unpublished games
	if !isOrganizer && (!game.Game.PublishedAt.Valid || game.Game.PublishedAt.Time.After(srv.clock.Now())) {
		http.Error(w, "game not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	var apiGame api.GameDetail
	apiGame.FromDb(game)
	err = json.NewEncoder(w).Encode(apiGame)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (srv *server) PatchApiGamesId(w http.ResponseWriter, r *http.Request, id string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	now := srv.clock.Now()

	tx, err := srv.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to begin transaction: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	querierWithTx := srv.querier.WithTx(tx)

	game, err := querierWithTx.GameGetById(r.Context(), id)
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

	// Validate name
	if req.Name != nil {
		if len(*req.Name) == 0 {
			http.Error(w, "name cannot be empty", http.StatusBadRequest)
			return
		}
		if len(*req.Name) > 100 {
			http.Error(w, "name cannot exceed 100 characters", http.StatusBadRequest)
			return
		}
	}

	// Validate description
	if req.Description != nil && len(*req.Description) > 1000 {
		http.Error(w, "description cannot exceed 1000 characters", http.StatusBadRequest)
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

	if req.PublishedAt.IsNull() {
		if !game.PublishedAt.Time.After(now) {
			http.Error(w, "game is already published and cannot be unpublished", http.StatusBadRequest)
			return
		}
		params.ClearPublishedAt = true
	}

	if req.PublishedAt.IsSpecified() {
		var publishAt sql.NullTime
		if !req.PublishedAt.IsNull() {
			publishAt = sql.NullTime{Time: req.PublishedAt.MustGet(), Valid: true}
			if publishAt.Time.Before(now) {
				publishAt = sql.NullTime{Time: now, Valid: true}
			}
		}

		if game.PublishedAt.Valid {
			if game.PublishedAt.Time.After(now) {
				params.PublishedAt = publishAt
			} else {
				http.Error(w, "game has already been published", http.StatusBadRequest)
				return
			}
		} else {
			params.PublishedAt = publishAt
		}
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
		newMaxPlayers := int64(*req.MaxPlayers)
		params.MaxPlayers = sql.NullInt64{Valid: true, Int64: newMaxPlayers}
		newGameSpotsLeft := game.GameSpotsLeft + (newMaxPlayers - game.MaxPlayers)
		if newGameSpotsLeft < 0 {
			newGameSpotsLeft = 0
		}
		if newGameSpotsLeft > newMaxPlayers {
			newGameSpotsLeft = newMaxPlayers
		}
		params.GameSpotsLeft = sql.NullInt64{Valid: true, Int64: newGameSpotsLeft}
	}

	if req.MaxGuestsPerPlayer != nil {
		params.MaxGuestsPerPlayer = sql.NullInt64{
			Valid: true,
			Int64: int64(*req.MaxGuestsPerPlayer),
		}
	}

	err = querierWithTx.GameUpdate(r.Context(), params)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to update game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	updatedGame, err := querierWithTx.GameGetById(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve updated game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Fetch organizer information
	organizerRow, err := querierWithTx.UserGetById(r.Context(), updatedGame.OrganizerID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve organizer: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	tx.Commit()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	gameWithOrganizer := db.GameGetByIdWithOrganizerRow{
		Game: updatedGame,
		User: organizerRow.User,
	}

	var apiGameDetail api.GameDetail
	apiGameDetail.FromDb(gameWithOrganizer)
	err = json.NewEncoder(w).Encode(apiGameDetail)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
