package middleware

import "context"

// ctxKey is a private type to prevent collisions with other packages.
type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
	ctxKeyJTI
)

// WithUserID stores the authenticated user ID in the context.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyUserID, id)
}

// UserIDFromContext retrieves the authenticated user ID from the context.
// Returns empty string if not set.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// WithJTI stores the JWT ID (jti) in the context.
func WithJTI(ctx context.Context, jti string) context.Context {
	return context.WithValue(ctx, ctxKeyJTI, jti)
}

// JTIFromContext retrieves the JWT ID from the context.
// Returns empty string if not set.
func JTIFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyJTI).(string)
	return v
}
