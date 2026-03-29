package app

import (
	"log"
	"net/http"
	"os"

	"github.com/michaelwinser/appbase/auth"
)

// TestAuthMiddleware allows test scripts and agents to authenticate via
// the X-Test-User header when TRAVEL_TEST_MODE=true.
//
// SECURITY: This must NEVER be enabled in production. The middleware logs
// a warning on startup and on every authenticated request.
func TestAuthMiddleware() func(http.Handler) http.Handler {
	enabled := os.Getenv("TRAVEL_TEST_MODE") == "true"
	if !enabled {
		return func(next http.Handler) http.Handler { return next }
	}

	log.Println("WARNING: test authentication enabled (TRAVEL_TEST_MODE=true)")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			testUser := r.Header.Get("X-Test-User")
			if testUser != "" {
				ctx := auth.WithIdentity(r.Context(), testUser, testUser)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
