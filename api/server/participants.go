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
	"github.com/dmateusp/opengym/ptr"
)

func (s *server) GetApiGamesIdParticipants(w http.ResponseWriter, r *http.Request, id string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Verify game exists and user has access to it
	game, err := s.querier.GameGetById(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve game: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Non-organizers cannot view participants of unpublished or future games
	if game.OrganizerID != int64(authInfo.UserId) {
		if !game.PublishedAt.Valid || game.PublishedAt.Time.After(time.Now()) {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
	}

	// Get participants list
	rows, err := s.querier.ParticipantsList(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve participants: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Compute participation status for each participant
	participants := make([]api.ParticipantWithUser, 0, len(rows))
	goingCount := int64(0)

	for _, row := range rows {
		var status api.ParticipationStatus

		// Determine participation status based on going flag and position
		if row.GameParticipant.Going.Valid && row.GameParticipant.Going.Bool {
			// User marked as going - check if they're in the main list or waitlisted
			if game.MaxPlayers == -1 || goingCount < game.MaxPlayers {
				// Either unlimited players or there's space in the main list
				if err := status.FromParticipationStatusUpdate(api.Going); err != nil {
					http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
					return
				}
				goingCount++
			} else {
				// Waitlisted
				if err := status.FromParticipationStatus1(api.Waitlisted); err != nil {
					http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
					return
				}
			}
		} else {
			// User marked as not going or hasn't set status
			if err := status.FromParticipationStatusUpdate(api.NotGoing); err != nil {
				http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

		// Convert user from DB model to API model
		var user api.User
		user.FromDb(row.User)

		participants = append(participants, api.ParticipantWithUser{
			Status:    status,
			User:      user,
			CreatedAt: ptr.Ptr(row.GameParticipant.CreatedAt),
			UpdatedAt: ptr.Ptr(row.GameParticipant.UpdatedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(participants); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

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
