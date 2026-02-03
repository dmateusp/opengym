package server

import (
	"database/sql"
	"flag"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/clock"
	"github.com/dmateusp/opengym/db"
)

var (
	baseUrl         = flag.String("base-url", "http://localhost:8080", "base url of the server")
	frontendBaseUrl = flag.String("frontend.base-url", "http://localhost:5173", "base url of the frontend")
)

func GetBaseUrl() string {
	return *baseUrl
}

type server struct {
	querier                     db.QuerierWithTxSupport
	randomAlphanumericGenerator RandomAlphanumericGenerator
	clock                       clock.Clock
	dbConn                      *sql.DB
}

func NewServer(
	querier db.QuerierWithTxSupport,
	randomAlphanumericGenerator RandomAlphanumericGenerator,
	clock clock.Clock,
	dbConn *sql.DB,
) *server {
	return &server{
		querier:                     querier,
		randomAlphanumericGenerator: randomAlphanumericGenerator,
		clock:                       clock,
		dbConn:                      dbConn,
	}
}

var _ api.ServerInterface = (*server)(nil)
