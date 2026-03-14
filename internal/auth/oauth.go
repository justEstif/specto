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

// AppAuthRedirectURI returns the callback URL for app-level OAuth login.
func (s *OAuthService) AppAuthRedirectURI(provider string) string {
	return fmt.Sprintf("%s/auth/%s/callback", s.BaseURL, provider)
}

// appAuthConfigs defines the OAuth configurations for app-level login providers.
var appAuthConfigs = map[string]*core.OAuthConfig{
	"google": {
		ProviderName: "Google",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"openid", "email", "profile"},
	},
	"github": {
		ProviderName: "GitHub",
		AuthURL:      "https://github.com/login/oauth/authorize",
		TokenURL:     "https://github.com/login/oauth/access_token",
		Scopes:       []string{"read:user", "user:email"},
	},
}

// AppAuthConfig returns the OAuth config for an app-level login provider.
// Returns nil if the provider is not configured (no client credentials).
func (s *OAuthService) AppAuthConfig(provider string) *core.OAuthConfig {
	if _, ok := s.Clients[provider]; !ok {
		return nil
	}
	return appAuthConfigs[provider]
}

// BuildAppAuthURL constructs the full OAuth authorization URL for app login.
// Uses the app auth redirect URI pattern (/auth/{provider}/callback).
func (s *OAuthService) BuildAppAuthURL(provider string, cfg *core.OAuthConfig, state string) (string, error) {
	creds, ok := s.Clients[provider]
	if !ok {
		return "", ErrMissingOAuthClient
	}

	u, err := url.Parse(cfg.AuthURL)
	if err != nil {
		return "", fmt.Errorf("parsing auth URL: %w", err)
	}

	q := u.Query()
	q.Set("client_id", creds.ClientID)
	q.Set("redirect_uri", s.AppAuthRedirectURI(provider))
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(cfg.Scopes, " "))
	q.Set("state", state)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// ExchangeAppAuthCode exchanges an authorization code for tokens using the app auth redirect URI.
func (s *OAuthService) ExchangeAppAuthCode(provider string, cfg *core.OAuthConfig, code string) (*TokenResponse, error) {
	creds, ok := s.Clients[provider]
	if !ok {
		return nil, ErrMissingOAuthClient
	}

	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.AppAuthRedirectURI(provider)},
		"client_id":     {creds.ClientID},
		"client_secret": {creds.ClientSecret},
	}

	return s.postTokenRequest(cfg.TokenURL, data)
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

// RefreshPluginToken implements core.TokenRefresher. It delegates to RefreshToken
// and adapts the response to the core.TokenRefreshResult type.
func (s *OAuthService) RefreshPluginToken(pluginName string, cfg *core.OAuthConfig, refreshToken string) (*core.TokenRefreshResult, error) {
	resp, err := s.RefreshToken(pluginName, cfg, refreshToken)
	if err != nil {
		return nil, err
	}
	return &core.TokenRefreshResult{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
	}, nil
}

// --- Provider user info fetchers ---

// ProviderUserInfo holds user info fetched from an OAuth provider's API.
type ProviderUserInfo struct {
	Subject     string  // Provider-specific user ID
	Email       string  // User email
	DisplayName string  // Display name
	AvatarURL   *string // Optional avatar URL
}

// FetchGoogleUserInfo retrieves user info from Google's userinfo API.
func FetchGoogleUserInfo(accessToken string) (*ProviderUserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google userinfo returned %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding google userinfo: %w", err)
	}

	info := &ProviderUserInfo{
		Subject:     raw.ID,
		Email:       raw.Email,
		DisplayName: raw.Name,
	}
	if raw.Picture != "" {
		info.AvatarURL = &raw.Picture
	}
	return info, nil
}

// FetchGithubUserInfo retrieves user info from GitHub's user API.
func FetchGithubUserInfo(accessToken string) (*ProviderUserInfo, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch user profile
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user returned %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding github user: %w", err)
	}

	// If email is not public, fetch from /user/emails
	email := raw.Email
	if email == "" {
		email, _ = fetchGithubPrimaryEmail(accessToken)
	}

	displayName := raw.Name
	if displayName == "" {
		displayName = raw.Login
	}

	info := &ProviderUserInfo{
		Subject:     fmt.Sprintf("%d", raw.ID),
		Email:       email,
		DisplayName: displayName,
	}
	if raw.AvatarURL != "" {
		info.AvatarURL = &raw.AvatarURL
	}
	return info, nil
}

func fetchGithubPrimaryEmail(accessToken string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github emails returned %d", resp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no verified email found")
}

// postTokenRequest sends a POST to the token endpoint and parses the response.
func (s *OAuthService) postTokenRequest(tokenURL string, data url.Values) (*TokenResponse, error) {
	req, reqErr := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if reqErr != nil {
		return nil, fmt.Errorf("building token request: %w", reqErr)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json") // Required for GitHub

	resp, err := s.HTTPClient.Do(req)
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
