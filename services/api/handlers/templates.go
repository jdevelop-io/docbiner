package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// --- Interfaces for testability ---

// TemplateStore abstracts template persistence for the templates handler.
type TemplateStore interface {
	Create(ctx context.Context, params CreateTemplateParams) (*domain.Template, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Template, error)
	Update(ctx context.Context, id uuid.UUID, params UpdateTemplateParams) (*domain.Template, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// TemplateRenderer abstracts template rendering for the preview endpoint.
type TemplateRenderer interface {
	Render(engine string, template string, data map[string]interface{}) (string, error)
}

// --- Request/Response structs ---

// CreateTemplateParams holds parameters for creating a new template.
type CreateTemplateParams struct {
	OrgID       uuid.UUID
	CreatedBy   uuid.UUID
	Name        string
	Engine      domain.TemplateEngine
	HTMLContent string
	CSSContent  *string
	SampleData  []byte
}

// UpdateTemplateParams holds parameters for updating a template.
type UpdateTemplateParams struct {
	Name        *string
	Engine      *domain.TemplateEngine
	HTMLContent *string
	CSSContent  *string
	SampleData  []byte
}

// CreateTemplateRequest is the JSON body for POST /v1/templates.
type CreateTemplateRequest struct {
	Name        string                 `json:"name"`
	Engine      string                 `json:"engine"`
	HTMLContent string                 `json:"html_content"`
	CSSContent  string                 `json:"css_content"`
	SampleData  map[string]interface{} `json:"sample_data"`
}

// UpdateTemplateRequest is the JSON body for PUT /v1/templates/:id.
type UpdateTemplateRequest struct {
	Name        *string                `json:"name"`
	Engine      *string                `json:"engine"`
	HTMLContent *string                `json:"html_content"`
	CSSContent  *string                `json:"css_content"`
	SampleData  map[string]interface{} `json:"sample_data"`
}

// PreviewInlineRequest is the JSON body for POST /v1/templates/preview (no saved template).
type PreviewInlineRequest struct {
	Engine     string                 `json:"engine"`
	HTMLContent string               `json:"html_content"`
	CSSContent  string               `json:"css_content"`
	Data        map[string]interface{} `json:"data"`
}

// PreviewTemplateRequest is the JSON body for POST /v1/templates/:id/preview.
type PreviewTemplateRequest struct {
	Data map[string]interface{} `json:"data"`
}

// PreviewTemplateResponse is the response for POST /v1/templates/:id/preview.
type PreviewTemplateResponse struct {
	HTML string `json:"html"`
}

// --- Valid engines ---

var validEngines = map[string]domain.TemplateEngine{
	"handlebars": domain.TemplateEngineHandlebars,
	"liquid":     domain.TemplateEngineLiquid,
}

// --- Handler ---

// TemplateHandler handles template CRUD and preview requests.
type TemplateHandler struct {
	store    TemplateStore
	renderer TemplateRenderer
}

// NewTemplateHandler creates a new TemplateHandler.
func NewTemplateHandler(s TemplateStore, r TemplateRenderer) *TemplateHandler {
	return &TemplateHandler{
		store:    s,
		renderer: r,
	}
}

// Create handles POST /v1/templates.
func (h *TemplateHandler) Create(c echo.Context) error {
	var req CreateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate required fields.
	if strings.TrimSpace(req.Name) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "name is required",
		})
	}

	if strings.TrimSpace(req.HTMLContent) == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "html_content is required",
		})
	}

	engine, ok := validEngines[req.Engine]
	if !ok {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid engine: must be one of handlebars, liquid",
		})
	}

	// Extract auth context values.
	orgID, _ := c.Get("org_id").(uuid.UUID)
	userID, _ := c.Get("user_id").(uuid.UUID)

	// Serialize sample_data.
	var sampleDataJSON []byte
	if req.SampleData != nil {
		sampleDataJSON, _ = json.Marshal(req.SampleData)
	}

	// Prepare CSS content.
	var cssContent *string
	if req.CSSContent != "" {
		cssContent = &req.CSSContent
	}

	tpl, err := h.store.Create(c.Request().Context(), CreateTemplateParams{
		OrgID:       orgID,
		CreatedBy:   userID,
		Name:        req.Name,
		Engine:      engine,
		HTMLContent: req.HTMLContent,
		CSSContent:  cssContent,
		SampleData:  sampleDataJSON,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create template",
		})
	}

	return c.JSON(http.StatusCreated, tpl)
}

// List handles GET /v1/templates.
func (h *TemplateHandler) List(c echo.Context) error {
	orgID, _ := c.Get("org_id").(uuid.UUID)

	templates, err := h.store.ListByOrg(c.Request().Context(), orgID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list templates",
		})
	}

	// Return empty array instead of null.
	if templates == nil {
		templates = []*domain.Template{}
	}

	return c.JSON(http.StatusOK, templates)
}

// Get handles GET /v1/templates/:id.
func (h *TemplateHandler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid template ID",
		})
	}

	tpl, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Template not found",
		})
	}

	return c.JSON(http.StatusOK, tpl)
}

// Update handles PUT /v1/templates/:id.
func (h *TemplateHandler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid template ID",
		})
	}

	var req UpdateTemplateRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	// Validate engine if provided.
	var enginePtr *domain.TemplateEngine
	if req.Engine != nil {
		engine, ok := validEngines[*req.Engine]
		if !ok {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "validation_error",
				Message: "Invalid engine: must be one of handlebars, liquid",
			})
		}
		enginePtr = &engine
	}

	// Serialize sample_data.
	var sampleDataJSON []byte
	if req.SampleData != nil {
		sampleDataJSON, _ = json.Marshal(req.SampleData)
	}

	tpl, err := h.store.Update(c.Request().Context(), id, UpdateTemplateParams{
		Name:        req.Name,
		Engine:      enginePtr,
		HTMLContent: req.HTMLContent,
		CSSContent:  req.CSSContent,
		SampleData:  sampleDataJSON,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Template not found",
		})
	}

	return c.JSON(http.StatusOK, tpl)
}

// Delete handles DELETE /v1/templates/:id.
func (h *TemplateHandler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid template ID",
		})
	}

	if err := h.store.Delete(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Template not found",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// Preview handles POST /v1/templates/:id/preview.
func (h *TemplateHandler) Preview(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "Invalid template ID",
		})
	}

	// Get the template.
	tpl, err := h.store.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Template not found",
		})
	}

	// Parse request body for data.
	var req PreviewTemplateRequest
	if c.Request().ContentLength > 0 {
		if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
			return c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "bad_request",
				Message: "Invalid request body",
			})
		}
	}

	// Use provided data, or fall back to sample_data.
	data := req.Data
	if data == nil && len(tpl.SampleData) > 0 {
		_ = json.Unmarshal(tpl.SampleData, &data)
	}

	// Build full HTML with optional CSS.
	htmlContent := tpl.HTMLContent
	if tpl.CSSContent != nil && *tpl.CSSContent != "" {
		htmlContent = "<style>" + *tpl.CSSContent + "</style>" + htmlContent
	}

	// Render template.
	rendered, err := h.renderer.Render(string(tpl.Engine), htmlContent, data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "render_error",
			Message: "Failed to render template",
		})
	}

	return c.JSON(http.StatusOK, PreviewTemplateResponse{
		HTML: rendered,
	})
}

// PreviewInline handles POST /v1/templates/preview — renders without a saved template.
func (h *TemplateHandler) PreviewInline(c echo.Context) error {
	var req PreviewInlineRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "bad_request",
			Message: "Invalid request body",
		})
	}

	if req.Engine == "" || req.HTMLContent == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: "engine and html_content are required",
		})
	}

	htmlContent := req.HTMLContent
	if req.CSSContent != "" {
		htmlContent = "<style>" + req.CSSContent + "</style>" + htmlContent
	}

	rendered, err := h.renderer.Render(req.Engine, htmlContent, req.Data)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "render_error",
			Message: "Failed to render template",
		})
	}

	return c.JSON(http.StatusOK, PreviewTemplateResponse{
		HTML: rendered,
	})
}
