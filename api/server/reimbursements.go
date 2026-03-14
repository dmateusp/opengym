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
	"github.com/oapi-codegen/nullable"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func (s *server) GetApiGamesIdReimbursementsParticipantId(w http.ResponseWriter, r *http.Request, id string, participantId string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	userID := int64(authInfo.UserId)
	isOrganizer := game.OrganizerID == userID

	targetParticipantID, err := strconv.ParseInt(participantId, 10, 64)
	if err != nil {
		http.Error(w, "invalid participant_id", http.StatusBadRequest)
		return
	}

	if !isOrganizer && userID != targetParticipantID {
		http.Error(w, "forbidden: only the participant or the organizer can access this record", http.StatusForbidden)
		return
	}

	participant, err := s.querier.ParticipantGetByGameAndUser(r.Context(), db.ParticipantGetByGameAndUserParams{
		GameID: id,
		UserID: targetParticipantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "participant not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve participant: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	response := api.ReimbursementRecord{
		ParticipantId:           participantId,
		GameId:                  id,
		ReimbursementReference:  participant.ReimbursementReference.String,
		CreatedAt:               ptr.Ptr(participant.CreatedAt),
		UpdatedAt:               ptr.Ptr(participant.UpdatedAt),
		ReimbursedAt:            sqlNullTimeToNullable(participant.ReimbursedAt),
		ReimbursementReceivedAt: sqlNullTimeToNullable(participant.ReimbursementReceivedAt),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
	}
}

func (s *server) GetApiGamesIdReimbursements(w http.ResponseWriter, r *http.Request, id string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	userID := int64(authInfo.UserId)
	if game.OrganizerID != userID {
		http.Error(w, "forbidden: only the organizer can view reimbursements", http.StatusForbidden)
		return
	}

	if !game.FrozenAt.Valid || game.FrozenAt.Time.After(s.clock.Now()) {
		http.Error(w, "reimbursements are only available for frozen games", http.StatusBadRequest)
		return
	}

	rows, err := s.querier.ParticipantsList(r.Context(), db.ParticipantsListParams{
		OrganizerID: game.OrganizerID,
		GameID:      id,
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve reimbursements: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	type reimbursementParticipant struct {
		row       db.ParticipantsListRow
		groupSize int64
	}

	billableParticipants := make([]reimbursementParticipant, 0, len(rows))
	totalBillableCount := int64(0)

	for _, row := range rows {
		if !row.GameParticipant.Going.Valid || !row.GameParticipant.Going.Bool {
			continue
		}

		groupSize := int64(1)
		if row.GameParticipant.Guests.Valid {
			groupSize += row.GameParticipant.Guests.Int64
		}

		// Reimbursements apply only to participants that fit in the main list.
		if totalBillableCount+groupSize > game.MaxPlayers {
			continue
		}

		totalBillableCount += groupSize
		billableParticipants = append(billableParticipants, reimbursementParticipant{
			row:       row,
			groupSize: groupSize,
		})
	}

	entries := make([]api.GameReimbursementEntry, 0, len(billableParticipants))
	for _, participant := range billableParticipants {
		row := participant.row
		guests := 0
		if row.GameParticipant.Guests.Valid {
			guests = int(row.GameParticipant.Guests.Int64)
		}
		var name *string
		if row.User.Name.Valid {
			name = &row.User.Name.String
		}
		var picture *string
		if row.User.Photo.Valid {
			picture = &row.User.Photo.String
		}

		amountOwedCents := ceilDiv(game.TotalPriceCents*participant.groupSize, totalBillableCount)

		entries = append(entries, api.GameReimbursementEntry{
			ReimbursementReference: row.GameParticipant.ReimbursementReference.String,
			AmountOwedCents:        amountOwedCents,
			Guests:                 guests,
			Participant: api.User{
				Id:      strconv.FormatInt(row.User.ID, 10),
				Email:   openapi_types.Email(row.User.Email),
				Name:    name,
				Picture: picture,
			},
			ReimbursedAt:            sqlNullTimeToNullable(row.GameParticipant.ReimbursedAt),
			ReimbursementReceivedAt: sqlNullTimeToNullable(row.GameParticipant.ReimbursementReceivedAt),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
	}
}

func (s *server) PutApiGamesIdReimbursements(w http.ResponseWriter, r *http.Request, id string) {
	authInfo, ok := auth.FromCtx(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	if !game.FrozenAt.Valid || game.FrozenAt.Time.After(s.clock.Now()) {
		http.Error(w, "reimbursements are only available for frozen games", http.StatusBadRequest)
		return
	}

	var req api.UpdateReimbursementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
		return
	}

	userID := int64(authInfo.UserId)
	isOrganizer := game.OrganizerID == userID
	participantID := userID

	if isOrganizer {
		organizerReq, err := req.AsUpdateReimbursementRequest0()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
			return
		}

		if organizerReq.ParticipantId == "" || !organizerReq.ReimbursementReceivedAt.IsSpecified() {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		participantID, err = strconv.ParseInt(organizerReq.ParticipantId, 10, 64)
		if err != nil {
			http.Error(w, "invalid participantId", http.StatusBadRequest)
			return
		}

		reimbursementReceivedAt := nullableToNullTime(organizerReq.ReimbursementReceivedAt)
		rowsAffected, err := s.querier.ParticipantUpdateReimbursementReceivedAt(r.Context(), db.ParticipantUpdateReimbursementReceivedAtParams{
			GameID:                  id,
			UserID:                  participantID,
			ReimbursementReceivedAt: reimbursementReceivedAt,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to update reimbursement status: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if rowsAffected == 0 {
			http.Error(w, "participant not found", http.StatusNotFound)
			return
		}
	} else {
		organizerReq, err := req.AsUpdateReimbursementRequest0()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if organizerReq.ParticipantId != "" || organizerReq.ReimbursementReceivedAt.IsSpecified() {
			http.Error(w, "participants cannot set participantId or reimbursement_received_at", http.StatusBadRequest)
			return
		}

		participantReq, err := req.AsUpdateReimbursementRequest1()
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid request body: %s", err.Error()), http.StatusBadRequest)
			return
		}
		if !participantReq.ReimbursedAt.IsSpecified() {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if _, err := s.querier.ParticipantGetByGameAndUser(r.Context(), db.ParticipantGetByGameAndUserParams{
			GameID: id,
			UserID: userID,
		}); err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "forbidden: you are not a participant of this game", http.StatusForbidden)
				return
			}
			http.Error(w, fmt.Sprintf("failed to retrieve participant: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		reimbursedAt := nullableToNullTime(participantReq.ReimbursedAt)
		rowsAffected, err := s.querier.ParticipantUpdateReimbursedAt(r.Context(), db.ParticipantUpdateReimbursedAtParams{
			GameID:       id,
			UserID:       userID,
			ReimbursedAt: reimbursedAt,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to update reimbursement status: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		if rowsAffected == 0 {
			http.Error(w, "forbidden: you are not a participant of this game", http.StatusForbidden)
			return
		}
	}

	participant, err := s.querier.ParticipantGetByGameAndUser(r.Context(), db.ParticipantGetByGameAndUserParams{
		GameID: id,
		UserID: participantID,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "participant not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve participant: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	response := api.ReimbursementRecord{
		ParticipantId:           strconv.FormatInt(participantID, 10),
		GameId:                  id,
		ReimbursementReference:  participant.ReimbursementReference.String,
		CreatedAt:               ptr.Ptr(participant.CreatedAt),
		UpdatedAt:               ptr.Ptr(participant.UpdatedAt),
		ReimbursedAt:            sqlNullTimeToNullable(participant.ReimbursedAt),
		ReimbursementReceivedAt: sqlNullTimeToNullable(participant.ReimbursementReceivedAt),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
	}
}

func nullableToNullTime(value nullable.Nullable[time.Time]) sql.NullTime {
	if value.IsNull() || !value.IsSpecified() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value.MustGet(), Valid: true}
}

func sqlNullTimeToNullable(value sql.NullTime) nullable.Nullable[time.Time] {
	if !value.Valid {
		return nullable.NewNullNullable[time.Time]()
	}
	return nullable.NewNullableWithValue(value.Time)
}

func ceilDiv(numerator int64, denominator int64) int64 {
	if denominator <= 0 {
		return 0
	}
	return (numerator + denominator - 1) / denominator
}
