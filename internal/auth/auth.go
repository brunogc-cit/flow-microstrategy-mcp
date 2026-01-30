package auth

import "context"

type contextKey string

const (
	basicAuthUserKey contextKey = "basicAuthUser"
	basicAuthPassKey contextKey = "basicAuthPass"
	bearerTokenKey   contextKey = "bearerToken"
	apiTokenAuthKey  contextKey = "apiTokenAuth"
)

// WithBasicAuth adds basic auth credentials to the context
func WithBasicAuth(ctx context.Context, user, pass string) context.Context {
	ctx = context.WithValue(ctx, basicAuthUserKey, user)
	ctx = context.WithValue(ctx, basicAuthPassKey, pass)
	return ctx
}

// GetBasicAuthCredentials retrieves basic auth credentials from the context
func GetBasicAuthCredentials(ctx context.Context) (string, string, bool) {
	user, okUser := ctx.Value(basicAuthUserKey).(string)
	pass, okPass := ctx.Value(basicAuthPassKey).(string)
	return user, pass, okUser && okPass
}

// WithBearerToken adds bearer token to the context
func WithBearerToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, bearerTokenKey, token)
}

// GetBearerToken retrieves bearer token from the context
func GetBearerToken(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(bearerTokenKey).(string)
	return token, ok
}

// WithAPITokenAuth marks the context as authenticated via API token
// This indicates the server should use configured Neo4j credentials
func WithAPITokenAuth(ctx context.Context) context.Context {
	return context.WithValue(ctx, apiTokenAuthKey, true)
}

// IsAPITokenAuth checks if the request was authenticated via API token
func IsAPITokenAuth(ctx context.Context) bool {
	val, ok := ctx.Value(apiTokenAuthKey).(bool)
	return ok && val
}

// HasAuth checks if either basic auth, bearer token, or API token auth is present in the context
func HasAuth(ctx context.Context) bool {
	_, _, okBasic := GetBasicAuthCredentials(ctx)
	_, okBearer := GetBearerToken(ctx)
	okAPIToken := IsAPITokenAuth(ctx)
	return okBasic || okBearer || okAPIToken
}
