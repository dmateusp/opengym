package server

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dmateusp/opengym/api"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleClientId     = flag.String("oauth2.google.client-id", "", "Google client ID")
	googleClientSecret = flag.String("oauth2.google.client-secret", "", "Google client secret")
)

const (
	cookieOAuthState    = "oauth2_state"
	cookieOAuthVerifier = "oauth2_verifier"
)

func (srv *server) GetAuthProviderCallback(w http.ResponseWriter, r *http.Request, provider api.GetAuthProviderCallbackParamsProvider, params api.GetAuthProviderCallbackParams) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (srv *server) GetAuthProviderLogin(w http.ResponseWriter, r *http.Request, provider api.GetAuthProviderLoginParamsProvider) {
	redirectUrl, err := url.JoinPath(*baseUrl, "oauth2", string(provider), "callback")
	if err != nil {
		http.Error(w, fmt.Sprintf("could not construct the callback url: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var oauthConfig *oauth2.Config
	switch provider {
	case api.Google:
		oauthConfig = &oauth2.Config{
			ClientID:     *googleClientId,
			ClientSecret: *googleClientSecret,
			RedirectURL:  redirectUrl,
			Scopes:       []string{},
			Endpoint:     google.Endpoint,
		}
	default:
		http.Error(w, fmt.Sprintf("provider %s is not supported", provider), http.StatusNotImplemented)
		return
	}

	state := uuid.NewString()
	verifier := oauth2.GenerateVerifier()

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     cookieOAuthState,
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		},
	)
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     cookieOAuthVerifier,
			Value:    verifier,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		},
	)

	http.Redirect(w, r, oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}
