package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/api/server"
	"github.com/dmateusp/opengym/log"

	"github.com/lmittmann/tint"
)

var (
	serverAddr = flag.String("server-addr", ":8080", "server address")
)

func main() {
	flag.Parse()

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

	ctx := context.Background()
	logger.InfoContext(ctx, "Starting opengym server", slog.String("server_addr", *serverAddr))

	err := http.ListenAndServe(*serverAddr, api.HandlerWithOptions(server.NewServer(), api.StdHTTPServerOptions{
		Middlewares: []api.MiddlewareFunc{ // Middleware is executed last to first
			log.LogRequestsAndResponsesMiddleware(),
			log.AddLoggerToContextMiddleware(logger), // this is the one that executes first
		},
	}))
	if err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
