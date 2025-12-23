package server

import (
	"net/http"

	"github.com/dmateusp/opengym/api"
)

func (srv *server) GetAuthProviderCallback(w http.ResponseWriter, r *http.Request, provider api.GetAuthProviderCallbackParamsProvider, params api.GetAuthProviderCallbackParams) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (srv *server) GetAuthProviderLogin(w http.ResponseWriter, r *http.Request, provider api.GetAuthProviderLoginParamsProvider) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}
