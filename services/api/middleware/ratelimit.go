package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RateLimiter abstracts the rate limiting backend so the middleware stays
// testable without a real Redis connection.
type RateLimiter interface {
	// Allow checks whether a request identified by key is allowed under the
	// given limit and window. It returns whether the request is allowed, how
	// many requests remain in the current window, and when the window resets.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, resetAt time.Time, err error)
}

// RedisRateLimiter implements RateLimiter using a Redis-backed fixed-window
// counter. The key format is ratelimit:{api_key_id}:{window_timestamp}.
type RedisRateLimiter struct {
	client RedisClient
}

// RedisClient is the minimal Redis interface needed by RedisRateLimiter.
type RedisClient interface {
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
}

// NewRedisRateLimiter creates a new RedisRateLimiter with the given client.
func NewRedisRateLimiter(client RedisClient) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// Allow implements the RateLimiter interface using a fixed-window counter.
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Time, error) {
	windowSec := int64(window.Seconds())
	now := time.Now()
	windowStart := now.Unix() / windowSec * windowSec
	resetAt := time.Unix(windowStart+windowSec, 0)

	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, windowStart)

	count, err := r.client.Incr(ctx, redisKey)
	if err != nil {
		return false, 0, resetAt, fmt.Errorf("redis incr: %w", err)
	}

	// Set expiry on first increment to auto-cleanup.
	if count == 1 {
		if err := r.client.Expire(ctx, redisKey, window); err != nil {
			return false, 0, resetAt, fmt.Errorf("redis expire: %w", err)
		}
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	allowed := int(count) <= limit

	return allowed, remaining, resetAt, nil
}

// RateLimitConfig holds configuration for the rate limiting middleware.
type RateLimitConfig struct {
	// Limiter is the rate limiter backend.
	Limiter RateLimiter
	// DefaultLimit is the default number of requests per window (free plan).
	DefaultLimit int
	// Window is the rate limiting window duration.
	Window time.Duration
	// KeyOverrides allows per-key limit overrides (e.g. for paid plans).
	KeyOverrides map[string]int
}

// DefaultRateLimitConfig returns a config with sensible defaults.
func DefaultRateLimitConfig(limiter RateLimiter) RateLimitConfig {
	return RateLimitConfig{
		Limiter:      limiter,
		DefaultLimit: 60, // Free plan: 60 req/min
		Window:       time.Minute,
		KeyOverrides: make(map[string]int),
	}
}

// rateLimitExceededResponse is the JSON body returned on 429.
type rateLimitExceededResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	RetryAfter int64  `json:"retry_after"`
}

// RateLimit returns an Echo middleware that enforces per-API-key rate limiting.
// It reads "api_key_id" from the Echo context (set by the auth middleware) and
// uses the provided RateLimiter to check and enforce limits.
//
// If no api_key_id is present in context, the middleware passes through without
// rate limiting (the request may have been authenticated by a different mechanism
// or the endpoint may not require authentication).
func RateLimit(config RateLimitConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Read api_key_id from the Echo context (set by auth middleware).
			raw := c.Get("api_key_id")
			if raw == nil {
				// No API key in context: skip rate limiting.
				return next(c)
			}

			var keyID string
			switch v := raw.(type) {
			case uuid.UUID:
				keyID = v.String()
			case string:
				keyID = v
			default:
				keyID = fmt.Sprintf("%v", v)
			}

			// Determine the limit for this key.
			limit := config.DefaultLimit
			if override, ok := config.KeyOverrides[keyID]; ok {
				limit = override
			}

			allowed, remaining, resetAt, err := config.Limiter.Allow(
				c.Request().Context(),
				keyID,
				limit,
				config.Window,
			)
			if err != nil {
				// On rate limiter errors, fail open: let the request through
				// but log the error. In production this should be wired to a
				// logger.
				return next(c)
			}

			// Set rate limit headers on every response.
			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Response().Header().Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

			if !allowed {
				retryAfter := int64(time.Until(resetAt).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}

				c.Response().Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))

				return c.JSON(http.StatusTooManyRequests, rateLimitExceededResponse{
					Error:      "rate_limit_exceeded",
					Message:    "Too many requests",
					RetryAfter: retryAfter,
				})
			}

			return next(c)
		}
	}
}
