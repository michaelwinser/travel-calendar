// Package entity defines the internal domain models with database tags.
package entity

import (
	"strings"
	"time"

	"github.com/user/travel-calendar/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// GoogleCredentials stores OAuth tokens for a user's Google account.
// This entity is not directly exposed via API - it stores sensitive credentials.
type GoogleCredentials struct {
	UserID       string    `db:"user_id"`
	AccessToken  string    `db:"access_token"`
	RefreshToken string    `db:"refresh_token"`
	TokenType    string    `db:"token_type"`
	ExpiresAt    time.Time `db:"expires_at"`
	Scopes       string    `db:"scopes"` // comma-separated scopes
	Email        *string   `db:"email"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// IsExpired returns true if the access token has expired.
func (g *GoogleCredentials) IsExpired() bool {
	return time.Now().After(g.ExpiresAt)
}

// GetScopes returns the scopes as a slice.
func (g *GoogleCredentials) GetScopes() []string {
	if g.Scopes == "" {
		return nil
	}
	return strings.Split(g.Scopes, ",")
}

// SetScopes sets the scopes from a slice.
func (g *GoogleCredentials) SetScopes(scopes []string) {
	g.Scopes = strings.Join(scopes, ",")
}

// ToAPIStatus converts credentials to an API GoogleAuthStatus response.
// This is safe to return - it only exposes non-sensitive metadata.
func (g *GoogleCredentials) ToAPIStatus() api.GoogleAuthStatus {
	status := api.GoogleAuthStatus{
		Connected: true,
		ExpiresAt: &g.ExpiresAt,
	}

	if g.Email != nil {
		email := openapi_types.Email(*g.Email)
		status.Email = &email
	}

	scopes := g.GetScopes()
	if len(scopes) > 0 {
		status.Scopes = &scopes
	}

	return status
}

// NewGoogleCredentials creates a new GoogleCredentials entity.
func NewGoogleCredentials(userID, accessToken, refreshToken, tokenType string, expiresAt time.Time, scopes []string) GoogleCredentials {
	now := time.Now()
	creds := GoogleCredentials{
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		ExpiresAt:    expiresAt,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	creds.SetScopes(scopes)
	return creds
}
