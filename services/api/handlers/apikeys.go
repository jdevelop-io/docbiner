package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/docbiner/docbiner/internal/apikey"
	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Interfaces ---

// APIKeyStore abstracts API key persistence.
type APIKeyStore interface {
	Create(ctx context.Context, orgID, createdBy uuid.UUID, keyHash, keyPrefix, name string, env domain.APIKeyEnvironment) (*domain.APIKey, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.APIKey, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// --- Request/Response ---

// CreateAPIKeyRequest is the JSON body for POST /v1/api-keys.
type CreateAPIKeyRequest struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

// APIKeyResponse is a single key in list responses (never includes the raw key).
type APIKeyResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	KeyPrefix   string    `json:"key_prefix"`
	Environment string    `json:"environment"`
	CreatedAt   string    `json:"created_at"`
}

// CreateAPIKeyResponse includes the raw key shown only once.
type CreateAPIKeyResponse struct {
	APIKeyResponse
	Key string `json:"key"`
}

// --- Handler ---

// APIKeyHandler manages API key CRUD endpoints.
type APIKeyHandler struct {
	store APIKeyStore
}

// NewAPIKeyHandler creates a new APIKeyHandler.
func NewAPIKeyHandler(store APIKeyStore) *APIKeyHandler {
	return &APIKeyHandler{store: store}
}

// Create handles POST /v1/api-keys.
func (h *APIKeyHandler) Create(c echo.Context) error {
	var req CreateAPIKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if strings.TrimSpace(req.Name) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Name is required",
		})
	}

	env := domain.APIKeyEnvLive
	if req.Environment == "test" {
		env = domain.APIKeyEnvTest
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing user context",
		})
	}
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing org context",
		})
	}

	generated, err := apikey.Generate(string(env))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to generate API key",
		})
	}

	ctx := c.Request().Context()
	key, err := h.store.Create(ctx, orgID, userID, generated.Hash, generated.Prefix, req.Name, env)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create API key",
		})
	}

	return c.JSON(http.StatusCreated, CreateAPIKeyResponse{
		APIKeyResponse: APIKeyResponse{
			ID:          key.ID,
			Name:        key.Name,
			KeyPrefix:   key.KeyPrefix,
			Environment: string(key.Environment),
			CreatedAt:   key.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
		Key: generated.Raw,
	})
}

// List handles GET /v1/api-keys.
func (h *APIKeyHandler) List(c echo.Context) error {
	orgID, ok := c.Get("org_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Missing org context",
		})
	}

	ctx := c.Request().Context()
	keys, err := h.store.ListByOrg(ctx, orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list API keys",
		})
	}

	resp := make([]APIKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = APIKeyResponse{
			ID:          k.ID,
			Name:        k.Name,
			KeyPrefix:   k.KeyPrefix,
			Environment: string(k.Environment),
			CreatedAt:   k.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return c.JSON(http.StatusOK, resp)
}

// Delete handles DELETE /v1/api-keys/:id.
func (h *APIKeyHandler) Delete(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid API key ID",
		})
	}

	ctx := c.Request().Context()
	if err := h.store.Delete(ctx, id); err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete API key",
		})
	}

	return c.NoContent(http.StatusNoContent)
}
