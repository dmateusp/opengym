package server

import (
	"flag"

	"github.com/dmateusp/opengym/api"
)

var (
	baseUrl = flag.String("base-url", "http://localhost:8080", "base url of the server")
)

type server struct{}

func NewServer() *server {
	return &server{}
}

var _ api.ServerInterface = (*server)(nil)
