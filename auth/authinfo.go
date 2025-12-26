package auth

import (
	"context"
	"flag"
)

const (
	JTWCookie = "opengym_jwt"
	Issuer    = "opengym"
)

var signingSecret = flag.String("auth.signing-secret", "", "Secret used to sign OAuth2 state and JWTs")

func GetSigningSecret() string {
	return *signingSecret
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
