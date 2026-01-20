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
	organizerPresentGoing := false

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
				if row.User.ID == game.OrganizerID {
					organizerPresentGoing = true
				}
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
			UpdatedAt: ptr.Ptr(row.GameParticipant.UpdatedAt),
			Guests:    guests,
		})
	}

	// If organizer is going, overflow past waitlist becomes not-going for the last overflow group
	if organizerPresentGoing {
		maxTotal := game.MaxPlayers + game.MaxWaitlistSize
		totalCount := int64(0)
		for i := range participants {
			// Recompute group size from participants[i].Guests
			pc := int64(1)
			if participants[i].Guests > 0 {
				pc += int64(participants[i].Guests)
			}
			status, _ := participants[i].Status.AsParticipationStatusUpdate()
			if status == api.Going {
				totalCount += pc
			} else {
				// treat waitlisted as going for capacity tally
				if _, err := participants[i].Status.AsParticipationStatus1(); err == nil {
					totalCount += pc
				}
			}
			if totalCount > maxTotal {
				var s api.ParticipationStatus
				_ = s.FromParticipationStatusUpdate(api.NotGoing)
				participants[i].Status = s
				break
			}
		}
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

	var (
		gameSpotsLeft,
		waitlistSpotsLeft sql.NullInt64
	)
	// Reject if the game doesn't have enough spots left
	switch req.Status {
	case api.Going:
		if participantsGroup <= int(game.GameSpotsLeft) {
			gameSpotsLeft.Int64 = game.GameSpotsLeft - int64(participantsGroup)
			gameSpotsLeft.Valid = true
		} else if participantsGroup <= int(game.WaitlistSpotsLeft) {
			waitlistSpotsLeft.Int64 = game.WaitlistSpotsLeft - int64(participantsGroup)
			waitlistSpotsLeft.Valid = true
		} else {
			// Organizer can always join even if both are full; no spots counters change
			if game.OrganizerID != int64(authInfo.UserId) {
				http.Error(w, "not enough spots left", http.StatusBadRequest)
				return
			}
		}
	case api.NotGoing: // We have to figure out if we're freeing spots in the "going" list, waitlist, or if the player was already in the "not going" list
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
		waitlistCount := 0
		for _, participant := range participants {
			pc := 1
			if participant.GameParticipant.Guests.Valid {
				pc += int(participant.GameParticipant.Guests.Int64)
			}

			if participant.GameParticipant.Going.Valid && participant.GameParticipant.Going.Bool {
				if goingCount+pc <= int(game.MaxPlayers) {
					// main list
					if participant.User.ID == int64(authInfo.UserId) {
						gameSpotsLeft.Int64 = game.GameSpotsLeft + int64(pc)
						gameSpotsLeft.Valid = true
						break
					}
					goingCount += pc
				} else if waitlistCount+pc <= int(game.MaxWaitlistSize) {
					// waitlist
					if participant.User.ID == int64(authInfo.UserId) {
						waitlistSpotsLeft.Int64 = game.WaitlistSpotsLeft + int64(pc)
						waitlistSpotsLeft.Valid = true
						break
					}
					waitlistCount += pc
				} else {
					// overflow beyond waitlist; considered not-going already
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

		// Clear guests when someone is not going
		guests.Int64 = 0
		guests.Valid = true
		participantsGroup = 1

	default:
		http.Error(w, "invalid status: "+string(req.Status), http.StatusBadRequest)
		return
	}

	if gameSpotsLeft.Valid || waitlistSpotsLeft.Valid {
		if err := querierWithTx.GameUpdate(r.Context(), db.GameUpdateParams{
			ID:                id,
			GameSpotsLeft:     gameSpotsLeft,
			WaitlistSpotsLeft: waitlistSpotsLeft,
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
