package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
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
	"github.com/dmateusp/opengym/log"
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

type OAuthState struct {
	Nonce        string `json:"nonce"`         // random UUID
	RedirectPage string `json:"redirect_page"` // relative path to page the user was on when the oauth flow was initiated
}

func computeStateSignature(encodedPayload string, exp int64) []byte {
	mac := hmac.New(sha256.New, []byte(auth.GetSigningSecret()))
	mac.Write([]byte(encodedPayload))
	mac.Write([]byte(":"))
	mac.Write([]byte(strconv.FormatInt(exp, 10)))
	return mac.Sum(nil)
}

func normalizeRedirectPage(raw string) (string, error) {
	if raw == "" {
		return "/", nil
	}

	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid redirect page: %w", err)
	}

	if parsed.IsAbs() || parsed.Host != "" {
		return "", fmt.Errorf("redirect page must be a relative path")
	}

	normalized := parsed.Path
	if parsed.RawQuery != "" {
		normalized += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		normalized += "#" + parsed.Fragment
	}

	return normalized, nil
}

func makeStateToken(ttl time.Duration, redirectPage string) (string, int64, error) {
	normalizedRedirect, err := normalizeRedirectPage(redirectPage)
	if err != nil {
		return "", 0, err
	}

	state := OAuthState{
		Nonce:        uuid.NewString(),
		RedirectPage: normalizedRedirect,
	}

	encodedPayload, err := json.Marshal(state)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal oauth state: %w", err)
	}

	exp := time.Now().Add(ttl).Unix()
	encodedPayloadStr := base64.RawURLEncoding.EncodeToString(encodedPayload)
	sig := base64.RawURLEncoding.EncodeToString(computeStateSignature(encodedPayloadStr, exp))

	return encodedPayloadStr + ":" + strconv.FormatInt(exp, 10) + ":" + sig, exp, nil
}

func parseStateToken(token string) (*OAuthState, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid oauth state format")
	}

	encodedPayload := parts[0]
	expStr := parts[1]
	sigStr := parts[2]

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth state expiry: %w", err)
	}
	if time.Now().Unix() > exp {
		return nil, fmt.Errorf("oauth state expired")
	}

	expected := computeStateSignature(encodedPayload, exp)
	provided, err := base64.RawURLEncoding.DecodeString(sigStr)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth state signature encoding: %w", err)
	}

	if !hmac.Equal(expected, provided) {
		return nil, fmt.Errorf("oauth state signature mismatch")
	}

	decodedPayload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("invalid oauth state payload: %w", err)
	}

	var state OAuthState
	if err := json.Unmarshal(decodedPayload, &state); err != nil {
		return nil, fmt.Errorf("invalid oauth state payload: %w", err)
	}

	if state.Nonce == "" {
		return nil, fmt.Errorf("missing nonce in oauth state")
	}

	normalizedRedirect, err := normalizeRedirectPage(state.RedirectPage)
	if err != nil {
		return nil, err
	}
	state.RedirectPage = normalizedRedirect

	return &state, nil
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

	redirectUrl, err := url.JoinPath(*baseUrl, "api/auth", string(provider), "callback")
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

	state, err := parseStateToken(*params.State)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid or expired oauth state: %s", err.Error()), http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieOAuthState,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   stateCookie.Secure,
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

	http.SetCookie(w, &http.Cookie{
		Name:     cookieOAuthVerifier,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   verifierCookie.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

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

	http.Redirect(w, r, state.RedirectPage, http.StatusFound)
}

func (srv *server) GetApiAuthProviderLogin(w http.ResponseWriter, r *http.Request, provider api.GetApiAuthProviderLoginParamsProvider) {
	redirectUrl, err := url.JoinPath(*baseUrl, "api/auth", string(provider), "callback")
	if err != nil {
		http.Error(w, fmt.Sprintf("could not construct the callback url: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var oauthConfig *oauth2.Config
	switch provider {
	case api.GetApiAuthProviderLoginParamsProviderGoogle:
		if *googleClientId == "" || *googleClientSecret == "" {
			log.FromCtx(r.Context()).ErrorContext(r.Context(), "Google client ID or client secret not set")
			http.Error(w, "the back-end is not configured to handle Google auth", http.StatusBadRequest)
			return
		}
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

	state, exp, err := makeStateToken(5*time.Minute, r.URL.Query().Get("redirect_page"))
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid redirect page: %s", err.Error()), http.StatusBadRequest)
		return
	}
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

func (srv *server) GetApiAuthMe(w http.ResponseWriter, r *http.Request) {
	jwtCookie, err := r.Cookie(auth.JTWCookie)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jwtToken, err := jwt.ParseWithClaims(jwtCookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(auth.GetSigningSecret()), nil
	}, jwt.WithIssuer(auth.Issuer), jwt.WithExpirationRequired(), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sub, err := jwtToken.Claims.GetSubject()
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId, err := strconv.Atoi(sub)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	dbUser, err := srv.querier.UserGetById(r.Context(), int64(userId))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, fmt.Sprintf("failed to retrieve user: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	var apiUser api.User
	apiUser.FromDb(dbUser)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiUser); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %s", err.Error()), http.StatusInternalServerError)
		return
	}
}
