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
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock Renderer ---

type mockRenderer struct {
	pdfResult        []byte
	screenshotResult []byte
	err              error

	// Capture last call for assertions.
	lastPDFOpts        renderer.PDFOptions
	lastScreenshotOpts renderer.ScreenshotOptions
	lastSource         string
	lastMethod         string
}

func (m *mockRenderer) HTMLToPDF(html string, opts renderer.PDFOptions) ([]byte, error) {
	m.lastSource = html
	m.lastPDFOpts = opts
	m.lastMethod = "HTMLToPDF"
	return m.pdfResult, m.err
}

func (m *mockRenderer) URLToPDF(url string, opts renderer.PDFOptions) ([]byte, error) {
	m.lastSource = url
	m.lastPDFOpts = opts
	m.lastMethod = "URLToPDF"
	return m.pdfResult, m.err
}

func (m *mockRenderer) HTMLToScreenshot(html string, opts renderer.ScreenshotOptions) ([]byte, error) {
	m.lastSource = html
	m.lastScreenshotOpts = opts
	m.lastMethod = "HTMLToScreenshot"
	return m.screenshotResult, m.err
}

func (m *mockRenderer) URLToScreenshot(url string, opts renderer.ScreenshotOptions) ([]byte, error) {
	m.lastSource = url
	m.lastScreenshotOpts = opts
	m.lastMethod = "URLToScreenshot"
	return m.screenshotResult, m.err
}

// --- Mock Job Store ---

type mockJobStore struct {
	job      *domain.Job
	createErr error
	completeErr error
	failErr    error

	lastCreateParams JobCreateParams
	completedID      uuid.UUID
	completedSize    int64
	completedDuration int64
	failedID         uuid.UUID
	failedMsg        string
}

func newMockJobStore() *mockJobStore {
	return &mockJobStore{
		job: &domain.Job{
			ID:        uuid.New(),
			Status:    domain.JobStatusProcessing,
			CreatedAt: time.Now(),
		},
	}
}

func (m *mockJobStore) Create(_ context.Context, params JobCreateParams) (*domain.Job, error) {
	m.lastCreateParams = params
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.job, nil
}

func (m *mockJobStore) Complete(_ context.Context, id uuid.UUID, resultSize int64, durationMs int64) error {
	m.completedID = id
	m.completedSize = resultSize
	m.completedDuration = durationMs
	return m.completeErr
}

func (m *mockJobStore) Fail(_ context.Context, id uuid.UUID, errMsg string, _ int64) error {
	m.failedID = id
	m.failedMsg = errMsg
	return m.failErr
}

// --- Test Helpers ---

func setupConvertTest(r RendererService, j JobStore) (*echo.Echo, *ConvertHandler) {
	e := echo.New()
	h := NewConvertHandler(r, j)

	e.POST("/v1/convert", func(c echo.Context) error {
		// Simulate auth middleware setting context values.
		c.Set("org_id", uuid.New())
		c.Set("api_key_id", uuid.New())
		c.Set("environment", domain.APIKeyEnvLive)
		return h.Handle(c)
	})

	return e, h
}

func setupConvertTestWithEnv(r RendererService, j JobStore, env domain.APIKeyEnvironment) (*echo.Echo, *ConvertHandler) {
	e := echo.New()
	h := NewConvertHandler(r, j)

	e.POST("/v1/convert", func(c echo.Context) error {
		c.Set("org_id", uuid.New())
		c.Set("api_key_id", uuid.New())
		c.Set("environment", env)
		return h.Handle(c)
	})

	return e, h
}

func doConvert(e *echo.Echo, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/convert", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests ---

func TestConvert_ValidPDFFromHTML(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF-1.4 fake")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html><body>Hello</body></html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, "%PDF-1.4 fake", rec.Body.String())
	assert.Equal(t, "HTMLToPDF", mockR.lastMethod)
	assert.Equal(t, "<html><body>Hello</body></html>", mockR.lastSource)

	// Job should be created and completed.
	assert.Equal(t, domain.InputTypeHTML, mockJ.lastCreateParams.InputType)
	assert.Equal(t, domain.OutputFormatPDF, mockJ.lastCreateParams.OutputFormat)
	assert.Equal(t, domain.DeliverySync, mockJ.lastCreateParams.DeliveryMethod)
	assert.Equal(t, mockJ.job.ID, mockJ.completedID)
}

func TestConvert_ValidPDFFromURL(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF-1.4 url")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "https://example.com", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, "URLToPDF", mockR.lastMethod)
	assert.Equal(t, "https://example.com", mockR.lastSource)
	assert.Equal(t, domain.InputTypeURL, mockJ.lastCreateParams.InputType)
}

func TestConvert_DefaultFormatIsPDF(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF-default")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, "HTMLToPDF", mockR.lastMethod)
}

func TestConvert_ValidPNGRequest(t *testing.T) {
	mockR := &mockRenderer{screenshotResult: []byte{0x89, 0x50, 0x4E, 0x47}}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "png"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
	assert.Equal(t, "HTMLToScreenshot", mockR.lastMethod)
	assert.Equal(t, domain.OutputFormatPNG, mockJ.lastCreateParams.OutputFormat)
}

func TestConvert_ValidJPEGRequest(t *testing.T) {
	mockR := &mockRenderer{screenshotResult: []byte{0xFF, 0xD8, 0xFF}}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "jpeg"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	assert.Equal(t, "HTMLToScreenshot", mockR.lastMethod)
}

func TestConvert_ValidWebPRequest(t *testing.T) {
	mockR := &mockRenderer{screenshotResult: []byte("RIFF")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "webp"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/webp", rec.Header().Get("Content-Type"))
	assert.Equal(t, "HTMLToScreenshot", mockR.lastMethod)
}

func TestConvert_ScreenshotFromURL(t *testing.T) {
	mockR := &mockRenderer{screenshotResult: []byte{0x89, 0x50, 0x4E, 0x47}}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "https://example.com", "format": "png"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "URLToScreenshot", mockR.lastMethod)
}

func TestConvert_MissingSource(t *testing.T) {
	mockR := &mockRenderer{}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "source is required")
}

func TestConvert_EmptySource(t *testing.T) {
	mockR := &mockRenderer{}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
}

func TestConvert_WhitespaceOnlySource(t *testing.T) {
	mockR := &mockRenderer{}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "   ", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConvert_InvalidFormat(t *testing.T) {
	mockR := &mockRenderer{}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "bmp"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "Invalid format")
}

func TestConvert_InvalidJSON(t *testing.T) {
	mockR := &mockRenderer{}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{not valid json}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConvert_RendererError(t *testing.T) {
	mockR := &mockRenderer{err: errors.New("chromium crashed")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "conversion_error", errResp.Error)

	// Job should be marked as failed.
	assert.Equal(t, mockJ.job.ID, mockJ.failedID)
	assert.Equal(t, "chromium crashed", mockJ.failedMsg)
}

func TestConvert_JobCreateError(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF-1.4")}
	mockJ := newMockJobStore()
	mockJ.createErr = errors.New("db connection failed")
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "internal_error", errResp.Error)
}

func TestConvert_PDFOptionsPassedThrough(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{
		"source": "<html>test</html>",
		"format": "pdf",
		"options": {
			"page_size": "Letter",
			"landscape": true,
			"margin_top": "10mm",
			"margin_right": "15mm",
			"margin_bottom": "10mm",
			"margin_left": "15mm",
			"header_html": "<div>Header</div>",
			"footer_html": "<div>Footer</div>",
			"css": "body { color: red; }",
			"js": "console.log('hi');",
			"wait_for": "#content",
			"delay_ms": 500,
			"scale": 1.5,
			"print_background": true
		}
	}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Letter", mockR.lastPDFOpts.PageSize)
	assert.True(t, mockR.lastPDFOpts.Landscape)
	assert.Equal(t, "10mm", mockR.lastPDFOpts.MarginTop)
	assert.Equal(t, "15mm", mockR.lastPDFOpts.MarginRight)
	assert.Equal(t, "10mm", mockR.lastPDFOpts.MarginBottom)
	assert.Equal(t, "15mm", mockR.lastPDFOpts.MarginLeft)
	assert.Equal(t, "<div>Header</div>", mockR.lastPDFOpts.HeaderHTML)
	assert.Equal(t, "<div>Footer</div>", mockR.lastPDFOpts.FooterHTML)
	assert.Equal(t, "body { color: red; }", mockR.lastPDFOpts.CSS)
	assert.Equal(t, "console.log('hi');", mockR.lastPDFOpts.JS)
	assert.Equal(t, "#content", mockR.lastPDFOpts.WaitFor)
	assert.Equal(t, 500, mockR.lastPDFOpts.DelayMs)
	assert.Equal(t, 1.5, mockR.lastPDFOpts.Scale)
	assert.True(t, mockR.lastPDFOpts.PrintBG)
}

func TestConvert_ScreenshotOptionsPassedThrough(t *testing.T) {
	mockR := &mockRenderer{screenshotResult: []byte{0x89}}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{
		"source": "<html>test</html>",
		"format": "png",
		"options": {
			"width": 1920,
			"height": 1080,
			"full_page": true,
			"css": "body { background: white; }",
			"js": "document.title = 'test';",
			"wait_for": ".loaded",
			"delay_ms": 200
		}
	}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1920, mockR.lastScreenshotOpts.Width)
	assert.Equal(t, 1080, mockR.lastScreenshotOpts.Height)
	assert.True(t, mockR.lastScreenshotOpts.FullPage)
	assert.Equal(t, "body { background: white; }", mockR.lastScreenshotOpts.CSS)
	assert.Equal(t, "document.title = 'test';", mockR.lastScreenshotOpts.JS)
	assert.Equal(t, ".loaded", mockR.lastScreenshotOpts.WaitFor)
	assert.Equal(t, 200, mockR.lastScreenshotOpts.DelayMs)
	assert.Equal(t, "png", mockR.lastScreenshotOpts.Format)
}

func TestConvert_TestEnvironmentAddsWatermark(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF-test")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTestWithEnv(mockR, mockJ, domain.APIKeyEnvTest)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "TEST", mockR.lastPDFOpts.WatermarkText)
	assert.Equal(t, 0.15, mockR.lastPDFOpts.WatermarkOpacity)
	assert.True(t, mockJ.lastCreateParams.IsTest)
}

func TestConvert_LiveEnvironmentNoWatermark(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF-live")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTestWithEnv(mockR, mockJ, domain.APIKeyEnvLive)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, mockR.lastPDFOpts.WatermarkText)
	assert.False(t, mockJ.lastCreateParams.IsTest)
}

func TestConvert_HTTPSourceDetectedAsURL(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "http://example.com"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "URLToPDF", mockR.lastMethod)
	assert.Equal(t, domain.InputTypeURL, mockJ.lastCreateParams.InputType)
}

func TestConvert_NilOptionsDefaultsPDF(t *testing.T) {
	mockR := &mockRenderer{pdfResult: []byte("%PDF")}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Default print_background should be true when no options.
	assert.True(t, mockR.lastPDFOpts.PrintBG)
}

func TestConvert_CompletedJobMetadata(t *testing.T) {
	content := []byte("test pdf content 12345")
	mockR := &mockRenderer{pdfResult: content}
	mockJ := newMockJobStore()
	e, _ := setupConvertTest(mockR, mockJ)

	body := `{"source": "<html>test</html>", "format": "pdf"}`
	rec := doConvert(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(len(content)), mockJ.completedSize)
	assert.Greater(t, mockJ.completedDuration, int64(-1))
}
