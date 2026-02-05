package auth

import (
	"context"
	"flag"

	"github.com/dmateusp/opengym/flagsecret"
)

const (
	JWTCookie = "opengym_jwt"
	Issuer    = "opengym"
)

var signingSecret flagsecret.Secret

func init() {
	flag.Var(&signingSecret, "auth.signing-secret", "Secret used to sign OAuth2 state and JWTs (if set as a flag, supports file://<path to file>)")
}

func GetSigningSecret() string {
	return signingSecret.Value()
}

type AuthInfo struct {
	UserId int
}

type authInfoCtxKey struct{}

func FromCtx(ctx context.Context) (AuthInfo, bool) {
	authInfo, ok := ctx.Value(authInfoCtxKey{}).(AuthInfo)
	return authInfo, ok
}

func WithAuthInfo(ctx context.Context, authInfo AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoCtxKey{}, authInfo)
}
