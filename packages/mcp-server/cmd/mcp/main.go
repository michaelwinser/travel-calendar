// Travel Calendar MCP Server
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/user/travel-calendar/mcp-server/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:3000"
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "mcp-server",
		})
	})

	// MCP endpoint
	mcpHandler := handler.NewMCPHandler(backendURL)
	r.Post("/mcp", mcpHandler.ServeHTTP)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("MCP server starting on %s (backend: %s)", addr, backendURL)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
