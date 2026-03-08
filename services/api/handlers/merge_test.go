package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock PDF Merger ---

type mockPDFMerger struct {
	result []byte
	err    error

	lastPDFs [][]byte
}

func (m *mockPDFMerger) Merge(pdfs [][]byte) ([]byte, error) {
	m.lastPDFs = pdfs
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

// --- Mock Renderer for merge tests ---

type mockMergeRenderer struct {
	pdfResults [][]byte
	callIndex  int
	err        error

	lastSources []string
}

func (m *mockMergeRenderer) HTMLToPDF(html string, opts renderer.PDFOptions) ([]byte, error) {
	m.lastSources = append(m.lastSources, html)
	if m.err != nil {
		return nil, m.err
	}
	if m.callIndex < len(m.pdfResults) {
		result := m.pdfResults[m.callIndex]
		m.callIndex++
		return result, nil
	}
	return []byte("%PDF-default"), nil
}

func (m *mockMergeRenderer) URLToPDF(url string, opts renderer.PDFOptions) ([]byte, error) {
	m.lastSources = append(m.lastSources, url)
	if m.err != nil {
		return nil, m.err
	}
	if m.callIndex < len(m.pdfResults) {
		result := m.pdfResults[m.callIndex]
		m.callIndex++
		return result, nil
	}
	return []byte("%PDF-url"), nil
}

func (m *mockMergeRenderer) HTMLToScreenshot(html string, opts renderer.ScreenshotOptions) ([]byte, error) {
	return nil, nil
}

func (m *mockMergeRenderer) URLToScreenshot(url string, opts renderer.ScreenshotOptions) ([]byte, error) {
	return nil, nil
}

// --- Test Helpers ---

func setupMergeTest(r RendererService, m PDFMerger) *echo.Echo {
	e := echo.New()
	h := NewMergeHandler(r, m)

	e.POST("/v1/merge", func(c echo.Context) error {
		c.Set("org_id", uuid.New())
		c.Set("api_key_id", uuid.New())
		c.Set("environment", domain.APIKeyEnvLive)
		return h.Handle(c)
	})

	return e
}

func doMerge(e *echo.Echo, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/merge", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// --- Tests ---

func TestMerge_TwoSources_Success(t *testing.T) {
	mockR := &mockMergeRenderer{
		pdfResults: [][]byte{[]byte("%PDF-page1"), []byte("%PDF-page2")},
	}
	mockM := &mockPDFMerger{result: []byte("%PDF-merged")}
	e := setupMergeTest(mockR, mockM)

	body := `{
		"sources": [
			{"source": "<html>Page 1</html>"},
			{"source": "<html>Page 2</html>"}
		]
	}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
	assert.Equal(t, "%PDF-merged", rec.Body.String())

	// Verify merger was called with 2 PDFs.
	require.Len(t, mockM.lastPDFs, 2)
	assert.Equal(t, []byte("%PDF-page1"), mockM.lastPDFs[0])
	assert.Equal(t, []byte("%PDF-page2"), mockM.lastPDFs[1])
}

func TestMerge_SingleSource_Success(t *testing.T) {
	mockR := &mockMergeRenderer{
		pdfResults: [][]byte{[]byte("%PDF-single")},
	}
	mockM := &mockPDFMerger{result: []byte("%PDF-single")}
	e := setupMergeTest(mockR, mockM)

	body := `{
		"sources": [
			{"source": "<html>Only page</html>"}
		]
	}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/pdf", rec.Header().Get("Content-Type"))
}

func TestMerge_MixedSources_HTMLAndURL(t *testing.T) {
	mockR := &mockMergeRenderer{
		pdfResults: [][]byte{[]byte("%PDF-html"), []byte("%PDF-url")},
	}
	mockM := &mockPDFMerger{result: []byte("%PDF-mixed")}
	e := setupMergeTest(mockR, mockM)

	body := `{
		"sources": [
			{"source": "<html>Page 1</html>"},
			{"source": "https://example.com"}
		]
	}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify the first was HTML and second was URL.
	require.Len(t, mockR.lastSources, 2)
	assert.Equal(t, "<html>Page 1</html>", mockR.lastSources[0])
	assert.Equal(t, "https://example.com", mockR.lastSources[1])
}

func TestMerge_EmptySources_Error(t *testing.T) {
	mockR := &mockMergeRenderer{}
	mockM := &mockPDFMerger{}
	e := setupMergeTest(mockR, mockM)

	body := `{"sources": []}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "validation_error", errResp.Error)
	assert.Contains(t, errResp.Message, "At least one source")
}

func TestMerge_NoSourcesField_Error(t *testing.T) {
	mockR := &mockMergeRenderer{}
	mockM := &mockPDFMerger{}
	e := setupMergeTest(mockR, mockM)

	body := `{}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMerge_EmptySourceValue_Error(t *testing.T) {
	mockR := &mockMergeRenderer{}
	mockM := &mockPDFMerger{}
	e := setupMergeTest(mockR, mockM)

	body := `{
		"sources": [
			{"source": "<html>Page 1</html>"},
			{"source": ""}
		]
	}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Contains(t, errResp.Message, "non-empty")
}

func TestMerge_RendererError(t *testing.T) {
	mockR := &mockMergeRenderer{err: errors.New("chromium crashed")}
	mockM := &mockPDFMerger{}
	e := setupMergeTest(mockR, mockM)

	body := `{
		"sources": [
			{"source": "<html>Page 1</html>"}
		]
	}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "conversion_error", errResp.Error)
}

func TestMerge_MergerError(t *testing.T) {
	mockR := &mockMergeRenderer{
		pdfResults: [][]byte{[]byte("%PDF-1"), []byte("%PDF-2")},
	}
	mockM := &mockPDFMerger{err: errors.New("merge failed")}
	e := setupMergeTest(mockR, mockM)

	body := `{
		"sources": [
			{"source": "<html>Page 1</html>"},
			{"source": "<html>Page 2</html>"}
		]
	}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "merge_error", errResp.Error)
}

func TestMerge_InvalidJSON(t *testing.T) {
	mockR := &mockMergeRenderer{}
	mockM := &mockPDFMerger{}
	e := setupMergeTest(mockR, mockM)

	body := `{not valid}`
	rec := doMerge(e, body)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
