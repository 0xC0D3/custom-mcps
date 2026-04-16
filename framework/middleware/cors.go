package middleware

import (
	"net/http"
)

// CORS returns HTTP middleware that sets CORS headers on responses.
// origins is the list of allowed origins. Use "*" to allow all origins.
// If no origins are provided the middleware is a no-op pass-through.
func CORS(origins ...string) HTTPMiddleware {
	allowed := make(map[string]bool, len(origins))
	allowAll := false
	for _, o := range origins {
		if o == "*" {
			allowAll = true
		}
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			var matchedOrigin string
			switch {
			case allowAll:
				matchedOrigin = "*"
			case allowed[origin]:
				matchedOrigin = origin
			}

			if matchedOrigin != "" {
				w.Header().Set("Access-Control-Allow-Origin", matchedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, Mcp-Session-Id")
				w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
