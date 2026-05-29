package jwt

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenPair holds both access and refresh tokens along with their JTIs and TTLs.
type TokenPair struct {
	Access     string
	Refresh    string
	JTIAccess  string
	JTIRefresh string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// JWTService signs and parses RS256 JWT tokens.
type JWTService interface {
	Issue(ctx context.Context, userID string) (TokenPair, error)
	Parse(ctx context.Context, token string) (*Claims, error)
}

type rs256Service struct {
	priv       *rsa.PrivateKey
	pub        *rsa.PublicKey
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewRS256Service creates a JWTService backed by RSA 2048 keys.
func NewRS256Service(priv *rsa.PrivateKey, pub *rsa.PublicKey, accessTTL, refreshTTL time.Duration) JWTService {
	return &rs256Service{
		priv:       priv,
		pub:        pub,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

func (s *rs256Service) Issue(_ context.Context, userID string) (TokenPair, error) {
	jtiA := uuid.New().String()
	jtiR := uuid.New().String()
	now := time.Now()

	accessToken, err := s.sign(userID, jtiA, now, s.accessTTL)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign access token: %w", err)
	}
	refreshToken, err := s.sign(userID, jtiR, now, s.refreshTTL)
	if err != nil {
		return TokenPair{}, fmt.Errorf("sign refresh token: %w", err)
	}

	return TokenPair{
		Access:     accessToken,
		Refresh:    refreshToken,
		JTIAccess:  jtiA,
		JTIRefresh: jtiR,
		AccessTTL:  s.accessTTL,
		RefreshTTL: s.refreshTTL,
	}, nil
}

func (s *rs256Service) sign(userID, jti string, now time.Time, ttl time.Duration) (string, error) {
	claims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(ttl)),
		},
		UserID: userID,
	}
	return gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims).SignedString(s.priv)
}

func (s *rs256Service) Parse(_ context.Context, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := gojwt.ParseWithClaims(tokenStr, claims, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.pub, nil
	})
	if err != nil {
		if errors.Is(err, gojwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

var (
	ErrTokenInvalid = errors.New("token invalid")
	ErrTokenExpired = errors.New("token expired")
)
