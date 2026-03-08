package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock RateLimiter ---

type mockRateLimiter struct {
	allowed   bool
	remaining int
	resetAt   time.Time
	err       error

	// Captured call arguments for assertions.
	calledKey    string
	calledLimit  int
	calledWindow time.Duration
	callCount    int
}

func (m *mockRateLimiter) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, int, time.Time, error) {
	m.calledKey = key
	m.calledLimit = limit
	m.calledWindow = window
	m.callCount++
	return m.allowed, m.remaining, m.resetAt, m.err
}

// --- Helpers ---

// setupRateLimitEcho creates an Echo instance with the rate limit middleware
// and a simple handler that returns 200. Optionally sets api_key_id in context
// via a preceding middleware to simulate the auth middleware.
func setupRateLimitEcho(config RateLimitConfig, apiKeyID interface{}) *echo.Echo {
	e := echo.New()

	// Simulate auth middleware setting api_key_id.
	setCtx := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if apiKeyID != nil {
				c.Set("api_key_id", apiKeyID)
			}
			return next(c)
		}
	}

	e.Use(setCtx)
	e.Use(RateLimit(config))

	e.GET("/api/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	return e
}

func doRequest(e *echo.Echo) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests ---

func TestRateLimit_UnderLimit(t *testing.T) {
	resetAt := time.Now().Add(30 * time.Second).Truncate(time.Second)
	keyID := uuid.New()

	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 55,
		resetAt:   resetAt,
	}

	config := RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60,
		Window:       time.Minute,
		KeyOverrides: make(map[string]int),
	}

	e := setupRateLimitEcho(config, keyID)
	rec := doRequest(e)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify rate limit headers are set.
	assert.Equal(t, "60", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "55", rec.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, strconv.FormatInt(resetAt.Unix(), 10), rec.Header().Get("X-RateLimit-Reset"))

	// Verify limiter was called with the right arguments.
	assert.Equal(t, keyID.String(), limiter.calledKey)
	assert.Equal(t, 60, limiter.calledLimit)
	assert.Equal(t, time.Minute, limiter.calledWindow)
}

func TestRateLimit_AtLimit_Returns429(t *testing.T) {
	resetAt := time.Now().Add(45 * time.Second).Truncate(time.Second)
	keyID := uuid.New()

	limiter := &mockRateLimiter{
		allowed:   false,
		remaining: 0,
		resetAt:   resetAt,
	}

	config := RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60,
		Window:       time.Minute,
		KeyOverrides: make(map[string]int),
	}

	e := setupRateLimitEcho(config, keyID)
	rec := doRequest(e)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Check Retry-After header is present and positive.
	retryAfter := rec.Header().Get("Retry-After")
	require.NotEmpty(t, retryAfter)
	retryAfterSec, err := strconv.ParseInt(retryAfter, 10, 64)
	require.NoError(t, err)
	assert.Greater(t, retryAfterSec, int64(0))

	// Check rate limit headers.
	assert.Equal(t, "60", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", rec.Header().Get("X-RateLimit-Remaining"))
	assert.Equal(t, strconv.FormatInt(resetAt.Unix(), 10), rec.Header().Get("X-RateLimit-Reset"))

	// Check JSON body.
	var body rateLimitExceededResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "rate_limit_exceeded", body.Error)
	assert.Equal(t, "Too many requests", body.Message)
	assert.Greater(t, body.RetryAfter, int64(0))
}

func TestRateLimit_NoAPIKeyID_SkipsRateLimiting(t *testing.T) {
	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 59,
		resetAt:   time.Now().Add(30 * time.Second),
	}

	config := RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60,
		Window:       time.Minute,
		KeyOverrides: make(map[string]int),
	}

	// Pass nil as apiKeyID so it's not set in context.
	e := setupRateLimitEcho(config, nil)
	rec := doRequest(e)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Limiter should NOT have been called.
	assert.Equal(t, 0, limiter.callCount)

	// No rate limit headers should be set.
	assert.Empty(t, rec.Header().Get("X-RateLimit-Limit"))
	assert.Empty(t, rec.Header().Get("X-RateLimit-Remaining"))
	assert.Empty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestRateLimit_HeadersAlwaysSet(t *testing.T) {
	resetAt := time.Now().Add(10 * time.Second).Truncate(time.Second)
	keyID := uuid.New()

	tests := []struct {
		name      string
		allowed   bool
		remaining int
		limit     int
	}{
		{"first request", true, 59, 60},
		{"half used", true, 30, 60},
		{"last allowed", true, 0, 60},
		{"exceeded", false, 0, 60},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			limiter := &mockRateLimiter{
				allowed:   tc.allowed,
				remaining: tc.remaining,
				resetAt:   resetAt,
			}

			config := RateLimitConfig{
				Limiter:      limiter,
				DefaultLimit: tc.limit,
				Window:       time.Minute,
				KeyOverrides: make(map[string]int),
			}

			e := setupRateLimitEcho(config, keyID)
			rec := doRequest(e)

			// X-RateLimit-* headers must always be present.
			assert.Equal(t, strconv.Itoa(tc.limit), rec.Header().Get("X-RateLimit-Limit"))
			assert.Equal(t, strconv.Itoa(tc.remaining), rec.Header().Get("X-RateLimit-Remaining"))
			assert.Equal(t, strconv.FormatInt(resetAt.Unix(), 10), rec.Header().Get("X-RateLimit-Reset"))
		})
	}
}

func TestRateLimit_KeyOverrides(t *testing.T) {
	resetAt := time.Now().Add(30 * time.Second).Truncate(time.Second)
	keyID := uuid.New()

	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 295,
		resetAt:   resetAt,
	}

	config := RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60,
		Window:       time.Minute,
		KeyOverrides: map[string]int{
			keyID.String(): 300, // Paid plan override.
		},
	}

	e := setupRateLimitEcho(config, keyID)
	rec := doRequest(e)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Limiter should have received the overridden limit.
	assert.Equal(t, 300, limiter.calledLimit)

	// Headers should reflect the overridden limit.
	assert.Equal(t, "300", rec.Header().Get("X-RateLimit-Limit"))
}

func TestRateLimit_StringAPIKeyID(t *testing.T) {
	resetAt := time.Now().Add(30 * time.Second).Truncate(time.Second)
	keyIDStr := "custom-key-id-123"

	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 59,
		resetAt:   resetAt,
	}

	config := RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60,
		Window:       time.Minute,
		KeyOverrides: make(map[string]int),
	}

	e := setupRateLimitEcho(config, keyIDStr)
	rec := doRequest(e)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, keyIDStr, limiter.calledKey)
}

func TestRateLimit_LimiterError_FailsOpen(t *testing.T) {
	keyID := uuid.New()

	limiter := &mockRateLimiter{
		err: assert.AnError,
	}

	config := RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60,
		Window:       time.Minute,
		KeyOverrides: make(map[string]int),
	}

	e := setupRateLimitEcho(config, keyID)
	rec := doRequest(e)

	// Should fail open: request goes through.
	assert.Equal(t, http.StatusOK, rec.Code)

	// Limiter was called.
	assert.Equal(t, 1, limiter.callCount)
}

func TestRateLimit_DefaultConfig(t *testing.T) {
	limiter := &mockRateLimiter{
		allowed:   true,
		remaining: 59,
		resetAt:   time.Now().Add(30 * time.Second),
	}

	config := DefaultRateLimitConfig(limiter)

	assert.Equal(t, 60, config.DefaultLimit)
	assert.Equal(t, time.Minute, config.Window)
	assert.NotNil(t, config.KeyOverrides)
	assert.Equal(t, limiter, config.Limiter)
}

// --- Redis implementation tests (unit, no real Redis) ---

type mockRedisClient struct {
	incrResult int64
	incrErr    error
	expireErr  error
	ttlResult  time.Duration
	ttlErr     error

	incrCalledKey    string
	expireCalledKey  string
	expireCalledTTL  time.Duration
	incrCallCount    int
	expireCallCount  int
}

func (m *mockRedisClient) Incr(_ context.Context, key string) (int64, error) {
	m.incrCalledKey = key
	m.incrCallCount++
	return m.incrResult, m.incrErr
}

func (m *mockRedisClient) Expire(_ context.Context, key string, ttl time.Duration) error {
	m.expireCalledKey = key
	m.expireCalledTTL = ttl
	m.expireCallCount++
	return m.expireErr
}

func (m *mockRedisClient) TTL(_ context.Context, _ string) (time.Duration, error) {
	return m.ttlResult, m.ttlErr
}

func TestRedisRateLimiter_FirstRequest_SetsExpiry(t *testing.T) {
	client := &mockRedisClient{
		incrResult: 1, // First request in window.
	}

	limiter := NewRedisRateLimiter(client)
	allowed, remaining, resetAt, err := limiter.Allow(context.Background(), "key-123", 60, time.Minute)

	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 59, remaining)
	assert.False(t, resetAt.IsZero())

	// Expire should have been called for the first request.
	assert.Equal(t, 1, client.expireCallCount)
	assert.Equal(t, time.Minute, client.expireCalledTTL)
}

func TestRedisRateLimiter_SubsequentRequest_NoExpiry(t *testing.T) {
	client := &mockRedisClient{
		incrResult: 5, // 5th request in window.
	}

	limiter := NewRedisRateLimiter(client)
	allowed, remaining, _, err := limiter.Allow(context.Background(), "key-123", 60, time.Minute)

	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, 55, remaining)

	// Expire should NOT have been called (count > 1).
	assert.Equal(t, 0, client.expireCallCount)
}

func TestRedisRateLimiter_ExceedsLimit(t *testing.T) {
	client := &mockRedisClient{
		incrResult: 61, // Over the 60 limit.
	}

	limiter := NewRedisRateLimiter(client)
	allowed, remaining, _, err := limiter.Allow(context.Background(), "key-123", 60, time.Minute)

	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining) // Clamped to 0, not negative.
}

func TestRedisRateLimiter_IncrError(t *testing.T) {
	client := &mockRedisClient{
		incrErr: assert.AnError,
	}

	limiter := NewRedisRateLimiter(client)
	_, _, _, err := limiter.Allow(context.Background(), "key-123", 60, time.Minute)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis incr")
}

func TestRedisRateLimiter_ExpireError(t *testing.T) {
	client := &mockRedisClient{
		incrResult: 1, // First request triggers Expire.
		expireErr:  assert.AnError,
	}

	limiter := NewRedisRateLimiter(client)
	_, _, _, err := limiter.Allow(context.Background(), "key-123", 60, time.Minute)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "redis expire")
}

func TestRedisRateLimiter_KeyFormat(t *testing.T) {
	client := &mockRedisClient{
		incrResult: 1,
	}

	limiter := NewRedisRateLimiter(client)
	_, _, _, err := limiter.Allow(context.Background(), "my-api-key", 60, time.Minute)

	require.NoError(t, err)
	assert.Contains(t, client.incrCalledKey, "ratelimit:my-api-key:")
}
