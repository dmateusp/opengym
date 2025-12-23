package server

import "github.com/dmateusp/opengym/api"

type server struct{}

func NewServer() *server {
	return &server{}
}

var _ api.ServerInterface = (*server)(nil)
