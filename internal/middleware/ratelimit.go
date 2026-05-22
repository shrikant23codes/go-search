package middleware

import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

type rateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

func (rl *rateLimiter) get(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if exists {
		return limiter
	}
	// 100 req/s with burst = 20
	limiter = rate.NewLimiter(rate.Limit(100), 20)
	rl.limiters[key] = limiter
	return limiter
}

func RateLimit() func(http.Handler) http.Handler {
	rl := newRateLimiter()

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				key = r.RemoteAddr // fallback to IP
			}

			if !rl.get(key).Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}
