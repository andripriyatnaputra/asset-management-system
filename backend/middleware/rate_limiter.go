package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	r        rate.Limit
	b        int
}

func newStore(r rate.Limit, b int) *rateLimiterStore {
	s := &rateLimiterStore{
		limiters: make(map[string]*ipLimiter),
		r:        r,
		b:        b,
	}
	// Cleanup stale entries every 5 minutes
	go func() {
		for range time.Tick(5 * time.Minute) {
			s.mu.Lock()
			for ip, l := range s.limiters {
				if time.Since(l.lastSeen) > 10*time.Minute {
					delete(s.limiters, ip)
				}
			}
			s.mu.Unlock()
		}
	}()
	return s
}

func (s *rateLimiterStore) get(ip string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()
	if l, ok := s.limiters[ip]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}
	l := &ipLimiter{limiter: rate.NewLimiter(s.r, s.b), lastSeen: time.Now()}
	s.limiters[ip] = l
	return l.limiter
}

// stores for different limit tiers
var (
	// Auth endpoints: 10 req/min (burst 5)
	authStore = newStore(rate.Every(6*time.Second), 5)
	// API endpoints: 120 req/min (burst 30)
	apiStore = newStore(rate.Every(500*time.Millisecond), 30)
	// Export endpoints: 10 req/min (burst 3) — heavy operations
	exportStore = newStore(rate.Every(6*time.Second), 3)
)

func rateLimitMiddleware(store *rateLimiterStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !store.get(ip).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "terlalu banyak permintaan, coba lagi sebentar",
			})
			return
		}
		c.Next()
	}
}

// RateLimitAuth applies tight rate limiting for login/refresh endpoints.
func RateLimitAuth() gin.HandlerFunc {
	return rateLimitMiddleware(authStore)
}

// RateLimitAPI applies standard rate limiting for authenticated API endpoints.
func RateLimitAPI() gin.HandlerFunc {
	return rateLimitMiddleware(apiStore)
}

// RateLimitExport applies strict rate limiting for heavy export endpoints.
func RateLimitExport() gin.HandlerFunc {
	return rateLimitMiddleware(exportStore)
}
