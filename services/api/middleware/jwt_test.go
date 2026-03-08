package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/auth"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

// setupJWTEcho creates an Echo instance with the JWT middleware and a handler
// that returns 200 with the context values.
func setupJWTEcho(validator TokenValidator) *echo.Echo {
	e := echo.New()
	g := e.Group("/dashboard", JWTAuth(validator))
	g.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id": c.Get("user_id"),
			"org_id":  c.Get("org_id"),
			"role":    c.Get("role"),
		})
	})
	return e
}

// --- Tests ---

func TestJWTAuth_ValidToken(t *testing.T) {
	secret := "test-jwt-secret"
	svc := auth.New(secret, 1*time.Hour)

	userID := uuid.New()
	orgID := uuid.New()
	role := "owner"

	token, err := svc.Generate(userID, orgID, role)
	require.NoError(t, err)

	e := setupJWTEcho(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, userID.String(), body["user_id"])
	assert.Equal(t, orgID.String(), body["org_id"])
	assert.Equal(t, role, body["role"])
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	svc := auth.New("test-secret", 1*time.Hour)
	e := setupJWTEcho(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
	// No Authorization header.
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assertJWTUnauthorizedBody(t, rec)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	svc := auth.New("test-secret", 1*time.Hour)
	e := setupJWTEcho(svc)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-value")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assertJWTUnauthorizedBody(t, rec)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	// Generate a token that is already expired.
	generator := auth.New("test-secret", -1*time.Hour)

	token, err := generator.Generate(uuid.New(), uuid.New(), "member")
	require.NoError(t, err)

	// Validate with a service that has the same secret (but valid expiration).
	// The token itself is expired, so validation should fail.
	validator := auth.New("test-secret", 1*time.Hour)
	e := setupJWTEcho(validator)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assertJWTUnauthorizedBody(t, rec)
}

func TestJWTAuth_WrongSecret(t *testing.T) {
	generator := auth.New("secret-one", 1*time.Hour)
	validator := auth.New("secret-two", 1*time.Hour)

	token, err := generator.Generate(uuid.New(), uuid.New(), "member")
	require.NoError(t, err)

	e := setupJWTEcho(validator)

	req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assertJWTUnauthorizedBody(t, rec)
}

func TestJWTAuth_InvalidBearerFormat(t *testing.T) {
	svc := auth.New("test-secret", 1*time.Hour)
	e := setupJWTEcho(svc)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "Token some-jwt-token"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"bearer with no token", "Bearer "},
		{"bearer only", "Bearer"},
		{"empty value", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assertJWTUnauthorizedBody(t, rec)
		})
	}
}

func TestJWTAuth_CaseInsensitiveBearer(t *testing.T) {
	secret := "test-jwt-secret"
	svc := auth.New(secret, 1*time.Hour)

	token, err := svc.Generate(uuid.New(), uuid.New(), "admin")
	require.NoError(t, err)

	e := setupJWTEcho(svc)

	// Use "bearer" (lowercase) instead of "Bearer".
	req := httptest.NewRequest(http.MethodGet, "/dashboard/test", nil)
	req.Header.Set("Authorization", "bearer "+token)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// assertJWTUnauthorizedBody verifies the JSON body matches the expected 401 response.
func assertJWTUnauthorizedBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "Invalid or missing JWT token", body["message"])
}
