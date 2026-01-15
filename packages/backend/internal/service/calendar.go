// Package service implements business logic for the Travel Calendar application.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/entity"
	"github.com/user/travel-calendar/backend/internal/store"
)

// CalendarService handles Google Calendar OAuth and operations.
type CalendarService struct {
	store        *store.Store
	clientID     string
	clientSecret string
	redirectURL  string
	httpClient   *http.Client
}

// CalendarConfig holds configuration for the calendar service.
type CalendarConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewCalendarService creates a new CalendarService.
func NewCalendarService(s *store.Store, config CalendarConfig) *CalendarService {
	return &CalendarService{
		store:        s,
		clientID:     config.ClientID,
		clientSecret: config.ClientSecret,
		redirectURL:  config.RedirectURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// IsConfigured returns true if the calendar service is properly configured.
func (c *CalendarService) IsConfigured() bool {
	return c.clientID != "" && c.clientSecret != "" && c.redirectURL != ""
}

// DefaultUserID is used for single-user mode.
const DefaultUserID = "default"

// OAuth scopes for Google Calendar
const (
	ScopeCalendarReadonly  = "https://www.googleapis.com/auth/calendar.readonly"
	ScopeCalendarEvents    = "https://www.googleapis.com/auth/calendar.events"
	ScopeCalendarReadWrite = "https://www.googleapis.com/auth/calendar"
)

// GetAuthURL generates the OAuth URL for Google Calendar authorization.
func (c *CalendarService) GetAuthURL(scopes string) (*api.OAuthUrlResponse, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("Google Calendar not configured")
	}

	// Parse scopes - default to readonly
	scopeList := ScopeCalendarReadonly
	if scopes != "" {
		scopeList = scopes
	}

	// Generate state token for CSRF protection
	state := uuid.New().String()

	// Build OAuth URL
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", c.redirectURL)
	params.Set("response_type", "code")
	params.Set("scope", scopeList)
	params.Set("access_type", "offline")
	params.Set("prompt", "consent") // Force consent to get refresh token
	params.Set("state", state)

	authURL := "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()

	// Note: state is included in URL params but not returned in response
	// Frontend should extract it from the URL if needed for CSRF validation
	return &api.OAuthUrlResponse{
		Url: authURL,
	}, nil
}

// tokenResponse represents the OAuth token response from Google.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// HandleCallback exchanges the authorization code for tokens and stores them.
func (c *CalendarService) HandleCallback(ctx context.Context, code string) (*api.GoogleAuthStatus, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("Google Calendar not configured")
	}

	// Exchange code for tokens
	tokens, err := c.exchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	// Get user email from token info
	email, err := c.getUserEmail(ctx, tokens.AccessToken)
	if err != nil {
		// Non-fatal - we can still proceed without email
		email = ""
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	// Create and save credentials
	creds := entity.NewGoogleCredentials(
		DefaultUserID,
		tokens.AccessToken,
		tokens.RefreshToken,
		tokens.TokenType,
		expiresAt,
		strings.Split(tokens.Scope, " "),
	)
	if email != "" {
		creds.Email = &email
	}

	if err := c.store.SaveGoogleCredentials(&creds); err != nil {
		return nil, fmt.Errorf("saving credentials: %w", err)
	}

	status := creds.ToAPIStatus()
	return &status, nil
}

// exchangeCode exchanges an authorization code for tokens.
func (c *CalendarService) exchangeCode(ctx context.Context, code string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("redirect_uri", c.redirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s", string(body))
	}

	var tokens tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return &tokens, nil
}

// getUserEmail retrieves the user's email from the token info endpoint.
func (c *CalendarService) getUserEmail(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get user info")
	}

	var userInfo struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", err
	}

	return userInfo.Email, nil
}

// GetAuthStatus returns the current authentication status.
func (c *CalendarService) GetAuthStatus(ctx context.Context) (*api.GoogleAuthStatus, error) {
	creds, err := c.store.GetGoogleCredentials(DefaultUserID)
	if err != nil {
		return nil, fmt.Errorf("getting credentials: %w", err)
	}

	if creds == nil {
		return &api.GoogleAuthStatus{Connected: false}, nil
	}

	status := creds.ToAPIStatus()
	return &status, nil
}

// Disconnect revokes access and removes credentials.
func (c *CalendarService) Disconnect(ctx context.Context) error {
	creds, err := c.store.GetGoogleCredentials(DefaultUserID)
	if err != nil {
		return fmt.Errorf("getting credentials: %w", err)
	}

	if creds == nil {
		return nil // Already disconnected
	}

	// Revoke token at Google
	if err := c.revokeToken(ctx, creds.AccessToken); err != nil {
		// Log but don't fail - we still want to delete local credentials
		fmt.Printf("Warning: failed to revoke token at Google: %v\n", err)
	}

	// Delete local credentials
	if err := c.store.DeleteGoogleCredentials(DefaultUserID); err != nil {
		return fmt.Errorf("deleting credentials: %w", err)
	}

	// Delete associated user calendars
	if err := c.store.DeleteUserCalendarsByUser(DefaultUserID); err != nil {
		return fmt.Errorf("deleting user calendars: %w", err)
	}

	return nil
}

// revokeToken revokes an access token at Google.
func (c *CalendarService) revokeToken(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/revoke?token="+url.QueryEscape(token), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("revoke failed: %s", string(body))
	}

	return nil
}

// RefreshTokenIfNeeded refreshes the access token if it's expired.
func (c *CalendarService) RefreshTokenIfNeeded(ctx context.Context) (*entity.GoogleCredentials, error) {
	creds, err := c.store.GetGoogleCredentials(DefaultUserID)
	if err != nil {
		return nil, fmt.Errorf("getting credentials: %w", err)
	}

	if creds == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	// Check if token needs refresh (refresh 5 minutes before expiry)
	if time.Now().Add(5 * time.Minute).Before(creds.ExpiresAt) {
		return creds, nil // Token still valid
	}

	// Refresh the token
	newTokens, err := c.refreshToken(ctx, creds.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}

	// Update credentials
	creds.AccessToken = newTokens.AccessToken
	creds.ExpiresAt = time.Now().Add(time.Duration(newTokens.ExpiresIn) * time.Second)
	creds.UpdatedAt = time.Now()

	// If we got a new refresh token, update it
	if newTokens.RefreshToken != "" {
		creds.RefreshToken = newTokens.RefreshToken
	}

	if err := c.store.SaveGoogleCredentials(creds); err != nil {
		return nil, fmt.Errorf("saving refreshed credentials: %w", err)
	}

	return creds, nil
}

// refreshToken refreshes an OAuth token.
func (c *CalendarService) refreshToken(ctx context.Context, refreshToken string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s", string(body))
	}

	var tokens tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	return &tokens, nil
}

// ListCalendars fetches available calendars from Google.
func (c *CalendarService) ListCalendars(ctx context.Context) ([]api.GoogleCalendar, error) {
	creds, err := c.RefreshTokenIfNeeded(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/calendar/v3/users/me/calendarList", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list calendars: %s", string(body))
	}

	var result struct {
		Items []struct {
			ID              string `json:"id"`
			Summary         string `json:"summary"`
			Description     string `json:"description"`
			Primary         bool   `json:"primary"`
			BackgroundColor string `json:"backgroundColor"`
			AccessRole      string `json:"accessRole"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	calendars := make([]api.GoogleCalendar, len(result.Items))
	for i, item := range result.Items {
		cal := api.GoogleCalendar{
			Id:   item.ID,
			Name: item.Summary,
		}
		if item.Description != "" {
			cal.Description = &item.Description
		}
		if item.Primary {
			cal.Primary = &item.Primary
		}
		if item.BackgroundColor != "" {
			cal.BackgroundColor = &item.BackgroundColor
		}
		if item.AccessRole != "" {
			role := api.GoogleCalendarAccessRole(item.AccessRole)
			cal.AccessRole = &role
		}
		calendars[i] = cal
	}

	return calendars, nil
}

// GetSelectedCalendars returns the user's selected calendars.
func (c *CalendarService) GetSelectedCalendars(ctx context.Context) ([]api.UserCalendar, error) {
	calendars, err := c.store.ListUserCalendars(DefaultUserID)
	if err != nil {
		return nil, fmt.Errorf("listing user calendars: %w", err)
	}

	return entity.UserCalendarsToAPI(calendars), nil
}

// SetSelectedCalendars updates the user's selected calendars.
// It first fetches calendar names from Google, then stores the selection.
func (c *CalendarService) SetSelectedCalendars(ctx context.Context, req *api.SetSelectedCalendarsRequest) ([]api.UserCalendar, error) {
	// Build a map of calendar IDs to enable
	enabledIDs := make(map[string]bool)
	for _, id := range req.CalendarIds {
		enabledIDs[id] = true
	}

	// Fetch calendar list from Google to get names
	googleCalendars, err := c.ListCalendars(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing Google calendars: %w", err)
	}

	// Create calendar name map
	calendarNames := make(map[string]string)
	for _, cal := range googleCalendars {
		calendarNames[cal.Id] = cal.Name
	}

	now := time.Now()
	calendars := make([]entity.UserCalendar, 0, len(req.CalendarIds))

	for _, calID := range req.CalendarIds {
		name := calendarNames[calID]
		if name == "" {
			name = calID // Fall back to ID if name not found
		}
		calendars = append(calendars, entity.UserCalendar{
			ID:         uuid.New(),
			UserID:     DefaultUserID,
			CalendarID: calID,
			Name:       name,
			Enabled:    true, // All IDs in the request are enabled
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}

	if err := c.store.SetUserCalendars(DefaultUserID, calendars); err != nil {
		return nil, fmt.Errorf("setting user calendars: %w", err)
	}

	return entity.UserCalendarsToAPI(calendars), nil
}
