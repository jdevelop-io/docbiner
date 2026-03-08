package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/docbiner/docbiner/internal/apikey"
	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// KeyLookup abstracts the API key store so the middleware stays testable
// without a real database connection.
type KeyLookup interface {
	GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error)
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// unauthorizedResponse is the standard 401 payload.
var unauthorizedResponse = map[string]string{
	"error":   "unauthorized",
	"message": "Invalid or missing API key",
}

// APIKeyAuth returns an Echo middleware that authenticates requests using
// Bearer API keys. On success it sets "org_id", "api_key_id", and
// "environment" on the Echo context for downstream handlers.
func APIKeyAuth(lookup KeyLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Extract the Authorization header.
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return c.JSON(http.StatusUnauthorized, unauthorizedResponse)
			}

			// 2. Expect "Bearer <token>" format.
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, unauthorizedResponse)
			}

			rawKey := parts[1]
			if rawKey == "" {
				return c.JSON(http.StatusUnauthorized, unauthorizedResponse)
			}

			// 3. Hash the raw key and look it up.
			hash := apikey.Hash(rawKey)

			key, err := lookup.GetByHash(c.Request().Context(), hash)
			if err != nil || key == nil {
				return c.JSON(http.StatusUnauthorized, unauthorizedResponse)
			}

			// 4. Populate context values for downstream handlers.
			c.Set("org_id", key.OrgID)
			c.Set("api_key_id", key.ID)
			c.Set("environment", key.Environment)

			// 5. Update last_used_at asynchronously (fire-and-forget).
			go func(id uuid.UUID) {
				_ = lookup.UpdateLastUsed(context.Background(), id)
			}(key.ID)

			return next(c)
		}
	}
}
