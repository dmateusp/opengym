package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/db"
)

func (s *server) PostApiGamesIdParticipants(w http.ResponseWriter, r *http.Request, id string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req api.UpdateGameParticipationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if req.Status != api.Going && req.Status != api.NotGoing {
		http.Error(w, "invalid status", http.StatusBadRequest)
		return
	}

	game, err := s.querier.GameGetById(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Non-organizers cannot interact with unpublished or future games.
	if game.OrganizerID != int64(authInfo.UserId) {
		if !game.PublishedAt.Valid || game.PublishedAt.Time.After(time.Now()) {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
	}

	going := sql.NullBool{Bool: req.Status == api.Going, Valid: true}
	confirmed := sql.NullBool{Valid: true, Bool: true}

	if err := s.querier.ParticipantsUpsert(r.Context(), db.ParticipantsUpsertParams{
		UserID:    int64(authInfo.UserId),
		GameID:    id,
		Going:     going,
		Confirmed: confirmed,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to update participation: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var status api.ParticipationStatus
	if err := status.FromParticipationStatusUpdate(req.Status); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	resp := api.GameParticipation{
		GameId: id,
		UserId: strconv.FormatInt(int64(authInfo.UserId), 10),
		Status: status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
