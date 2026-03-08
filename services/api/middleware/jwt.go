package middleware

import (
	"net/http"
	"strings"

	"github.com/docbiner/docbiner/internal/auth"
	"github.com/labstack/echo/v4"
)

// TokenValidator abstracts JWT validation so the middleware stays testable.
type TokenValidator interface {
	Validate(token string) (*auth.Claims, error)
}

// jwtUnauthorizedResponse is the standard 401 payload for JWT auth failures.
var jwtUnauthorizedResponse = map[string]string{
	"error":   "unauthorized",
	"message": "Invalid or missing JWT token",
}

// JWTAuth returns an Echo middleware that authenticates requests using
// JWT Bearer tokens. On success it sets "user_id", "org_id", and "role"
// on the Echo context for downstream handlers.
func JWTAuth(validator TokenValidator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Extract the Authorization header.
			header := c.Request().Header.Get("Authorization")
			if header == "" {
				return c.JSON(http.StatusUnauthorized, jwtUnauthorizedResponse)
			}

			// 2. Expect "Bearer <token>" format.
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, jwtUnauthorizedResponse)
			}

			tokenString := strings.TrimSpace(parts[1])
			if tokenString == "" {
				return c.JSON(http.StatusUnauthorized, jwtUnauthorizedResponse)
			}

			// 3. Validate the token.
			claims, err := validator.Validate(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, jwtUnauthorizedResponse)
			}

			// 4. Populate context values for downstream handlers.
			c.Set("user_id", claims.UserID)
			c.Set("org_id", claims.OrgID)
			c.Set("role", claims.Role)

			return next(c)
		}
	}
}
