// Package auth provides authentication middleware and context helpers.
package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/user/travel-calendar/backend/internal/store"
)

type contextKey string

const (
	userIDKey contextKey = "userID"
	emailKey  contextKey = "email"
	cookieName           = "travel_session"
	sessionTTL           = 30 * 24 * time.Hour // 30 days
)

// UserIDFromContext returns the authenticated user's ID.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}

// EmailFromContext returns the authenticated user's email.
func EmailFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(emailKey).(string); ok {
		return v
	}
	return ""
}

// Middleware returns HTTP middleware that enforces session authentication.
// It always populates user context from the session cookie if valid.
// For non-exempt API paths, it rejects requests without a valid session.
func Middleware(s store.StoreInterface) func(http.Handler) http.Handler {
	exemptPrefixes := []string{
		"/api/auth/",
		"/health",
	}

	isExempt := func(path string) bool {
		for _, prefix := range exemptPrefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}
		// Non-API routes (frontend static files) are always exempt
		if !strings.HasPrefix(path, "/api/") {
			return true
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always try to populate context from session cookie
			if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
				session, err := s.GetSession(cookie.Value)
				if err == nil && session != nil && !session.IsExpired() {
					ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
					ctx = context.WithValue(ctx, emailKey, session.Email)
					r = r.WithContext(ctx)
				} else if session != nil && session.IsExpired() {
					s.DeleteSession(session.ID)
					http.SetCookie(w, &http.Cookie{
						Name:     cookieName,
						Value:    "",
						Path:     "/",
						MaxAge:   -1,
						HttpOnly: true,
					})
				}
			}

			// Exempt paths pass through regardless of auth
			if isExempt(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Non-exempt API paths require authentication
			if UserIDFromContext(r.Context()) == "" {
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
