package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/michaelwinser/appbase/auth"
)

func TestTestAuthMiddleware_Disabled(t *testing.T) {
	os.Unsetenv("TRAVEL_TEST_MODE")

	handler := TestAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID != "" {
			t.Errorf("expected empty userID when test mode disabled, got %q", userID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/activities", nil)
	req.Header.Set("X-Test-User", "attacker@evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}

func TestTestAuthMiddleware_Enabled(t *testing.T) {
	os.Setenv("TRAVEL_TEST_MODE", "true")
	defer os.Unsetenv("TRAVEL_TEST_MODE")

	handler := TestAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID != "test@example.com" {
			t.Errorf("expected userID %q, got %q", "test@example.com", userID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/activities", nil)
	req.Header.Set("X-Test-User", "test@example.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestTestAuthMiddleware_NoHeader(t *testing.T) {
	os.Setenv("TRAVEL_TEST_MODE", "true")
	defer os.Unsetenv("TRAVEL_TEST_MODE")

	handler := TestAuthMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := auth.UserID(r)
		if userID != "" {
			t.Errorf("expected empty userID without header, got %q", userID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/activities", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
}
