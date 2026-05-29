package jwt

import gojwt "github.com/golang-jwt/jwt/v5"

// Claims is the JWT payload. UserID is the subject; jti comes from RegisteredClaims.ID.
type Claims struct {
	gojwt.RegisteredClaims
	UserID string `json:"uid"`
}
