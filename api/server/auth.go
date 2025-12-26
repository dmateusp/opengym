package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dmateusp/opengym/api"
	"github.com/dmateusp/opengym/auth"
	"github.com/dmateusp/opengym/db"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"

	_ "modernc.org/sqlite"
)

var (
	googleClientId     = flag.String("auth.google.client-id", "", "Google client ID")
	googleClientSecret = flag.String("auth.google.client-secret", "", "Google client secret")
)

const (
	cookieOAuthState    = "oauth2_state"
	cookieOAuthVerifier = "oauth2_verifier"
)

func computeStateSignature(nonce string, exp int64) []byte {
	mac := hmac.New(sha256.New, []byte(auth.GetSigningSecret()))
	mac.Write([]byte(nonce))
	mac.Write([]byte(":"))
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return mac.Sum(nil)
}

func makeStateToken(ttl time.Duration) (string, int64) {
	nonce := uuid.NewString()
	exp := time.Now().Add(ttl).Unix()
	sig := base64.RawURLEncoding.EncodeToString(computeStateSignature(nonce, exp))
	return nonce + ":" + strconv.FormatInt(exp, 10) + ":" + sig, exp
}

func verifyStateToken(token string) bool {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return false
	}
	nonce := parts[0]
	expStr := parts[1]
	sigStr := parts[2]

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix() > exp {
		return false
	}

	expected := computeStateSignature(nonce, exp)
	provided, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil {
		return false
	}
	return hmac.Equal(expected, provided)
}

func (srv *server) GetApiAuthProviderCallback(w http.ResponseWriter, r *http.Request, provider api.GetApiAuthProviderCallbackParamsProvider, params api.GetApiAuthProviderCallbackParams) {
	if params.Error != nil {
		errorMsg := "authorization failed: " + *params.Error
		if params.ErrorDescription != nil {
			errorMsg += ", " + *params.ErrorDescription
		}
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	redirectUrl, err := url.JoinPath(*baseUrl, "auth", string(provider), "callback")
	if err != nil {
		http.Error(w, fmt.Sprintf("could not construct the callback url: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	// State verfication
	if params.State == nil || *params.State == "" {
		http.Error(w, "missing oauth state", http.StatusBadRequest)
		return
	}

	stateCookie, err := r.Cookie(cookieOAuthState)
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}

	if stateCookie.Value != *params.State {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	if !verifyStateToken(*params.State) {
		http.Error(w, "invalid or expired oauth state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieOAuthState,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	if params.Code == nil {
		http.Error(w, "missing oauth code", http.StatusBadRequest)
		return
	}

	verifierCookie, err := r.Cookie(cookieOAuthVerifier)
	if err != nil {
		http.Error(w, "missing verifier cookie", http.StatusBadRequest)
		return
	}

	var upsertDbUser db.UserUpsertRetuningIdParams
	switch provider {
	case api.GetApiAuthProviderCallbackParamsProviderGoogle:
		oauthConfig := &oauth2.Config{
			Endpoint:     google.Endpoint,
			ClientID:     *googleClientId,
			ClientSecret: *googleClientSecret,
			RedirectURL:  redirectUrl,
		}

		token, err := oauthConfig.Exchange(
			r.Context(),
			*params.Code,
			oauth2.AccessTypeOnline,
			oauth2.VerifierOption(verifierCookie.Value),
		)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to exchange code for token: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		peopleService, err := people.NewService(r.Context(), option.WithTokenSource(oauth2.StaticTokenSource(token)))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to instantiate people service to get basic information from Google: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		person, err := peopleService.People.Get("people/me").PersonFields("names,photos,emailAddresses").Do()
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get basic information from Google: %s", err.Error()), http.StatusInternalServerError)
			return
		}

		if len(person.EmailAddresses) == 0 {
			http.Error(w, "no email address found", http.StatusBadRequest)
			return
		}
		upsertDbUser.Email = person.EmailAddresses[0].Value
		if len(person.Names) > 0 {
			upsertDbUser.Name = sql.NullString{
				String: person.Names[0].DisplayName,
				Valid:  true,
			}
		}
		if len(person.Photos) > 0 {
			upsertDbUser.Photo = sql.NullString{
				String: person.Photos[0].Url,
				Valid:  true,
			}
		}

	default:
		http.Error(w, fmt.Sprintf("provider %s is not supported", provider), http.StatusBadRequest)
		return
	}

	userId, err := srv.querier.UserUpsertRetuningId(
		r.Context(),
		upsertDbUser,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to upsert user: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	jwtToken := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    auth.Issuer,
			Subject:   strconv.FormatInt(userId, 10),
			ExpiresAt: jwt.NewNumericDate(now.Add(4 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	)
	signedJwt, err := jwtToken.SignedString([]byte(auth.GetSigningSecret()))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to sign jwt token: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     auth.JTWCookie,
			Value:    signedJwt,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int((4 * time.Hour).Seconds()),
		},
	)

	// TODO: redirect to the page it came from
}

func (srv *server) GetApiAuthProviderLogin(w http.ResponseWriter, r *http.Request, provider api.GetApiAuthProviderLoginParamsProvider) {
	redirectUrl, err := url.JoinPath(*baseUrl, "auth", string(provider), "callback")
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
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.profile",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		}
	default:
		http.Error(w, fmt.Sprintf("provider %s is not supported", provider), http.StatusBadRequest)
		return
	}

	state, exp := makeStateToken(5 * time.Minute)
	verifier := oauth2.GenerateVerifier()

	isHTTPS := false
	if parsed, err := url.Parse(*baseUrl); err == nil && parsed.Scheme == "https" {
		isHTTPS = true
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     cookieOAuthState,
			Value:    state,
			Path:     "/",
			HttpOnly: true,
			Secure:   isHTTPS,
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(exp, 0),
			MaxAge:   int((5 * time.Minute).Seconds()),
		},
	)
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     cookieOAuthVerifier,
			Value:    verifier,
			Path:     "/",
			HttpOnly: true,
			Secure:   isHTTPS,
			SameSite: http.SameSiteLaxMode,
		},
	)

	http.Redirect(w, r, oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline, oauth2.S256ChallengeOption(verifier)), http.StatusFound)
}
