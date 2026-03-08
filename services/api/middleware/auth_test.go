package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/docbiner/docbiner/internal/apikey"
	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock implementation of KeyLookup ---

type mockKeyLookup struct {
	keys map[string]*domain.APIKey // keyed by hash

	mu             sync.Mutex
	lastUsedCalled []uuid.UUID
}

func newMockKeyLookup() *mockKeyLookup {
	return &mockKeyLookup{
		keys: make(map[string]*domain.APIKey),
	}
}

func (m *mockKeyLookup) addKey(rawKey string, key *domain.APIKey) {
	hash := apikey.Hash(rawKey)
	key.KeyHash = hash
	m.keys[hash] = key
}

func (m *mockKeyLookup) GetByHash(_ context.Context, keyHash string) (*domain.APIKey, error) {
	k, ok := m.keys[keyHash]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return k, nil
}

func (m *mockKeyLookup) UpdateLastUsed(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastUsedCalled = append(m.lastUsedCalled, id)
	return nil
}

// --- Helpers ---

// setupEcho creates an Echo instance with the auth middleware and a simple
// handler that returns 200. It returns the engine so callers can execute
// requests against it.
func setupEcho(lookup KeyLookup) *echo.Echo {
	e := echo.New()
	g := e.Group("/api", APIKeyAuth(lookup))
	g.GET("/test", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"org_id":     c.Get("org_id"),
			"api_key_id": c.Get("api_key_id"),
			"environment": c.Get("environment"),
		})
	})
	return e
}

// --- Tests ---

func TestAPIKeyAuth_ValidKey(t *testing.T) {
	mock := newMockKeyLookup()

	orgID := uuid.New()
	keyID := uuid.New()
	rawKey := "db_live_abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	mock.addKey(rawKey, &domain.APIKey{
		ID:          keyID,
		OrgID:       orgID,
		Environment: domain.APIKeyEnvLive,
	})

	e := setupEcho(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, orgID.String(), body["org_id"])
	assert.Equal(t, keyID.String(), body["api_key_id"])
	assert.Equal(t, string(domain.APIKeyEnvLive), body["environment"])
}

func TestAPIKeyAuth_MissingAuthorizationHeader(t *testing.T) {
	mock := newMockKeyLookup()
	e := setupEcho(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	// No Authorization header
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assertUnauthorizedBody(t, rec)
}

func TestAPIKeyAuth_InvalidBearerFormat(t *testing.T) {
	mock := newMockKeyLookup()
	e := setupEcho(mock)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "Token db_live_abc123"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"bearer with no token", "Bearer "},
		{"bearer only", "Bearer"},
		{"empty value", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assertUnauthorizedBody(t, rec)
		})
	}
}

func TestAPIKeyAuth_UnknownKey(t *testing.T) {
	mock := newMockKeyLookup()
	e := setupEcho(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer db_live_unknown_key_that_does_not_exist")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assertUnauthorizedBody(t, rec)
}

func TestAPIKeyAuth_TestEnvironmentKey(t *testing.T) {
	mock := newMockKeyLookup()

	orgID := uuid.New()
	keyID := uuid.New()
	rawKey := "db_test_abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	mock.addKey(rawKey, &domain.APIKey{
		ID:          keyID,
		OrgID:       orgID,
		Environment: domain.APIKeyEnvTest,
	})

	e := setupEcho(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+rawKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, orgID.String(), body["org_id"])
	assert.Equal(t, keyID.String(), body["api_key_id"])
	assert.Equal(t, string(domain.APIKeyEnvTest), body["environment"])
}

func TestAPIKeyAuth_CaseInsensitiveBearer(t *testing.T) {
	mock := newMockKeyLookup()

	orgID := uuid.New()
	keyID := uuid.New()
	rawKey := "db_live_caseinsensitivetest1234567890abcdef1234567890abcdef123456"

	mock.addKey(rawKey, &domain.APIKey{
		ID:          keyID,
		OrgID:       orgID,
		Environment: domain.APIKeyEnvLive,
	})

	e := setupEcho(mock)

	// Use "bearer" (lowercase) instead of "Bearer"
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	req.Header.Set("Authorization", "bearer "+rawKey)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// assertUnauthorizedBody verifies the JSON body matches the expected 401 response.
func assertUnauthorizedBody(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "unauthorized", body["error"])
	assert.Equal(t, "Invalid or missing API key", body["message"])
}
