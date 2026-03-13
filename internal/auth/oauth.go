package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/justestif/specto/internal/core"
)

var (
	ErrMissingOAuthClient = errors.New("no OAuth client configured for plugin")
	ErrTokenExchange      = errors.New("OAuth token exchange failed")
	ErrTokenRefresh       = errors.New("OAuth token refresh failed")
)

// OAuthClientCredentials holds client_id and client_secret for an OAuth provider.
type OAuthClientCredentials struct {
	ClientID     string
	ClientSecret string
}

// TokenResponse holds the parsed response from an OAuth token endpoint.
type TokenResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds until access token expires
}

// OAuthService handles OAuth authorization URL building, token exchange,
// and token refresh. It is provider-agnostic; individual plugin OAuthConfigs
// supply the provider-specific URLs and scopes.
type OAuthService struct {
	HTTPClient *http.Client
	BaseURL    string                            // server's public base URL
	Clients    map[string]OAuthClientCredentials // keyed by plugin name
}

// NewOAuthService creates an OAuthService. If httpClient is nil,
// a default client with a 30-second timeout is used.
func NewOAuthService(baseURL string, clients map[string]OAuthClientCredentials, httpClient *http.Client) *OAuthService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if clients == nil {
		clients = make(map[string]OAuthClientCredentials)
	}
	return &OAuthService{
		HTTPClient: httpClient,
		BaseURL:    baseURL,
		Clients:    clients,
	}
}

// GenerateState produces a cryptographically random, URL-safe state token
// for CSRF protection during the OAuth flow.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating OAuth state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// RedirectURI returns the callback URL for the given plugin.
func (s *OAuthService) RedirectURI(pluginName string) string {
	return fmt.Sprintf("%s/api/v1/plugins/%s/callback", s.BaseURL, pluginName)
}

// BuildAuthURL constructs the full OAuth authorization URL with all required
// query parameters: client_id, redirect_uri, response_type, scope, and state.
func (s *OAuthService) BuildAuthURL(pluginName string, cfg *core.OAuthConfig, state string) (string, error) {
	creds, ok := s.Clients[pluginName]
	if !ok {
		return "", ErrMissingOAuthClient
	}

	u, err := url.Parse(cfg.AuthURL)
	if err != nil {
		return "", fmt.Errorf("parsing auth URL: %w", err)
	}

	q := u.Query()
	q.Set("client_id", creds.ClientID)
	q.Set("redirect_uri", s.RedirectURI(pluginName))
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(cfg.Scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ExchangeCode exchanges an authorization code for access and refresh tokens.
func (s *OAuthService) ExchangeCode(pluginName string, cfg *core.OAuthConfig, code string) (*TokenResponse, error) {
	creds, ok := s.Clients[pluginName]
	if !ok {
		return nil, ErrMissingOAuthClient
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.RedirectURI(pluginName)},
		"client_id":     {creds.ClientID},
		"client_secret": {creds.ClientSecret},
	}

	return s.postTokenRequest(cfg.TokenURL, data)
}

// RefreshToken exchanges a refresh token for a new access token (and
// optionally a new refresh token).
func (s *OAuthService) RefreshToken(pluginName string, cfg *core.OAuthConfig, refreshToken string) (*TokenResponse, error) {
	creds, ok := s.Clients[pluginName]
	if !ok {
		return nil, ErrMissingOAuthClient
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {creds.ClientID},
		"client_secret": {creds.ClientSecret},
	}

	return s.postTokenRequest(cfg.TokenURL, data)
}

// postTokenRequest sends a POST to the token endpoint and parses the response.
func (s *OAuthService) postTokenRequest(tokenURL string, data url.Values) (*TokenResponse, error) {
	resp, err := s.HTTPClient.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("posting token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d, body: %s", ErrTokenExchange, resp.StatusCode, string(body))
	}

	var raw struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	if raw.Error != "" {
		return nil, fmt.Errorf("%w: %s: %s", ErrTokenExchange, raw.Error, raw.ErrorDesc)
	}

	if raw.AccessToken == "" {
		return nil, fmt.Errorf("%w: empty access_token in response", ErrTokenExchange)
	}

	return &TokenResponse{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		ExpiresIn:    raw.ExpiresIn,
	}, nil
}
