package server

import (
	"flag"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/db"
)

var (
	baseUrl = flag.String("base-url", "http://localhost:8080", "base url of the server")
)

type server struct {
	querier db.Querier
}

func NewServer(querier db.Querier) *server {
	return &server{
		querier: querier,
	}
}

var _ api.ServerInterface = (*server)(nil)
