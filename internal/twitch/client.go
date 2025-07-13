package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
)

// Client handles Twitch API interactions
type Client struct {
	config     *config.Config
	logger     *slog.Logger
	httpClient *http.Client
	token      *AppAccessToken
	tokenMutex sync.RWMutex
}

// AppAccessToken represents a Twitch app access token
type AppAccessToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// UserInfo represents Twitch user information
type UserInfo struct {
	ID              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageURL string `json:"profile_image_url"`
	OfflineImageURL string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	CreatedAt       string `json:"created_at"`
}

// ChannelInfo represents Twitch channel information
type ChannelInfo struct {
	BroadcasterID       string   `json:"broadcaster_id"`
	BroadcasterLogin    string   `json:"broadcaster_login"`
	BroadcasterName     string   `json:"broadcaster_name"`
	BroadcasterLanguage string   `json:"broadcaster_language"`
	GameID              string   `json:"game_id"`
	GameName            string   `json:"game_name"`
	Title               string   `json:"title"`
	Tags                []string `json:"tags"`
	IsMature            bool     `json:"is_mature"`
}

// FollowersResponse represents the response from the followers API
type FollowersResponse struct {
	Total int `json:"total"`
}

// NewClient creates a new Twitch API client
func NewClient(cfg *config.Config, logger *slog.Logger) *Client {
	return &Client{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start initializes the client and loads/refreshes the access token
func (c *Client) Start(ctx context.Context) error {
	// Load existing token
	if err := c.loadToken(); err != nil {
		c.logger.Warn("Failed to load existing token", "error", err)
	}

	// Get or refresh token
	if err := c.ensureValidToken(ctx); err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}

	c.logger.Info("Twitch API client started")
	return nil
}

// Stop saves the current token state
func (c *Client) Stop() error {
	if err := c.saveToken(); err != nil {
		c.logger.Error("Failed to save token", "error", err)
		return err
	}

	c.logger.Info("Twitch API client stopped")
	return nil
}

// GetUserInfo retrieves user information for a given user ID or login
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?id=%s", userID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Data []UserInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return &response.Data[0], nil
}

// GetUserInfoByLogin retrieves user information for a given login name
func (c *Client) GetUserInfoByLogin(ctx context.Context, login string) (*UserInfo, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", login)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Data []UserInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	return &response.Data[0], nil
}

// GetID returns the user ID (implements config.TwitchUserInfo interface)
func (u *UserInfo) GetID() string {
	return u.ID
}

// GetLogin returns the user login (implements config.TwitchUserInfo interface)
func (u *UserInfo) GetLogin() string {
	return u.Login
}

// GetUserInfoByLoginForConfig is an adapter method that returns a config.TwitchUserInfo interface
func (c *Client) GetUserInfoByLoginForConfig(ctx context.Context, login string) (config.TwitchUserInfo, error) {
	userInfo, err := c.GetUserInfoByLogin(ctx, login)
	if err != nil {
		return nil, err
	}
	
	return &TwitchUserInfoForConfig{
		ID:    userInfo.ID,
		Login: userInfo.Login,
	}, nil
}

// TwitchUserInfoForConfig represents basic user information for config resolution
type TwitchUserInfoForConfig struct {
	ID    string
	Login string
}

// GetID returns the user ID
func (u *TwitchUserInfoForConfig) GetID() string {
	return u.ID
}

// GetLogin returns the user login
func (u *TwitchUserInfoForConfig) GetLogin() string {
	return u.Login
}

// GetChannelInfo retrieves channel information for a given broadcaster ID
func (c *Client) GetChannelInfo(ctx context.Context, broadcasterID string) (*ChannelInfo, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/channels?broadcaster_id=%s", broadcasterID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response struct {
		Data []ChannelInfo `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("channel not found")
	}

	return &response.Data[0], nil
}

// GetFollowersCount retrieves the follower count for a given broadcaster ID
func (c *Client) GetFollowersCount(ctx context.Context, broadcasterID string) (int, error) {
	if err := c.ensureValidToken(ctx); err != nil {
		return 0, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/channels/followers?broadcaster_id=%s&first=1", broadcasterID)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	c.setAuthHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	var response FollowersResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Total, nil
}

// ensureValidToken ensures we have a valid access token
func (c *Client) ensureValidToken(ctx context.Context) error {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	// Check if we have a valid token
	if c.token != nil && time.Now().Before(c.token.ExpiresAt.Add(-5*time.Minute)) {
		return nil // Token is still valid
	}

	// Get new token
	token, err := c.getAppAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to get app access token: %w", err)
	}

	c.token = token
	c.logger.Info("Obtained new Twitch access token", "expires_at", token.ExpiresAt)

	return nil
}

// getAppAccessToken retrieves a new app access token using client credentials flow
func (c *Client) getAppAccessToken(ctx context.Context) (*AppAccessToken, error) {
	data := url.Values{}
	data.Set("client_id", c.config.Twitch.ClientID)
	data.Set("client_secret", c.config.Twitch.ClientSecret)
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://id.twitch.tv/oauth2/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var token AppAccessToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	// Calculate expiration time
	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	return &token, nil
}

// setAuthHeaders sets the required authentication headers for API requests
func (c *Client) setAuthHeaders(req *http.Request) {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()

	if c.token != nil {
		req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	}
	req.Header.Set("Client-Id", c.config.Twitch.ClientID)
}

// loadToken loads the access token from disk
func (c *Client) loadToken() error {
	if _, err := os.Stat(c.config.Twitch.TokenFile); os.IsNotExist(err) {
		return nil // No token file exists yet
	}

	data, err := os.ReadFile(c.config.Twitch.TokenFile)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var token AppAccessToken
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("failed to unmarshal token: %w", err)
	}

	c.tokenMutex.Lock()
	c.token = &token
	c.tokenMutex.Unlock()

	c.logger.Debug("Loaded token from disk", "expires_at", token.ExpiresAt)
	return nil
}

// saveToken saves the access token to disk
func (c *Client) saveToken() error {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()

	if c.token == nil {
		return nil // No token to save
	}

	data, err := json.MarshalIndent(c.token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(c.config.Twitch.TokenFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}