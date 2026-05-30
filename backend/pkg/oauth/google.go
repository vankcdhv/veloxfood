// Package oauth provides Google OAuth 2.0 (Authorization Code) helpers.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"project/pkg/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const userInfoEndpoint = "https://www.googleapis.com/oauth2/v3/userinfo"

// GoogleUser is the subset of Google userinfo we consume.
type GoogleUser struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// Client wraps the configured oauth2 flow.
type Client struct {
	cfg *oauth2.Config
}

// NewClient builds a Google OAuth client. Returns nil if not configured.
func NewClient(cfg config.GoogleOAuthConfig) *Client {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil
	}
	return &Client{cfg: &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}}
}

// AuthCodeURL builds the consent-screen redirect URL with the given state.
func (c *Client) AuthCodeURL(state string) string {
	return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// Exchange swaps an authorization code for a token and fetches the user profile.
func (c *Client) Exchange(ctx context.Context, code string) (*GoogleUser, error) {
	tok, err := c.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth: exchange code: %w", err)
	}

	httpClient := c.cfg.Client(ctx, tok)
	resp, err := httpClient.Get(userInfoEndpoint)
	if err != nil {
		return nil, fmt.Errorf("oauth: fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("oauth: userinfo status %d: %s", resp.StatusCode, string(body))
	}

	var u GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("oauth: decode userinfo: %w", err)
	}
	if u.Sub == "" || u.Email == "" {
		return nil, fmt.Errorf("oauth: incomplete userinfo")
	}
	return &u, nil
}
