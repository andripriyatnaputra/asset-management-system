package security

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SimpleTokenBlacklist implements a thread-safe in-memory blacklist.
// NOTE: For production with multiple instances use Redis (comment hints below).
var (
	tbOnce         sync.Once
	tokenBlacklist *inMemoryBlacklist
)

type inMemoryBlacklist struct {
	mu    sync.RWMutex
	items map[string]time.Time // token -> expiry of blacklist
}

func initBlacklist() {
	tokenBlacklist = &inMemoryBlacklist{
		items: make(map[string]time.Time),
	}
	// background cleanup
	go func() {
		t := time.NewTicker(5 * time.Minute)
		for range t.C {
			tokenBlacklist.cleanup()
		}
	}()
}

func (b *inMemoryBlacklist) add(token string, ttl time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.items[token] = time.Now().Add(ttl)
}

func (b *inMemoryBlacklist) exists(token string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	exp, ok := b.items[token]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		return false
	}
	return true
}

func (b *inMemoryBlacklist) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	for k, v := range b.items {
		if now.After(v) {
			delete(b.items, k)
		}
	}
}

// RevokeToken adds token to blacklist for provided ttl (e.g., token expiry or configurable)
func RevokeToken(token string, ttl time.Duration) {
	tbOnce.Do(initBlacklist)
	tokenBlacklist.add(token, ttl)
}

// IsTokenRevoked checks whether token is revoked
func IsTokenRevoked(token string) bool {
	tbOnce.Do(initBlacklist)
	return tokenBlacklist.exists(token)
}

// Middleware wrapper for checking token blacklist. Use this in your auth middleware:
// if IsTokenRevoked(tokenString) { abort 401 ... }
func CheckTokenNotRevoked(c *gin.Context, token string) bool {
	// convenience: returns true if revoked
	return IsTokenRevoked(token)
}

// ----------------------
// Production: Redis example (commented):
// import "github.com/go-redis/redis/v8"
// var rdb *redis.Client
// func InitRedis(addr, pass string) {
//   rdb = redis.NewClient(&redis.Options{Addr: addr, Password: pass})
// }
// func RevokeTokenRedis(ctx context.Context, token string, ttl time.Duration) {
//   rdb.Set(ctx, "revoked:"+token, "1", ttl)
// }
// func IsRevokedRedis(ctx context.Context, token string) bool {
//   v, _ := rdb.Get(ctx, "revoked:"+token).Result()
//   return v == "1"
// }
// ----------------------
