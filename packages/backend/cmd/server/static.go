// Static file serving for embedded frontend
package main

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

//go:embed dist/*
var staticFiles embed.FS

// staticCacheEnabled returns true if production caching should be enabled.
// Set STATIC_CACHE=true in production for 1-year cache on hashed assets.
// Defaults to false (no caching) for development.
func staticCacheEnabled() bool {
	return os.Getenv("STATIC_CACHE") == "true"
}

// staticHandler returns an http.Handler that serves the embedded static files.
// It handles SPA routing by serving index.html for non-file requests.
func staticHandler() http.Handler {
	// Strip "dist/" prefix from embedded filesystem
	distFS, err := fs.Sub(staticFiles, "dist")
	if err != nil {
		// If dist directory doesn't exist, return a handler that shows an error
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Frontend not built. Run: cd packages/frontend && pnpm build", http.StatusServiceUnavailable)
		})
	}

	fileServer := http.FileServer(http.FS(distFS))

	cacheEnabled := staticCacheEnabled()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Set cache headers for static assets
		if cacheEnabled && strings.HasPrefix(path, "/_app/") {
			// Immutable assets (hashed filenames) - cache for 1 year
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else if strings.HasSuffix(path, ".html") || path == "/" {
			// HTML files - no cache
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		// Try to serve the file directly if it has an extension
		if strings.Contains(path, ".") {
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for routes without extensions
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
