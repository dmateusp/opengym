package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/cors"
	"github.com/dmateusp/opengym/db"
	"github.com/dmateusp/opengym/demo"
	"github.com/dmateusp/opengym/flagfromenv"
	"github.com/dmateusp/opengym/log"

	"github.com/lmittmann/tint"
)

var (
	serverAddr    = flag.String("server-addr", ":8080", "server address")
	dbPath        = flag.String("db-path", "./opengym.db", "database path")
	serveFrontend = flag.Bool("serve-frontend", false, "serve frontend from frontend/dist")
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

	// Customize help output to show environment variable overrides
	flag.Usage = func() {
		replacer := strings.NewReplacer(".", "__", "-", "_")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nFlags can be overridden by environment variables with the prefix OPENGYM_\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Example: -server-addr can be set via OPENGYM_SERVER_ADDR\n\n")
		flag.VisitAll(func(f *flag.Flag) {
			envVar := "OPENGYM_" + replacer.Replace(strings.ToUpper(f.Name))
			fmt.Fprintf(flag.CommandLine.Output(), "  -%s\n", f.Name)
			fmt.Fprintf(flag.CommandLine.Output(), "    \t%s (default: %q)\n", f.Usage, f.DefValue)
			fmt.Fprintf(flag.CommandLine.Output(), "    \tEnvironment: %s\n", envVar)
		})
	}

	flag.Parse()

	err := flagfromenv.Parse("OPENGYM")
	if err != nil {
		logger.ErrorContext(ctx, "Failed to parse flags from environment", "error", err)
		os.Exit(1)
	}

	if !demo.GetDemoMode() && auth.GetSigningSecret() == "" {
		logger.ErrorContext(ctx, "Please set a signing secret with the flag -auth.signing-secret, using the output of `openssl rand -hex 32` for example (save it somewhere)")
		os.Exit(1)
	}

	logger.InfoContext(ctx, "Starting opengym server", slog.String("server_addr", *serverAddr))

	dbConnPath := *dbPath
	if demo.GetDemoMode() {
		logger.InfoContext(ctx, "Demo mode is enabled, some demo users will be populated in the database and user impersonation is turned ON")
		dbConnPath = demo.GetDemoDbPath()
		if demo.GetDemoSigningSecret() == "" {
			logger.ErrorContext(ctx, "Please set a signing secret with the flag -demo.auth.signing-secret, using the output of `openssl rand -hex 32` for example (save it somewhere)")
			os.Exit(1)
		}
	}

	dbConn, err := sql.Open("sqlite", dbConnPath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to open database", "error", err)
		os.Exit(1)
	}
	defer dbConn.Close()

	querier := db.New(dbConn)
	if demo.GetDemoMode() {
		err = demo.SetUpDemoDatabase(ctx, dbConn, db.NewQuerierWrapper(querier))
		if err != nil {
			logger.ErrorContext(ctx, "Failed to set up demo database", "error", err)
			os.Exit(1)
		}
	}

	// Create the API handler with auth and logging middleware
	apiHandler := api.HandlerWithOptions(server.NewServer(db.NewQuerierWrapper(querier), server.NewRandomAlphanumericGenerator(), clock.RealClock{}, dbConn), api.StdHTTPServerOptions{
		Middlewares: []api.MiddlewareFunc{ // Middleware is executed last to first
			auth.AuthMiddleware,
			log.LogRequestsAndResponsesMiddleware,
			log.AddLoggerToContextMiddleware(logger), // runs first
		},
	})

	// Wrap entire handler with CORS middleware to handle OPTIONS before routing
	handler := cors.CORSMiddleware(apiHandler)

	finalHandler := handler
	if *serveFrontend {
		logger.InfoContext(ctx, "Serving frontend from frontend/dist")
		mux := http.NewServeMux()
		mux.Handle("/api/", handler)
		mux.Handle("/", http.FileServer(http.Dir("frontend/dist")))
		finalHandler = mux
	}

	err = http.ListenAndServe(*serverAddr, finalHandler)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to start server", "error", err)
		os.Exit(1)
	}
}
