package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newTestRouter(mw gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", mw, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestRateLimitAuth_AllowsUnderLimit(t *testing.T) {
	r := newTestRouter(RateLimitAuth())

	// burst=5 — first 5 requests should all pass
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimitAuth_BlocksOverLimit(t *testing.T) {
	r := newTestRouter(RateLimitAuth())
	// Use a distinct IP so quota is fresh
	ip := "10.0.0.99:9999"

	// Drain the burst (5 tokens)
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ip
		r.ServeHTTP(w, req)
	}

	// The 6th request must be rejected
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = ip
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimitAPI_AllowsUnderLimit(t *testing.T) {
	r := newTestRouter(RateLimitAPI())

	// burst=30 — first 30 requests should all pass
	for i := 0; i < 30; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "2.2.2.2:1234"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimitAPI_IsolatesPerIP(t *testing.T) {
	r := newTestRouter(RateLimitAPI())
	// Drain burst for IP A
	for i := 0; i < 30; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "3.3.3.3:1234"
		r.ServeHTTP(w, req)
	}

	// IP B must still have its own fresh quota
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "4.4.4.4:1234"
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "different IP should not be affected")
}

func TestRateLimitExport_StrictBurst(t *testing.T) {
	r := newTestRouter(RateLimitExport())
	ip := "5.5.5.5:1234"

	// burst=3 allowed
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = ip
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "export request %d should pass", i+1)
	}

	// 4th must be blocked
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = ip
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "4th export request should be blocked")
}
