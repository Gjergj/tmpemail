package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple in-memory rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
	name     string
}

// NewRateLimiter creates a new rate limiter with the specified requests per minute
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return NewRateLimiterWithName(requestsPerMinute, "default")
}

// NewRateLimiterWithName creates a new rate limiter with a name for identification
func NewRateLimiterWithName(requestsPerMinute int, name string) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    requestsPerMinute,
		window:   time.Minute,
		name:     name,
	}
}

// getClientIP extracts the client IP address from the request
// Note: chi's RealIP middleware should be used before this to populate RemoteAddr correctly
func getClientIP(r *http.Request) string {
	return r.RemoteAddr
}

// isLimited checks if the IP is rate limited and records the request
func (rl *RateLimiter) isLimited(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get request timestamps for this IP
	timestamps, exists := rl.requests[ip]
	if !exists {
		timestamps = []time.Time{}
	}

	// Filter out requests outside the time window
	validTimestamps := make([]time.Time, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.After(windowStart) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if limit is exceeded
	if len(validTimestamps) >= rl.limit {
		return true
	}

	// Add current request
	validTimestamps = append(validTimestamps, now)
	rl.requests[ip] = validTimestamps

	return false
}

// Middleware returns a chi-compatible middleware function
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		if rl.isLimited(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"Rate limit exceeded. Please try again later."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request from the given IP should be allowed (for non-HTTP use cases)
func (rl *RateLimiter) Allow(ip string) bool {
	return !rl.isLimited(ip)
}

// Cleanup removes old entries from the rate limiter (should be called periodically)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	for ip, timestamps := range rl.requests {
		validTimestamps := make([]time.Time, 0, len(timestamps))
		for _, ts := range timestamps {
			if ts.After(windowStart) {
				validTimestamps = append(validTimestamps, ts)
			}
		}

		if len(validTimestamps) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validTimestamps
		}
	}
}
