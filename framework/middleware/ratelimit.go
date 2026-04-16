package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// tokenBucket implements a simple token bucket rate limiter for a single client.
type tokenBucket struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

// allow checks whether a request is allowed under the rate limit. It refills
// tokens based on elapsed time and decrements the bucket by one if possible.
func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastRefill = now

	if tb.tokens < 1 {
		return false
	}
	tb.tokens--
	return true
}

// RateLimit returns HTTP middleware that limits requests per client IP using a
// token bucket algorithm. requestsPerSecond controls the refill rate and burst
// controls the maximum number of tokens (and thus the burst size).
func RateLimit(requestsPerSecond, burst int) HTTPMiddleware {
	var buckets sync.Map

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, _ := net.SplitHostPort(r.RemoteAddr)
			if ip == "" {
				ip = r.RemoteAddr
			}

			val, _ := buckets.LoadOrStore(ip, &tokenBucket{
				tokens:     float64(burst),
				maxTokens:  float64(burst),
				refillRate: float64(requestsPerSecond),
				lastRefill: time.Now(),
			})
			bucket := val.(*tokenBucket)

			if !bucket.allow() {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limit exceeded",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
