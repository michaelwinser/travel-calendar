// Travel Calendar Backend Server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/user/travel-calendar/backend/internal/api"
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

	// Initialize store based on STORE_TYPE
	storeType := os.Getenv("STORE_TYPE")
	var db store.StoreInterface
	switch storeType {
	case "sqlite":
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = filepath.Join("data", "travel.db")
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}
		s, err := store.NewSQLite(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize SQLite store: %v", err)
		}
		db = s
		log.Printf("Using SQLite store: %s", dbPath)
	default:
		// Default to Firestore
		projectID := os.Getenv("FIREBASE_PROJECT_ID")
		if projectID == "" {
			projectID = "travel-calendar-dev"
		}
		s, err := store.NewFirestore(context.Background(), projectID)
		if err != nil {
			log.Fatalf("Failed to initialize Firestore store: %v", err)
		}
		db = s
		log.Printf("Using Firestore store (project: %s)", projectID)
	}
	defer db.Close()

	// Initialize service and handlers
	svc := service.New(db)
	h := handler.New(svc)
	mcpHandler := mcp.NewHandler(svc)

	// Initialize calendar service if configured
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if googleClientID != "" && googleClientSecret != "" {
		if googleRedirectURL == "" {
			googleRedirectURL = fmt.Sprintf("http://localhost:%s/oauth/google/callback", port)
		}
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
