package api

import (
	"github.com/dmateusp/opengym/db"
	"github.com/dmateusp/opengym/ptr"
)

func (game *Game) FromDb(dbGame db.Game) {
	game.Id = dbGame.ID
	game.OrganizerId = int(dbGame.OrganizerID)
	game.Name = dbGame.Name

	if dbGame.Description.Valid {
		desc := dbGame.Description.String
		game.Description = &desc
	}

	if dbGame.PublishedAt.Valid {
		t := dbGame.PublishedAt.Time
		game.PublishedAt = &t
	}

	game.TotalPriceCents = ptr.Ptr(dbGame.TotalPriceCents)

	if dbGame.Location.Valid {
		loc := dbGame.Location.String
		game.Location = &loc
	}

	if dbGame.StartsAt.Valid {
		t := dbGame.StartsAt.Time
		game.StartsAt = &t
	}

	game.DurationMinutes = ptr.Ptr(dbGame.DurationMinutes)
	game.MaxPlayers = ptr.Ptr(dbGame.MaxPlayers)
	game.MaxWaitlistSize = ptr.Ptr(dbGame.MaxWaitlistSize)
	game.MaxGuestsPerPlayer = ptr.Ptr(dbGame.MaxGuestsPerPlayer)

	game.CreatedAt = dbGame.CreatedAt
	game.UpdatedAt = dbGame.UpdatedAt
}
