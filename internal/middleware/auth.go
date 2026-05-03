package middleware

import (
	"net/http"
	"os"
)

// Auth creates middleware that validates the X-API-Key header.
func Auth(apiKeys []string, healthPath string) func(http.Handler) http.Handler {
	if os.Getenv("AUTH_DISABLED") != "" {
		return func(next http.Handler) http.Handler { return next }
	}

	keySet := make(map[string]bool, len(apiKeys))
	for _, k := range apiKeys {
		if k != "" {
			keySet[k] = true
		}
	}
	_ = keySet // suppress unused warning when empty
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == healthPath {
				next.ServeHTTP(w, r)
				return
			}
			key := r.Header.Get("X-API-Key")
			if key == "" || !keySet[key] {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized","message":"invalid or missing API key"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
