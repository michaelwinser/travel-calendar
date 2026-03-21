// Travel Calendar Backend Server
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/user/travel-calendar/backend/internal/api"
	"github.com/user/travel-calendar/backend/internal/auth"
	"github.com/user/travel-calendar/backend/internal/handler"
	"github.com/user/travel-calendar/backend/internal/mcp"
	"github.com/user/travel-calendar/backend/internal/service"
	"github.com/user/travel-calendar/backend/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Initialize store (SQLite or Firestore based on STORE_TYPE env var)
	db, err := store.New()
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer db.Close()

	// Initialize service and handlers
	svc := service.New(db)
	h := handler.New(svc, db)
	mcpHandler := mcp.NewHandler(svc)

	// Initialize calendar service if configured
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if googleClientID != "" && googleClientSecret != "" {
		calendarSvc := service.NewCalendarService(db, service.CalendarConfig{
			ClientID:     googleClientID,
			ClientSecret: googleClientSecret,
			RedirectURL:  googleRedirectURL,
		})
		h.SetCalendarService(calendarSvc)
		log.Printf("Google Calendar integration enabled")
	} else {
		log.Printf("Google Calendar integration disabled (GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET not set)")
	}

	// Set up router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS and Content-Type middleware for API and MCP routes
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Apply JSON content-type and CORS to API and MCP routes
			if strings.HasPrefix(req.URL.Path, "/api/") || req.URL.Path == "/health" || req.URL.Path == "/mcp" {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				if req.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			next.ServeHTTP(w, req)
		})
	})

	// Auth middleware — checks session cookie for /api/* routes
	r.Use(auth.Middleware(db))

	// Logout endpoint (outside OpenAPI — simple POST that clears session)
	r.Post("/api/auth/logout", func(w http.ResponseWriter, req *http.Request) {
		cookie, err := req.Cookie("travel_session")
		if err == nil && cookie.Value != "" {
			db.DeleteSession(cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "travel_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	// Delete account endpoint — removes all user data
	r.Delete("/api/account", func(w http.ResponseWriter, req *http.Request) {
		uid := auth.UserIDFromContext(req.Context())
		if uid == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if err := db.DeleteAllUserData(uid); err != nil {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"failed to delete account data"}`, http.StatusInternalServerError)
			return
		}
		// Clear session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "travel_session",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})

	// Register OpenAPI handlers for /api/* and /health
	api.HandlerFromMux(h, r)

	// Register MCP JSON-RPC handler
	r.Post("/mcp", mcpHandler.ServeHTTP)
	r.Head("/mcp", mcpHandler.VersionHandler)
	r.Get("/mcp", mcpHandler.VersionHandler)

	// Serve embedded static frontend for all other routes
	r.Handle("/*", staticHandler())

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Backend server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
