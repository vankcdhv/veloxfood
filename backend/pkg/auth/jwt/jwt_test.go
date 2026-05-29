package jwt_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	authjwt "project/pkg/auth/jwt"
)

func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	return priv, &priv.PublicKey
}

func TestIssueAndParse_RoundTrip(t *testing.T) {
	priv, pub := generateTestKeys(t)
	svc := authjwt.NewRS256Service(priv, pub, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	userID := "user-abc-123"
	pair, err := svc.Issue(ctx, userID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if pair.Access == "" || pair.Refresh == "" {
		t.Fatal("empty token strings")
	}
	if pair.JTIAccess == "" || pair.JTIRefresh == "" {
		t.Fatal("empty JTIs")
	}
	if pair.JTIAccess == pair.JTIRefresh {
		t.Fatal("access and refresh JTIs must differ")
	}

	// parse access token
	claims, err := svc.Parse(ctx, pair.Access)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("userID mismatch: got %q want %q", claims.UserID, userID)
	}
	if claims.RegisteredClaims.ID != pair.JTIAccess {
		t.Errorf("JTI mismatch: got %q want %q", claims.RegisteredClaims.ID, pair.JTIAccess)
	}

	// parse refresh token
	claimsR, err := svc.Parse(ctx, pair.Refresh)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	if claimsR.RegisteredClaims.ID != pair.JTIRefresh {
		t.Errorf("refresh JTI mismatch")
	}
}

func TestParse_ExpiredToken(t *testing.T) {
	priv, pub := generateTestKeys(t)
	// Issue with negative TTL → immediately expired
	svc := authjwt.NewRS256Service(priv, pub, -1*time.Second, 7*24*time.Hour)
	ctx := context.Background()

	pair, err := svc.Issue(ctx, "uid")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	_, err = svc.Parse(ctx, pair.Access)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestParse_InvalidSignature(t *testing.T) {
	priv1, _ := generateTestKeys(t)
	_, pub2 := generateTestKeys(t)

	// Sign with priv1, verify with pub2 → should fail
	svc := authjwt.NewRS256Service(priv1, pub2, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	pair, err := svc.Issue(ctx, "uid")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	_, err = svc.Parse(ctx, pair.Access)
	if err == nil {
		t.Fatal("expected error for invalid signature, got nil")
	}
}

func TestIssue_UniqueJTIsEachCall(t *testing.T) {
	priv, pub := generateTestKeys(t)
	svc := authjwt.NewRS256Service(priv, pub, 15*time.Minute, 7*24*time.Hour)
	ctx := context.Background()

	p1, _ := svc.Issue(ctx, "uid")
	p2, _ := svc.Issue(ctx, "uid")

	if p1.JTIAccess == p2.JTIAccess {
		t.Error("JTIs should be unique across calls")
	}
}
