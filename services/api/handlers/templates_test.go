package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Template Store ---

type mockTemplateStore struct {
	template    *domain.Template
	templates   []*domain.Template
	createErr   error
	getErr      error
	listErr     error
	updateErr   error
	deleteErr   error

	lastCreateParams CreateTemplateParams
	lastUpdateParams UpdateTemplateParams
	lastUpdateID     uuid.UUID
	lastDeleteID     uuid.UUID
}

func newMockTemplateStore() *mockTemplateStore {
	css := "h1 { color: blue; }"
	return &mockTemplateStore{
		template: &domain.Template{
			ID:          uuid.New(),
			OrgID:       uuid.New(),
			CreatedBy:   uuid.New(),
			Name:        "Test Template",
			Engine:      domain.TemplateEngineHandlebars,
			HTMLContent: "<h1>{{title}}</h1>",
			CSSContent:  &css,
			SampleData:  []byte(`{"title":"Sample"}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}

func (m *mockTemplateStore) Create(_ context.Context, params CreateTemplateParams) (*domain.Template, error) {
	m.lastCreateParams = params
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.template, nil
}

func (m *mockTemplateStore) GetByID(_ context.Context, id uuid.UUID) (*domain.Template, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.template, nil
}

func (m *mockTemplateStore) ListByOrg(_ context.Context, orgID uuid.UUID) ([]*domain.Template, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.templates != nil {
		return m.templates, nil
	}
	return []*domain.Template{m.template}, nil
}

func (m *mockTemplateStore) Update(_ context.Context, id uuid.UUID, params UpdateTemplateParams) (*domain.Template, error) {
	m.lastUpdateID = id
	m.lastUpdateParams = params
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.template, nil
}

func (m *mockTemplateStore) Delete(_ context.Context, id uuid.UUID) error {
	m.lastDeleteID = id
	return m.deleteErr
}

// --- Mock Template Renderer ---

type mockTemplateRenderer struct {
	result string
	err    error

	lastEngine   string
	lastTemplate string
	lastData     map[string]interface{}
}

func (m *mockTemplateRenderer) Render(engine string, template string, data map[string]interface{}) (string, error) {
	m.lastEngine = engine
	m.lastTemplate = template
	m.lastData = data
	if m.err != nil {
		return "", m.err
	}
	return m.result, nil
}

// --- Test Helpers ---

func setupTemplateTest(s TemplateStore, r TemplateRenderer) *echo.Echo {
	e := echo.New()
	h := NewTemplateHandler(s, r)

	orgID := uuid.New()
	userID := uuid.New()

	setAuth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("org_id", orgID)
			c.Set("user_id", userID)
			c.Set("environment", domain.APIKeyEnvLive)
			return next(c)
		}
	}

	v1 := e.Group("/v1", setAuth)
	v1.POST("/templates", h.Create)
	v1.GET("/templates", h.List)
	v1.GET("/templates/:id", h.Get)
	v1.PUT("/templates/:id", h.Update)
	v1.DELETE("/templates/:id", h.Delete)
	v1.POST("/templates/:id/preview", h.Preview)

	return e
}

func doRequest(e *echo.Echo, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests: Create ---

func TestTemplates_Create_Success(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{
		"name": "Invoice",
		"engine": "handlebars",
		"html_content": "<h1>{{title}}</h1>",
		"css_content": "h1 { color: blue; }",
		"sample_data": {"title": "My Invoice"}
	}`
	rec := doRequest(e, http.MethodPost, "/v1/templates", body)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var tpl domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tpl))
	assert.NotEmpty(t, tpl.ID)
	assert.Equal(t, "Invoice", store.lastCreateParams.Name)
	assert.Equal(t, domain.TemplateEngineHandlebars, store.lastCreateParams.Engine)
	assert.Equal(t, "<h1>{{title}}</h1>", store.lastCreateParams.HTMLContent)
	assert.NotNil(t, store.lastCreateParams.CSSContent)
	assert.NotNil(t, store.lastCreateParams.SampleData)
}

func TestTemplates_Create_MissingName(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"engine": "handlebars", "html_content": "<h1>test</h1>"}`
	rec := doRequest(e, http.MethodPost, "/v1/templates", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "name is required")
}

func TestTemplates_Create_MissingHTMLContent(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"name": "Test", "engine": "handlebars"}`
	rec := doRequest(e, http.MethodPost, "/v1/templates", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "html_content is required")
}

func TestTemplates_Create_InvalidEngine(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"name": "Test", "engine": "jinja", "html_content": "<h1>test</h1>"}`
	rec := doRequest(e, http.MethodPost, "/v1/templates", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "Invalid engine")
}

func TestTemplates_Create_StoreError(t *testing.T) {
	store := newMockTemplateStore()
	store.createErr = errors.New("db error")
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"name": "Test", "engine": "handlebars", "html_content": "<h1>test</h1>"}`
	rec := doRequest(e, http.MethodPost, "/v1/templates", body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestTemplates_Create_InvalidJSON(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{not valid json}`
	rec := doRequest(e, http.MethodPost, "/v1/templates", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Tests: List ---

func TestTemplates_List_Success(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodGet, "/v1/templates", "")

	assert.Equal(t, http.StatusOK, rec.Code)

	var templates []*domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &templates))
	assert.Len(t, templates, 1)
}

func TestTemplates_List_Empty(t *testing.T) {
	store := newMockTemplateStore()
	store.templates = []*domain.Template{}
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodGet, "/v1/templates", "")

	assert.Equal(t, http.StatusOK, rec.Code)

	var templates []*domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &templates))
	assert.Len(t, templates, 0)
}

func TestTemplates_List_StoreError(t *testing.T) {
	store := newMockTemplateStore()
	store.listErr = errors.New("db error")
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodGet, "/v1/templates", "")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// --- Tests: Get ---

func TestTemplates_Get_Success(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodGet, "/v1/templates/"+store.template.ID.String(), "")

	assert.Equal(t, http.StatusOK, rec.Code)

	var tpl domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tpl))
	assert.Equal(t, store.template.ID, tpl.ID)
}

func TestTemplates_Get_NotFound(t *testing.T) {
	store := newMockTemplateStore()
	store.getErr = errors.New("not found")
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodGet, "/v1/templates/"+uuid.New().String(), "")

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestTemplates_Get_InvalidID(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodGet, "/v1/templates/not-a-uuid", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Tests: Update ---

func TestTemplates_Update_Success(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"name": "Updated Name"}`
	rec := doRequest(e, http.MethodPut, "/v1/templates/"+store.template.ID.String(), body)

	assert.Equal(t, http.StatusOK, rec.Code)

	var tpl domain.Template
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tpl))
	assert.NotNil(t, store.lastUpdateParams.Name)
	assert.Equal(t, "Updated Name", *store.lastUpdateParams.Name)
}

func TestTemplates_Update_NotFound(t *testing.T) {
	store := newMockTemplateStore()
	store.updateErr = errors.New("not found")
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"name": "Updated"}`
	rec := doRequest(e, http.MethodPut, "/v1/templates/"+uuid.New().String(), body)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTemplates_Update_InvalidEngine(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	body := `{"engine": "jinja"}`
	rec := doRequest(e, http.MethodPut, "/v1/templates/"+store.template.ID.String(), body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "Invalid engine")
}

// --- Tests: Delete ---

func TestTemplates_Delete_Success(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodDelete, "/v1/templates/"+store.template.ID.String(), "")

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, store.template.ID, store.lastDeleteID)
}

func TestTemplates_Delete_NotFound(t *testing.T) {
	store := newMockTemplateStore()
	store.deleteErr = errors.New("not found")
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodDelete, "/v1/templates/"+uuid.New().String(), "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTemplates_Delete_InvalidID(t *testing.T) {
	store := newMockTemplateStore()
	renderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, renderer)

	rec := doRequest(e, http.MethodDelete, "/v1/templates/not-a-uuid", "")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- Tests: Preview ---

func TestTemplates_Preview_WithData(t *testing.T) {
	store := newMockTemplateStore()
	tmplRenderer := &mockTemplateRenderer{result: "<h1>Preview Test</h1>"}
	e := setupTemplateTest(store, tmplRenderer)

	body := `{"data": {"title": "Preview Test"}}`
	rec := doRequest(e, http.MethodPost, "/v1/templates/"+store.template.ID.String()+"/preview", body)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp PreviewTemplateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "<h1>Preview Test</h1>", resp.HTML)

	// Verify the renderer was called with the right engine.
	assert.Equal(t, "handlebars", tmplRenderer.lastEngine)
	// Verify CSS was prepended to the template.
	assert.Contains(t, tmplRenderer.lastTemplate, "<style>")
	assert.Contains(t, tmplRenderer.lastTemplate, "h1 { color: blue; }")
	// Verify data was passed.
	assert.Equal(t, "Preview Test", tmplRenderer.lastData["title"])
}

func TestTemplates_Preview_WithSampleData(t *testing.T) {
	store := newMockTemplateStore()
	tmplRenderer := &mockTemplateRenderer{result: "<h1>Sample</h1>"}
	e := setupTemplateTest(store, tmplRenderer)

	// Send empty body (no data provided) — should use sample_data.
	body := `{}`
	rec := doRequest(e, http.MethodPost, "/v1/templates/"+store.template.ID.String()+"/preview", body)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp PreviewTemplateResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "<h1>Sample</h1>", resp.HTML)

	// Should have used sample_data from the template.
	assert.Equal(t, "Sample", tmplRenderer.lastData["title"])
}

func TestTemplates_Preview_TemplateNotFound(t *testing.T) {
	store := newMockTemplateStore()
	store.getErr = errors.New("not found")
	tmplRenderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, tmplRenderer)

	body := `{"data": {"title": "test"}}`
	rec := doRequest(e, http.MethodPost, "/v1/templates/"+uuid.New().String()+"/preview", body)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTemplates_Preview_RenderError(t *testing.T) {
	store := newMockTemplateStore()
	tmplRenderer := &mockTemplateRenderer{err: errors.New("render failed")}
	e := setupTemplateTest(store, tmplRenderer)

	body := `{"data": {"title": "test"}}`
	rec := doRequest(e, http.MethodPost, "/v1/templates/"+store.template.ID.String()+"/preview", body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "render_error", errResp.Error)
}

func TestTemplates_Preview_InvalidID(t *testing.T) {
	store := newMockTemplateStore()
	tmplRenderer := &mockTemplateRenderer{}
	e := setupTemplateTest(store, tmplRenderer)

	body := `{"data": {"title": "test"}}`
	rec := doRequest(e, http.MethodPost, "/v1/templates/not-a-uuid/preview", body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
