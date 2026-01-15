// Package service implements business logic for the Travel Calendar application.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
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

// googleCalendarEvent represents a single event from the Google Calendar API.
type googleCalendarEvent struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Location    string `json:"location"`
	HtmlLink    string `json:"htmlLink"`
	Start       struct {
		Date     string `json:"date"`     // For all-day events: YYYY-MM-DD
		DateTime string `json:"dateTime"` // For timed events: RFC3339
	} `json:"start"`
	End struct {
		Date     string `json:"date"`
		DateTime string `json:"dateTime"`
	} `json:"end"`
}

// googleEventsResponse represents the response from Google Calendar events list API.
type googleEventsResponse struct {
	Items         []googleCalendarEvent `json:"items"`
	NextPageToken string                `json:"nextPageToken"`
}

// ListCalendarEvents fetches events from Google Calendar within a date range.
func (c *CalendarService) ListCalendarEvents(ctx context.Context, from, to time.Time, calendarID *string) ([]api.CalendarEvent, error) {
	creds, err := c.RefreshTokenIfNeeded(ctx)
	if err != nil {
		return nil, err
	}

	// Determine which calendars to fetch from
	var calendarIDs []string
	if calendarID != nil && *calendarID != "" {
		calendarIDs = []string{*calendarID}
	} else {
		// Get selected calendars
		userCalendars, err := c.store.ListUserCalendars(DefaultUserID)
		if err != nil {
			return nil, fmt.Errorf("listing user calendars: %w", err)
		}
		for _, uc := range userCalendars {
			if uc.Enabled {
				calendarIDs = append(calendarIDs, uc.CalendarID)
			}
		}
	}

	if len(calendarIDs) == 0 {
		return []api.CalendarEvent{}, nil
	}

	var allEvents []api.CalendarEvent

	for _, calID := range calendarIDs {
		events, err := c.fetchEventsFromCalendar(ctx, creds.AccessToken, calID, from, to)
		if err != nil {
			// Log error but continue with other calendars
			fmt.Printf("Warning: failed to fetch events from calendar %s: %v\n", calID, err)
			continue
		}
		allEvents = append(allEvents, events...)
	}

	return allEvents, nil
}

// fetchEventsFromCalendar fetches events from a single calendar.
func (c *CalendarService) fetchEventsFromCalendar(ctx context.Context, accessToken, calendarID string, from, to time.Time) ([]api.CalendarEvent, error) {
	var events []api.CalendarEvent
	pageToken := ""

	for {
		// Build URL with query parameters
		params := url.Values{}
		params.Set("timeMin", from.Format(time.RFC3339))
		params.Set("timeMax", to.Format(time.RFC3339))
		params.Set("singleEvents", "true")
		params.Set("orderBy", "startTime")
		params.Set("maxResults", "250")
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}

		reqURL := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events?%s",
			url.PathEscape(calendarID), params.Encode())

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to list events: %s", string(body))
		}

		var result googleEventsResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		// Convert Google events to API events
		for _, item := range result.Items {
			event, err := convertGoogleEvent(item, calendarID)
			if err != nil {
				// Skip events that can't be parsed
				continue
			}
			events = append(events, event)
		}

		if result.NextPageToken == "" {
			break
		}
		pageToken = result.NextPageToken
	}

	return events, nil
}

// convertGoogleEvent converts a Google Calendar event to an API event.
func convertGoogleEvent(ge googleCalendarEvent, calendarID string) (api.CalendarEvent, error) {
	event := api.CalendarEvent{
		Id:         ge.ID,
		CalendarId: calendarID,
		Summary:    ge.Summary,
	}

	if ge.Description != "" {
		event.Description = &ge.Description
	}
	if ge.Location != "" {
		event.Location = &ge.Location
	}
	if ge.HtmlLink != "" {
		event.HtmlLink = &ge.HtmlLink
	}

	// Parse start time
	if ge.Start.DateTime != "" {
		t, err := time.Parse(time.RFC3339, ge.Start.DateTime)
		if err != nil {
			return event, fmt.Errorf("parsing start datetime: %w", err)
		}
		event.Start = t
	} else if ge.Start.Date != "" {
		t, err := time.Parse("2006-01-02", ge.Start.Date)
		if err != nil {
			return event, fmt.Errorf("parsing start date: %w", err)
		}
		event.Start = t
		allDay := true
		event.AllDay = &allDay
	}

	// Parse end time
	if ge.End.DateTime != "" {
		t, err := time.Parse(time.RFC3339, ge.End.DateTime)
		if err != nil {
			return event, fmt.Errorf("parsing end datetime: %w", err)
		}
		event.End = t
	} else if ge.End.Date != "" {
		t, err := time.Parse("2006-01-02", ge.End.Date)
		if err != nil {
			return event, fmt.Errorf("parsing end date: %w", err)
		}
		event.End = t
	}

	return event, nil
}

// travelKeywords are words that indicate a travel-related event.
var travelKeywordsPattern = regexp.MustCompile(`(?i)\b(travel|flight|drive|park|hotel|checkin|check-in|train)\b`)

// urlPattern matches http:// or https:// URLs in the location field.
// These typically indicate virtual meetings (Zoom, Teams, Meet, etc.) rather than physical travel.
var urlPattern = regexp.MustCompile(`(?i)^https?://`)

// meetingRoomPattern matches Google Workspace meeting room resource identifiers.
// Examples: "AU-SYD-ODI-1-3-Oblique (7) [GVC]", "US-BLD-PEARL2930-1-B-Boxelder (2) [GVC, Preview]"
// These have patterns like:
// - Country-city codes: XX-YYY- (e.g., AU-SYD-, US-BLD-, US-NYC-)
// - Room capacity in parentheses: (7), (12)
// - Feature tags in brackets: [GVC], [GVC, Preview]
var meetingRoomPattern = regexp.MustCompile(`^[A-Z]{2}-[A-Z]{2,4}-|\(\d+\)|\[GVC`)

// isMeetingRoomLocation returns true if the location appears to be a meeting room resource name.
func isMeetingRoomLocation(location string) bool {
	return meetingRoomPattern.MatchString(location)
}

// isTravelRelatedEvent returns true if the event appears to be travel-related.
func isTravelRelatedEvent(event api.CalendarEvent) bool {
	// Has travel-related keywords in the summary
	if travelKeywordsPattern.MatchString(event.Summary) {
		return true
	}

	// Has a physical location set (not a URL or meeting room)
	if event.Location != nil && *event.Location != "" {
		location := *event.Location
		// Skip URLs (virtual meetings like Zoom, Teams, etc.)
		if urlPattern.MatchString(location) {
			return false
		}
		// Skip meeting room resource names
		if isMeetingRoomLocation(location) {
			return false
		}
		return true
	}

	return false
}

// SuggestTripsFromCalendar analyzes calendar events and suggests trips.
func (c *CalendarService) SuggestTripsFromCalendar(ctx context.Context, from, to time.Time) ([]api.TripSuggestion, error) {
	// Fetch all events in the date range
	events, err := c.ListCalendarEvents(ctx, from, to, nil)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}

	// Filter to travel-related events
	var travelEvents []api.CalendarEvent
	for _, event := range events {
		if isTravelRelatedEvent(event) {
			travelEvents = append(travelEvents, event)
		}
	}

	if len(travelEvents) == 0 {
		return []api.TripSuggestion{}, nil
	}

	// Sort by start time
	sort.Slice(travelEvents, func(i, j int) bool {
		return travelEvents[i].Start.Before(travelEvents[j].Start)
	})

	// Group events into trip suggestions
	suggestions := groupEventsIntoSuggestions(travelEvents)

	return suggestions, nil
}

// groupEventsIntoSuggestions groups travel events into trip suggestions.
// Events at the same location within 7 days are grouped together.
func groupEventsIntoSuggestions(events []api.CalendarEvent) []api.TripSuggestion {
	if len(events) == 0 {
		return []api.TripSuggestion{}
	}

	var suggestions []api.TripSuggestion
	var currentGroup []api.CalendarEvent
	var currentLocation string

	for _, event := range events {
		eventLocation := normalizeLocation(event)

		if len(currentGroup) == 0 {
			// Start a new group
			currentGroup = []api.CalendarEvent{event}
			currentLocation = eventLocation
			continue
		}

		// Check if this event should be grouped with the current group
		lastEvent := currentGroup[len(currentGroup)-1]
		daysSinceLast := event.Start.Sub(lastEvent.End).Hours() / 24

		// Group if: same location (or one is empty) AND within 7 days
		sameLocation := eventLocation == currentLocation ||
			eventLocation == "" ||
			currentLocation == ""

		if sameLocation && daysSinceLast <= 7 {
			currentGroup = append(currentGroup, event)
			// Update current location if this event has one and current doesn't
			if eventLocation != "" && currentLocation == "" {
				currentLocation = eventLocation
			}
		} else {
			// Finalize current group and start new one
			if suggestion := createSuggestion(currentGroup, currentLocation); suggestion != nil {
				suggestions = append(suggestions, *suggestion)
			}
			currentGroup = []api.CalendarEvent{event}
			currentLocation = eventLocation
		}
	}

	// Don't forget the last group
	if len(currentGroup) > 0 {
		if suggestion := createSuggestion(currentGroup, currentLocation); suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	return suggestions
}

// normalizeLocation extracts and normalizes the location from an event.
func normalizeLocation(event api.CalendarEvent) string {
	if event.Location != nil && *event.Location != "" {
		return strings.TrimSpace(*event.Location)
	}
	return ""
}

// createSuggestion creates a TripSuggestion from a group of events.
func createSuggestion(events []api.CalendarEvent, location string) *api.TripSuggestion {
	if len(events) == 0 {
		return nil
	}

	// Calculate date range
	startDate := events[0].Start
	endDate := events[len(events)-1].End

	// Generate deterministic ID from event IDs
	suggestionID := generateSuggestionID(events)

	// Generate trip name
	name := generateTripName(events, location)

	// Default location fallback
	if location == "" {
		location = "Unknown Location"
	}

	return &api.TripSuggestion{
		Id:           suggestionID,
		Name:         name,
		Location:     location,
		StartDate:    types.Date{Time: startDate},
		EndDate:      types.Date{Time: endDate},
		SourceEvents: events,
	}
}

// generateSuggestionID creates a deterministic ID from the event IDs.
func generateSuggestionID(events []api.CalendarEvent) string {
	// Collect and sort event IDs for determinism
	var ids []string
	for _, e := range events {
		ids = append(ids, e.Id)
	}
	sort.Strings(ids)

	// Hash the combined IDs
	combined := strings.Join(ids, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for a shorter ID
}

// generateTripName creates a descriptive name for the trip.
func generateTripName(events []api.CalendarEvent, location string) string {
	if location != "" {
		// Shorten location for the name (take first part before comma)
		shortLocation := location
		if idx := strings.Index(location, ","); idx > 0 {
			shortLocation = location[:idx]
		}
		return shortLocation + " Trip"
	}

	// Fall back to first event's summary
	if len(events) > 0 && events[0].Summary != "" {
		return events[0].Summary
	}

	return "Trip"
}

// ImportTripSuggestion creates a new trip from a suggestion.
func (c *CalendarService) ImportTripSuggestion(ctx context.Context, svc *Service, suggestionID string, from, to time.Time) (*api.Trip, error) {
	// Re-generate suggestions to find the matching one
	suggestions, err := c.SuggestTripsFromCalendar(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("generating suggestions: %w", err)
	}

	// Find the suggestion by ID
	var suggestion *api.TripSuggestion
	for _, s := range suggestions {
		if s.Id == suggestionID {
			suggestion = &s
			break
		}
	}

	if suggestion == nil {
		return nil, fmt.Errorf("suggestion not found: %s", suggestionID)
	}

	// Create the trip
	purpose := api.TripPurposeOther
	createReq := api.CreateTripRequest{
		Name:      suggestion.Name,
		Purpose:   purpose,
		StartDate: &suggestion.StartDate,
		EndDate:   &suggestion.EndDate,
		Location:  &suggestion.Location,
	}

	trip, err := svc.CreateTrip(&createReq)
	if err != nil {
		return nil, fmt.Errorf("creating trip: %w", err)
	}

	return trip, nil
}
