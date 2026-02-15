package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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
		if !game.PublishedAt.Valid || game.PublishedAt.Time.After(s.clock.Now()) {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
	}

	// Get participants list
	rows, err := s.querier.ParticipantsList(r.Context(), db.ParticipantsListParams{
		OrganizerID: game.OrganizerID,
		GameID:      game.ID,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve participants: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// Compute participation status for each participant; waitlist status doesn't enforce capacity except organizer overflow
	participants := make([]api.ParticipantWithUser, 0, len(rows))
	goingCount := int64(0)
	waitlistCount := int64(0)

	for _, row := range rows {
		var status api.ParticipationStatus

		// Determine participation status based on going flag and position
		if row.GameParticipant.Going.Valid && row.GameParticipant.Going.Bool {
			// User marked as going - compute segment based on capacities
			participantCount := int64(1)
			if row.GameParticipant.Guests.Valid {
				participantCount += row.GameParticipant.Guests.Int64
			}

			if goingCount+participantCount <= game.MaxPlayers {
				if err := status.FromParticipationStatusUpdate(api.Going); err != nil {
					http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
					return
				}
				goingCount += participantCount
			} else {
				// Waitlisted regardless of waitlist capacity; we track count for organizer overflow handling
				if err := status.FromParticipationStatus1(api.Waitlisted); err != nil {
					http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
					return
				}
				waitlistCount += participantCount
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

		var guests int
		if row.GameParticipant.Guests.Valid {
			guests = int(row.GameParticipant.Guests.Int64)
		}

		participants = append(participants, api.ParticipantWithUser{
			Status:    status,
			User:      user,
			CreatedAt: ptr.Ptr(row.GameParticipant.CreatedAt),
			UpdatedAt: ptr.Ptr(row.GameParticipant.GoingUpdatedAt),
			Guests:    guests,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(participants); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}

func (s *server) PutApiGamesIdParticipants(w http.ResponseWriter, r *http.Request, id string) {
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

	tx, err := s.dbConn.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to begin transaction: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	querierWithTx := s.querier.WithTx(tx)
	game, err := querierWithTx.GameGetById(r.Context(), id)
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
		if !game.PublishedAt.Valid || game.PublishedAt.Time.After(s.clock.Now()) {
			http.Error(w, "game not found", http.StatusNotFound)
			return
		}
	}

	going := sql.NullBool{Bool: req.Status == api.Going, Valid: true}

	confirmedAt := sql.NullTime{
		Time:  s.clock.Now(),
		Valid: req.Confirmed != nil,
	}

	// Participant and his/her potential guests
	participantsGroup := 1

	guests := sql.NullInt64{}
	if req.Guests != nil {
		if *req.Guests < 0 {
			http.Error(w, "invalid number of guests", http.StatusBadRequest)
			return
		}
		if int64(*req.Guests) <= game.MaxGuestsPerPlayer {
			guests.Int64 = int64(*req.Guests)
			guests.Valid = true
		} else {
			http.Error(w, "this game doesn't allow that many guests", http.StatusBadRequest)
			return
		}

		participantsGroup += int(*req.Guests)
	}

	var gameSpotsLeft sql.NullInt64
	var computedStatus api.ParticipationStatus

	switch req.Status {
	case api.Going:
		// Determine if organizer or if there's space in the main list
		isOrganizer := game.OrganizerID == int64(authInfo.UserId)
		if isOrganizer || participantsGroup <= int(game.GameSpotsLeft) {
			// Organizer has priority, or user fits in main list - they're going
			if err := computedStatus.FromParticipationStatusUpdate(api.Going); err != nil {
				http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
				return
			}
			if participantsGroup <= int(game.GameSpotsLeft) {
				// Decrement game spots
				gameSpotsLeft.Int64 = game.GameSpotsLeft - int64(participantsGroup)
				gameSpotsLeft.Valid = true
			} else if isOrganizer {
				// Organizer joining without space: recalculate GameSpotsLeft
				// because the organizer takes spots and may push others to waitlist
				participants, err := querierWithTx.ParticipantsList(r.Context(), db.ParticipantsListParams{
					OrganizerID: game.OrganizerID,
					GameID:      id,
				})
				if err != nil {
					http.Error(w, fmt.Sprintf("failed to list participants: %s", err.Error()), http.StatusInternalServerError)
					return
				}

				// Calculate how many of the existing going participants fit in the main list
				// with the organizer now having priority and taking spots
				mainListCount := int64(participantsGroup) // Start with the organizer's group size

				for _, participant := range participants {
					if !participant.GameParticipant.Going.Valid || !participant.GameParticipant.Going.Bool {
						continue
					}

					pc := int64(1)
					if participant.GameParticipant.Guests.Valid {
						pc += participant.GameParticipant.Guests.Int64
					}

					// Check if this participant fits in the main list with the organizer
					if mainListCount+pc <= int64(game.MaxPlayers) {
						mainListCount += pc
					}
					// Otherwise they stay/go to waitlist
				}

				// GameSpotsLeft = total capacity minus those who fit in the main list
				gameSpotsLeft.Int64 = int64(game.MaxPlayers) - mainListCount
				gameSpotsLeft.Valid = true
			}
		} else {
			// Non-organizer, not enough space: they're waitlisted
			if err := computedStatus.FromParticipationStatus1(api.Waitlisted); err != nil {
				http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
				return
			}
		}

	case api.NotGoing:
		if err := computedStatus.FromParticipationStatusUpdate(api.NotGoing); err != nil {
			http.Error(w, fmt.Sprintf("failed to encode status: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		// We need to determine if they were in the main list to free up spots
		participants, err := querierWithTx.ParticipantsList(r.Context(), db.ParticipantsListParams{
			OrganizerID: game.OrganizerID,
			GameID:      id,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to list participants: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		// Determine current participant segment by simulating read-time status
		goingCount := 0
		spotsFreed := int64(0)
		for _, participant := range participants {
			pc := 1
			if participant.GameParticipant.Guests.Valid {
				pc += int(participant.GameParticipant.Guests.Int64)
			}

			if participant.GameParticipant.Going.Valid && participant.GameParticipant.Going.Bool {
				if goingCount+pc <= int(game.MaxPlayers) {
					// main list - free up spots when this user leaves
					if participant.User.ID == int64(authInfo.UserId) {
						spotsFreed = int64(pc)
						break
					}
					goingCount += pc
				} else {
					// waitlist (unlimited) - no counter to update
					if participant.User.ID == int64(authInfo.UserId) {
						// no counters to update
						break
					}
				}
			} else {
				if participant.User.ID == int64(authInfo.UserId) {
					// already not going
					break
				}
			}
		}

		// If spots were freed, calculate how the remaining participants reorder
		if spotsFreed > 0 {
			// Calculate how many of the remaining going participants fit in the main list
			mainListCount := int64(0)
			for _, participant := range participants {
				// Skip the user who is leaving
				if participant.User.ID == int64(authInfo.UserId) {
					continue
				}

				if !participant.GameParticipant.Going.Valid || !participant.GameParticipant.Going.Bool {
					continue
				}

				pc := int64(1)
				if participant.GameParticipant.Guests.Valid {
					pc += participant.GameParticipant.Guests.Int64
				}

				// Count how many people fit in the main list
				if mainListCount+pc <= int64(game.MaxPlayers) {
					mainListCount += pc
				}
				// Note: we don't count waitlisted people towards spots available
			}

			// GameSpotsLeft = total capacity minus those who fit in the main list
			gameSpotsLeft.Int64 = int64(game.MaxPlayers) - mainListCount
			gameSpotsLeft.Valid = true
		}

		// Clear guests when someone is not going
		guests.Int64 = 0
		guests.Valid = true
		participantsGroup = 1

	default:
		http.Error(w, "invalid status: "+string(req.Status), http.StatusBadRequest)
		return
	}

	if gameSpotsLeft.Valid {
		if err := querierWithTx.GameUpdate(r.Context(), db.GameUpdateParams{
			ID:            id,
			GameSpotsLeft: gameSpotsLeft,
		}); err != nil {
			http.Error(w, fmt.Sprintf("failed to update game spots left: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	}

	if err := querierWithTx.ParticipantsUpsert(r.Context(), db.ParticipantsUpsertParams{
		UserID:         int64(authInfo.UserId),
		GameID:         id,
		Going:          going,
		ConfirmedAt:    confirmedAt,
		GoingUpdatedAt: s.clock.Now(),
		Guests:         guests,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to update participation: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("failed to commit transaction: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	resp := api.GameParticipation{
		GameId: id,
		UserId: strconv.FormatInt(int64(authInfo.UserId), 10),
		Status: computedStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
