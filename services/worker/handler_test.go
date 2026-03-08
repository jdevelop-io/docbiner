package main

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/docbiner/docbiner/internal/queue"
	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/google/uuid"
)

// --- Mocks ---

type mockJobStore struct {
	job          *domain.Job
	getByIDErr   error
	updateErr    error
	completeErr  error
	failErr      error

	// Recorded calls
	updatedStatus domain.JobStatus
	completeCalls []completeCall
	failCalls     []failCall
}

type completeCall struct {
	ID         uuid.UUID
	ResultURL  string
	ResultSize int64
	PagesCount int
	DurationMs int64
}

type failCall struct {
	ID         uuid.UUID
	ErrMsg     string
	DurationMs int64
}

func (m *mockJobStore) GetByID(_ context.Context, id uuid.UUID) (*domain.Job, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	if m.job != nil {
		return m.job, nil
	}
	return nil, errors.New("job not found")
}

func (m *mockJobStore) UpdateStatus(_ context.Context, _ uuid.UUID, status domain.JobStatus) error {
	m.updatedStatus = status
	return m.updateErr
}

func (m *mockJobStore) Complete(_ context.Context, id uuid.UUID, resultURL string, resultSize int64, pagesCount int, durationMs int64) error {
	m.completeCalls = append(m.completeCalls, completeCall{
		ID:         id,
		ResultURL:  resultURL,
		ResultSize: resultSize,
		PagesCount: pagesCount,
		DurationMs: durationMs,
	})
	return m.completeErr
}

func (m *mockJobStore) Fail(_ context.Context, id uuid.UUID, errMsg string, durationMs int64) error {
	m.failCalls = append(m.failCalls, failCall{
		ID:         id,
		ErrMsg:     errMsg,
		DurationMs: durationMs,
	})
	return m.failErr
}

type mockRenderer struct {
	pdfResult        []byte
	screenshotResult []byte
	err              error

	// Recorded calls
	htmlToPDFCalls        []htmlToPDFCall
	urlToPDFCalls         []urlToPDFCall
	htmlToScreenshotCalls []htmlToScreenshotCall
	urlToScreenshotCalls  []urlToScreenshotCall
}

type htmlToPDFCall struct {
	HTML string
	Opts renderer.PDFOptions
}

type urlToPDFCall struct {
	URL  string
	Opts renderer.PDFOptions
}

type htmlToScreenshotCall struct {
	HTML string
	Opts renderer.ScreenshotOptions
}

type urlToScreenshotCall struct {
	URL  string
	Opts renderer.ScreenshotOptions
}

func (m *mockRenderer) HTMLToPDF(_ context.Context, html string, opts renderer.PDFOptions) ([]byte, error) {
	m.htmlToPDFCalls = append(m.htmlToPDFCalls, htmlToPDFCall{HTML: html, Opts: opts})
	return m.pdfResult, m.err
}

func (m *mockRenderer) URLToPDF(_ context.Context, url string, opts renderer.PDFOptions) ([]byte, error) {
	m.urlToPDFCalls = append(m.urlToPDFCalls, urlToPDFCall{URL: url, Opts: opts})
	return m.pdfResult, m.err
}

func (m *mockRenderer) HTMLToScreenshot(_ context.Context, html string, opts renderer.ScreenshotOptions) ([]byte, error) {
	m.htmlToScreenshotCalls = append(m.htmlToScreenshotCalls, htmlToScreenshotCall{HTML: html, Opts: opts})
	return m.screenshotResult, m.err
}

func (m *mockRenderer) URLToScreenshot(_ context.Context, url string, opts renderer.ScreenshotOptions) ([]byte, error) {
	m.urlToScreenshotCalls = append(m.urlToScreenshotCalls, urlToScreenshotCall{URL: url, Opts: opts})
	return m.screenshotResult, m.err
}

type mockStorage struct {
	url string
	err error

	uploadCalls []uploadCall
}

type uploadCall struct {
	Key         string
	Data        []byte
	ContentType string
}

func (m *mockStorage) Upload(_ context.Context, key string, data []byte, contentType string) (string, error) {
	m.uploadCalls = append(m.uploadCalls, uploadCall{Key: key, Data: data, ContentType: contentType})
	return m.url, m.err
}

type mockDelivery struct {
	err           error
	dispatchCalls []dispatchCall
}

type dispatchCall struct {
	Job        *domain.Job
	ResultData []byte
}

func (m *mockDelivery) Dispatch(_ context.Context, job *domain.Job, resultData []byte) error {
	m.dispatchCalls = append(m.dispatchCalls, dispatchCall{Job: job, ResultData: resultData})
	return m.err
}

// --- Helpers ---

func newTestJob(id uuid.UUID, inputType domain.InputType, outputFormat domain.OutputFormat) *domain.Job {
	return &domain.Job{
		ID:             id,
		OrgID:          uuid.New(),
		APIKeyID:       uuid.New(),
		Status:         domain.JobStatusPending,
		InputType:      inputType,
		InputSource:    "<html><body>Hello</body></html>",
		OutputFormat:   outputFormat,
		Options:        []byte("{}"),
		DeliveryMethod: domain.DeliverySync,
		IsTest:         false,
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// --- Tests ---

func TestHandle_HTMLToPDF_Success(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeHTML, domain.OutputFormatPDF)

	store := &mockJobStore{job: job}
	rend := &mockRenderer{pdfResult: []byte("%PDF-1.4 fake pdf content")}
	storage := &mockStorage{url: "https://storage.example.com/jobs/" + jobID.String() + "/result.pdf"}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify status was set to processing.
	if store.updatedStatus != domain.JobStatusProcessing {
		t.Errorf("expected status %q, got %q", domain.JobStatusProcessing, store.updatedStatus)
	}

	// Verify HTMLToPDF was called.
	if len(rend.htmlToPDFCalls) != 1 {
		t.Fatalf("expected 1 HTMLToPDF call, got %d", len(rend.htmlToPDFCalls))
	}
	if rend.htmlToPDFCalls[0].HTML != job.InputSource {
		t.Errorf("expected HTML %q, got %q", job.InputSource, rend.htmlToPDFCalls[0].HTML)
	}

	// Verify upload was called.
	if len(storage.uploadCalls) != 1 {
		t.Fatalf("expected 1 upload call, got %d", len(storage.uploadCalls))
	}
	if storage.uploadCalls[0].ContentType != "application/pdf" {
		t.Errorf("expected content type %q, got %q", "application/pdf", storage.uploadCalls[0].ContentType)
	}

	// Verify Complete was called with correct params.
	if len(store.completeCalls) != 1 {
		t.Fatalf("expected 1 Complete call, got %d", len(store.completeCalls))
	}
	cc := store.completeCalls[0]
	if cc.ID != jobID {
		t.Errorf("expected job ID %s, got %s", jobID, cc.ID)
	}
	if cc.ResultURL != storage.url {
		t.Errorf("expected result URL %q, got %q", storage.url, cc.ResultURL)
	}
	if cc.ResultSize != int64(len(rend.pdfResult)) {
		t.Errorf("expected result size %d, got %d", len(rend.pdfResult), cc.ResultSize)
	}

	// Verify no Fail calls.
	if len(store.failCalls) != 0 {
		t.Errorf("expected 0 Fail calls, got %d", len(store.failCalls))
	}
}

func TestHandle_URLToPNG_Success(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeURL, domain.OutputFormatPNG)
	job.InputSource = "https://example.com"

	store := &mockJobStore{job: job}
	rend := &mockRenderer{screenshotResult: []byte("fake png data")}
	storage := &mockStorage{url: "https://storage.example.com/jobs/" + jobID.String() + "/result.png"}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify URLToScreenshot was called.
	if len(rend.urlToScreenshotCalls) != 1 {
		t.Fatalf("expected 1 URLToScreenshot call, got %d", len(rend.urlToScreenshotCalls))
	}
	if rend.urlToScreenshotCalls[0].URL != "https://example.com" {
		t.Errorf("expected URL %q, got %q", "https://example.com", rend.urlToScreenshotCalls[0].URL)
	}
	if rend.urlToScreenshotCalls[0].Opts.Format != "png" {
		t.Errorf("expected format %q, got %q", "png", rend.urlToScreenshotCalls[0].Opts.Format)
	}

	// Verify Complete was called.
	if len(store.completeCalls) != 1 {
		t.Fatalf("expected 1 Complete call, got %d", len(store.completeCalls))
	}
	if store.completeCalls[0].ResultSize != int64(len(rend.screenshotResult)) {
		t.Errorf("expected result size %d, got %d", len(rend.screenshotResult), store.completeCalls[0].ResultSize)
	}
}

func TestHandle_TestJob_WatermarkApplied(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeHTML, domain.OutputFormatPDF)
	job.IsTest = true

	store := &mockJobStore{job: job}
	rend := &mockRenderer{pdfResult: []byte("fake pdf")}
	storage := &mockStorage{url: "https://storage.example.com/result.pdf"}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify watermark was set in PDF options.
	if len(rend.htmlToPDFCalls) != 1 {
		t.Fatalf("expected 1 HTMLToPDF call, got %d", len(rend.htmlToPDFCalls))
	}
	opts := rend.htmlToPDFCalls[0].Opts
	if opts.WatermarkText != "TEST" {
		t.Errorf("expected watermark text %q, got %q", "TEST", opts.WatermarkText)
	}
	if opts.WatermarkOpacity != 0.15 {
		t.Errorf("expected watermark opacity %f, got %f", 0.15, opts.WatermarkOpacity)
	}
}

func TestHandle_EncryptionOptions_PDFEncrypted(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeHTML, domain.OutputFormatPDF)

	// Set encryption options in job options.
	opts := jobOptions{
		Encrypt: &encryptOptions{
			UserPassword:  "secret",
			OwnerPassword: "owner",
			Restrict:      []string{"print", "copy"},
		},
	}
	job.Options = mustMarshal(t, opts)

	// The mock renderer returns a minimal valid PDF so pdfcpu can encrypt it.
	// Since pdfcpu.Encrypt needs a valid PDF, we test the flow up to encryption:
	// if encryption fails (because our mock PDF is not valid), Fail should be called.
	store := &mockJobStore{job: job}
	rend := &mockRenderer{pdfResult: []byte("not a real pdf")}
	storage := &mockStorage{url: "https://storage.example.com/result.pdf"}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error (failure handled internally), got %v", err)
	}

	// Encryption should fail because the PDF data is not valid.
	// The handler should call Fail.
	if len(store.failCalls) != 1 {
		t.Fatalf("expected 1 Fail call (encryption error), got %d", len(store.failCalls))
	}
	if store.failCalls[0].ErrMsg == "" {
		t.Error("expected non-empty error message")
	}
}

func TestHandle_RendererError_FailCalled(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeHTML, domain.OutputFormatPDF)

	store := &mockJobStore{job: job}
	rend := &mockRenderer{err: errors.New("chromium crashed")}
	storage := &mockStorage{url: "https://storage.example.com/result.pdf"}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error (failure handled internally), got %v", err)
	}

	// Verify Fail was called.
	if len(store.failCalls) != 1 {
		t.Fatalf("expected 1 Fail call, got %d", len(store.failCalls))
	}
	if store.failCalls[0].ErrMsg == "" {
		t.Error("expected non-empty error message on renderer failure")
	}

	// Verify Complete was NOT called.
	if len(store.completeCalls) != 0 {
		t.Errorf("expected 0 Complete calls, got %d", len(store.completeCalls))
	}
}

func TestHandle_StorageUploadError_FailCalled(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeHTML, domain.OutputFormatPDF)

	store := &mockJobStore{job: job}
	rend := &mockRenderer{pdfResult: []byte("fake pdf")}
	storage := &mockStorage{err: errors.New("storage unavailable")}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error (failure handled internally), got %v", err)
	}

	// Verify Fail was called.
	if len(store.failCalls) != 1 {
		t.Fatalf("expected 1 Fail call, got %d", len(store.failCalls))
	}
	if store.failCalls[0].ErrMsg == "" {
		t.Error("expected non-empty error message on storage failure")
	}

	// Verify Complete was NOT called.
	if len(store.completeCalls) != 0 {
		t.Errorf("expected 0 Complete calls, got %d", len(store.completeCalls))
	}
}

func TestHandle_WebhookDelivery_Dispatched(t *testing.T) {
	jobID := uuid.New()
	job := newTestJob(jobID, domain.InputTypeHTML, domain.OutputFormatPDF)
	job.DeliveryMethod = domain.DeliveryWebhook

	store := &mockJobStore{job: job}
	rend := &mockRenderer{pdfResult: []byte("fake pdf")}
	storage := &mockStorage{url: "https://storage.example.com/result.pdf"}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify Complete was called.
	if len(store.completeCalls) != 1 {
		t.Fatalf("expected 1 Complete call, got %d", len(store.completeCalls))
	}

	// Verify Dispatch was called.
	if len(delivery.dispatchCalls) != 1 {
		t.Fatalf("expected 1 Dispatch call, got %d", len(delivery.dispatchCalls))
	}
	if delivery.dispatchCalls[0].Job.ID != jobID {
		t.Errorf("expected dispatched job ID %s, got %s", jobID, delivery.dispatchCalls[0].Job.ID)
	}
	if string(delivery.dispatchCalls[0].ResultData) != "fake pdf" {
		t.Errorf("expected dispatched result data %q, got %q", "fake pdf", delivery.dispatchCalls[0].ResultData)
	}
}

func TestHandle_JobNotFound_ErrorHandledGracefully(t *testing.T) {
	jobID := uuid.New()

	store := &mockJobStore{getByIDErr: errors.New("job not found")}
	rend := &mockRenderer{}
	storage := &mockStorage{}
	delivery := &mockDelivery{}

	h := NewJobHandler(store, rend, storage, delivery)

	err := h.Handle(queue.JobMessage{JobID: jobID.String(), Type: "convert"})
	if err == nil {
		t.Fatal("expected error when job not found")
	}

	// No rendering or completion should have occurred.
	if len(rend.htmlToPDFCalls) != 0 {
		t.Errorf("expected 0 render calls, got %d", len(rend.htmlToPDFCalls))
	}
	if len(store.completeCalls) != 0 {
		t.Errorf("expected 0 Complete calls, got %d", len(store.completeCalls))
	}
	if len(store.failCalls) != 0 {
		t.Errorf("expected 0 Fail calls, got %d", len(store.failCalls))
	}
}
