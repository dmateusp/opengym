package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/db"
	"github.com/dmateusp/opengym/flagfromenv"
	"github.com/dmateusp/opengym/log"

	"github.com/lmittmann/tint"
)

var (
	serverAddr = flag.String("server-addr", ":8080", "server address")
	dbPath     = flag.String("db-path", "./opengym.db", "database path")
)

func main() {
	ctx := context.Background()
	w := os.Stderr
	// Create a new logger
	logger := slog.New(tint.NewHandler(w, nil))
	// Set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
		}),
	))

	flag.Parse()

	err := flagfromenv.Parse("OPENGYM")
	if err != nil {
		logger.ErrorContext(ctx, "Failed to parse flags from environment", "error", err)
		os.Exit(1)
	}

	logger.InfoContext(ctx, "Starting opengym server", slog.String("server_addr", *serverAddr))

	dbConn, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to open database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	err = http.ListenAndServe(*serverAddr, api.HandlerWithOptions(server.NewServer(db.New(dbConn)), api.StdHTTPServerOptions{
		Middlewares: []api.MiddlewareFunc{ // Middleware is executed last to first
			log.LogRequestsAndResponsesMiddleware(),
			log.AddLoggerToContextMiddleware(logger), // this is the one that executes first
		},
	}))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to start server", "error", err)
		os.Exit(1)
	}
}
