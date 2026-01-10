package api

import (
	"strconv"

	"github.com/dmateusp/opengym/db"
	"github.com/dmateusp/opengym/ptr"
	openapi_types "github.com/oapi-codegen/runtime/types"
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
	game.WaitlistSpotsLeft = ptr.Ptr(dbGame.WaitlistSpotsLeft)
	game.GameSpotsLeft = ptr.Ptr(dbGame.GameSpotsLeft)

	game.CreatedAt = dbGame.CreatedAt
	game.UpdatedAt = dbGame.UpdatedAt
}

func (gameDetails *GameDetail) FromDb(dbGame db.GameGetByIdWithOrganizerRow) {
	gameDetails.Game.FromDb(dbGame.Game)
	gameDetails.Organizer.FromDb(dbGame.User)
}

func (user *User) FromDb(dbUser db.User) {
	user.Id = strconv.FormatInt(dbUser.ID, 10)
	user.Email = openapi_types.Email(dbUser.Email)
	user.IsDemo = dbUser.IsDemo

	if dbUser.Name.Valid {
		name := dbUser.Name.String
		user.Name = &name
	}

	if dbUser.Photo.Valid {
		photo := dbUser.Photo.String
		user.Picture = &photo
	}

	// Optional timestamps
	tCreated := dbUser.CreatedAt
	user.CreatedAt = &tCreated
	tUpdated := dbUser.UpdatedAt
	user.UpdatedAt = &tUpdated
}
