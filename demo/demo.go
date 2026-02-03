package demo

import (
	"context"
	"database/sql"
	"flag"
	"fmt"

	"github.com/dmateusp/opengym/db"
	"github.com/dmateusp/opengym/log"
	"github.com/google/uuid"
)

var (
	dbPath        = flag.String("demo.db.path", "./opengym_demo.db", "Database path for the demo instance")
	signingSecret = flag.String("demo.auth.signing-secret", "", "Secret used to sign JWTs, keep this secret different from the non-demo equivalent")
	demoMode      = flag.Bool("demo.enabled", false, "Run in demo mode")
)

const (
	DemoIssuer    = "opengym-demo"
	DemoJWTCookie = "opengym_demo_jwt"
)

func GetDemoDbPath() string {
	return *dbPath
}

func GetDemoSigningSecret() string {
	return *signingSecret
}

func GetDemoMode() bool {
	return *demoMode
}

func SetUpDemoDatabase(ctx context.Context, dbConn *sql.DB, querier db.QuerierWithTxSupport) error {
	tx, err := dbConn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	querier = querier.WithTx(tx)

	testUsers := []db.UserUpsertRetuningIdParams{
		{
			Name:  sql.NullString{String: "John Doe", Valid: true},
			Email: "john.doe@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Jane Doe", Valid: true},
			Email: "jane.doe@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Jim Beam", Valid: true},
			Email: "jim.beam@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Alice Smith", Valid: true},
			Email: "alice.smith@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Bob Johnson", Valid: true},
			Email: "bob.johnson@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Charlie Brown", Valid: true},
			Email: "charlie.brown@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Dana White", Valid: true},
			Email: "dana.white@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Evan Green", Valid: true},
			Email: "evan.green@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "Fiona Black", Valid: true},
			Email: "fiona.black@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
		{
			Name:  sql.NullString{String: "George Miller", Valid: true},
			Email: "george.miller@example.com",
			Photo: sql.NullString{String: "https://api.images.cat/300/300/" + uuid.NewString(), Valid: true},
		},
	}

	logger := log.FromCtx(ctx)

	logger.InfoContext(ctx, "Setting up the demo database")

	for _, t := range testUsers {
		t.IsDemo = true
		userId, err := querier.UserUpsertRetuningId(ctx, t)
		if err != nil {
			return fmt.Errorf("failed to upsert user: %w", err)
		}
		logger.InfoContext(ctx, "Demo user upserted", "user_id", userId)
	}

	return tx.Commit()
}
