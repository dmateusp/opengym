package server

import (
	"flag"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
)

var (
	baseUrl         = flag.String("base-url", "http://localhost:8080", "base url of the server")
	frontendBaseUrl = flag.String("frontend.base-url", "http://localhost:5173", "base url of the frontend")
)

type server struct {
	querier                     db.QuerierWithTxSupport
	randomAlphanumericGenerator RandomAlphanumericGenerator
	clock                       clock.Clock
}

func NewServer(
	querier db.QuerierWithTxSupport,
	randomAlphanumericGenerator RandomAlphanumericGenerator,
	clock clock.Clock,
) *server {
	return &server{
		querier:                     querier,
		randomAlphanumericGenerator: randomAlphanumericGenerator,
		clock:                       clock,
	}
}

var _ api.ServerInterface = (*server)(nil)
