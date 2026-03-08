package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock JobReader ---

type mockJobReader struct {
	job      *domain.Job
	jobs     []*domain.Job
	total    int
	getErr   error
	listErr  error

	lastListOrgID  uuid.UUID
	lastListParams ListParams
}

func (m *mockJobReader) GetByID(_ context.Context, id uuid.UUID) (*domain.Job, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.job != nil && m.job.ID == id {
		return m.job, nil
	}
	return nil, errors.New("not found")
}

func (m *mockJobReader) ListByOrg(_ context.Context, orgID uuid.UUID, params ListParams) ([]*domain.Job, int, error) {
	m.lastListOrgID = orgID
	m.lastListParams = params
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.jobs, m.total, nil
}

// --- Mock JobDeleter ---

type mockJobDeleter struct {
	err       error
	deletedID uuid.UUID
}

func (m *mockJobDeleter) Delete(_ context.Context, id uuid.UUID) error {
	m.deletedID = id
	return m.err
}

// --- Mock FileStore ---

type mockFileStore struct {
	signedURL    string
	signedURLErr error
	deleteErr    error

	lastKey    string
	lastExpiry time.Duration
	deletedKey string
}

func (m *mockFileStore) SignedURL(_ context.Context, key string, expiry time.Duration) (string, error) {
	m.lastKey = key
	m.lastExpiry = expiry
	if m.signedURLErr != nil {
		return "", m.signedURLErr
	}
	return m.signedURL, nil
}

func (m *mockFileStore) Delete(_ context.Context, key string) error {
	m.deletedKey = key
	return m.deleteErr
}

// --- Test Helpers ---

func newTestOrgID() uuid.UUID {
	return uuid.MustParse("11111111-1111-1111-1111-111111111111")
}

func newTestJobID() uuid.UUID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func newTestJob(orgID, jobID uuid.UUID, status domain.JobStatus) *domain.Job {
	return &domain.Job{
		ID:           jobID,
		OrgID:        orgID,
		APIKeyID:     uuid.New(),
		Status:       status,
		InputType:    domain.InputTypeHTML,
		InputSource:  "<html>test</html>",
		OutputFormat: domain.OutputFormatPDF,
		Options:      []byte("{}"),
		DeliveryMethod: domain.DeliverySync,
		CreatedAt:    time.Now(),
	}
}

func setupJobsHandler(reader *mockJobReader, deleter *mockJobDeleter, fs *mockFileStore) *JobsHandler {
	return NewJobsHandler(reader, deleter, fs)
}

func newJobsEcho(h *JobsHandler, orgID uuid.UUID) *echo.Echo {
	e := echo.New()

	v1 := e.Group("/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("org_id", orgID)
			c.Set("api_key_id", uuid.New())
			c.Set("environment", domain.APIKeyEnvLive)
			return next(c)
		}
	})

	v1.GET("/jobs", h.List)
	v1.GET("/jobs/:id", h.GetByID)
	v1.GET("/jobs/:id/download", h.Download)
	v1.DELETE("/jobs/:id", h.Delete)

	return e
}

// --- Tests ---

func TestListJobs_Success(t *testing.T) {
	orgID := newTestOrgID()
	job1 := newTestJob(orgID, uuid.New(), domain.JobStatusCompleted)
	job2 := newTestJob(orgID, uuid.New(), domain.JobStatusPending)

	reader := &mockJobReader{
		jobs:  []*domain.Job{job1, job2},
		total: 150,
	}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?page=1&per_page=20", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp JobListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, 2)
	assert.Equal(t, 1, resp.Pagination.Page)
	assert.Equal(t, 20, resp.Pagination.PerPage)
	assert.Equal(t, 150, resp.Pagination.Total)
	assert.Equal(t, 8, resp.Pagination.TotalPages)

	// Verify params passed to reader.
	assert.Equal(t, orgID, reader.lastListOrgID)
	assert.Equal(t, 1, reader.lastListParams.Page)
	assert.Equal(t, 20, reader.lastListParams.PerPage)
}

func TestListJobs_WithStatusFilter(t *testing.T) {
	orgID := newTestOrgID()
	completedJob := newTestJob(orgID, uuid.New(), domain.JobStatusCompleted)

	reader := &mockJobReader{
		jobs:  []*domain.Job{completedJob},
		total: 1,
	}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?status=completed", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp JobListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Data, 1)
	assert.Equal(t, "completed", reader.lastListParams.Status)
}

func TestListJobs_WithFormatFilter(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{
		jobs:  []*domain.Job{},
		total: 0,
	}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?format=pdf", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "pdf", reader.lastListParams.Format)
}

func TestListJobs_DefaultPagination(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{
		jobs:  []*domain.Job{},
		total: 0,
	}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 1, reader.lastListParams.Page)
	assert.Equal(t, 20, reader.lastListParams.PerPage)
}

func TestListJobs_PerPageCappedAt100(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{
		jobs:  []*domain.Job{},
		total: 0,
	}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs?per_page=500", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 100, reader.lastListParams.PerPage)
}

func TestListJobs_EmptyReturnsEmptyArray(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{
		jobs:  nil,
		total: 0,
	}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp JobListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Data)
	assert.Len(t, resp.Data, 0)
}

func TestGetJob_Success(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	job := newTestJob(orgID, jobID, domain.JobStatusCompleted)

	reader := &mockJobReader{job: job}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp domain.Job
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, jobID, resp.ID)
	assert.Equal(t, orgID, resp.OrgID)
}

func TestGetJob_NotFound(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{getErr: errors.New("not found")}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+unknownID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetJob_DifferentOrg(t *testing.T) {
	orgID := newTestOrgID()
	otherOrgID := uuid.New()
	jobID := newTestJobID()

	// Job belongs to a different org.
	job := newTestJob(otherOrgID, jobID, domain.JobStatusCompleted)

	reader := &mockJobReader{job: job}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID) // current user's org

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "not_found", errResp.Error)
}

func TestGetJob_InvalidID(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDownload_CompletedJob(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	resultURL := "jobs/result-file.pdf"
	job := newTestJob(orgID, jobID, domain.JobStatusCompleted)
	job.ResultURL = &resultURL

	reader := &mockJobReader{job: job}
	fs := &mockFileStore{signedURL: "https://storage.example.com/signed/result-file.pdf?token=abc"}
	h := setupJobsHandler(reader, &mockJobDeleter{}, fs)
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
	assert.Equal(t, "https://storage.example.com/signed/result-file.pdf?token=abc", rec.Header().Get("Location"))
	assert.Equal(t, resultURL, fs.lastKey)
	assert.Equal(t, 15*time.Minute, fs.lastExpiry)
}

func TestDownload_IncompleteJob(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	job := newTestJob(orgID, jobID, domain.JobStatusProcessing)

	reader := &mockJobReader{job: job}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "Job not completed", errResp.Message)
}

func TestDownload_PendingJob(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	job := newTestJob(orgID, jobID, domain.JobStatusPending)

	reader := &mockJobReader{job: job}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDownload_CompletedButNoResult(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	job := newTestJob(orgID, jobID, domain.JobStatusCompleted)
	// ResultURL is nil.

	reader := &mockJobReader{job: job}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+jobID.String()+"/download", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errResp))
	assert.Equal(t, "No result available", errResp.Message)
}

func TestDownload_NotFound(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{getErr: errors.New("not found")}
	h := setupJobsHandler(reader, &mockJobDeleter{}, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodGet, "/v1/jobs/"+uuid.New().String()+"/download", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDeleteJob_Success(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	resultURL := "jobs/result.pdf"
	job := newTestJob(orgID, jobID, domain.JobStatusCompleted)
	job.ResultURL = &resultURL

	reader := &mockJobReader{job: job}
	deleter := &mockJobDeleter{}
	fs := &mockFileStore{}
	h := setupJobsHandler(reader, deleter, fs)
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodDelete, "/v1/jobs/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, jobID, deleter.deletedID)
	assert.Equal(t, resultURL, fs.deletedKey)
}

func TestDeleteJob_SuccessNoFile(t *testing.T) {
	orgID := newTestOrgID()
	jobID := newTestJobID()
	job := newTestJob(orgID, jobID, domain.JobStatusPending)
	// No ResultURL.

	reader := &mockJobReader{job: job}
	deleter := &mockJobDeleter{}
	fs := &mockFileStore{}
	h := setupJobsHandler(reader, deleter, fs)
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodDelete, "/v1/jobs/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, jobID, deleter.deletedID)
	assert.Empty(t, fs.deletedKey) // No file deletion attempted.
}

func TestDeleteJob_NotFound(t *testing.T) {
	orgID := newTestOrgID()

	reader := &mockJobReader{getErr: errors.New("not found")}
	deleter := &mockJobDeleter{}
	h := setupJobsHandler(reader, deleter, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	unknownID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/v1/jobs/"+unknownID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, uuid.Nil, deleter.deletedID) // Delete should not be called.
}

func TestDeleteJob_DifferentOrg(t *testing.T) {
	orgID := newTestOrgID()
	otherOrgID := uuid.New()
	jobID := newTestJobID()
	job := newTestJob(otherOrgID, jobID, domain.JobStatusCompleted)

	reader := &mockJobReader{job: job}
	deleter := &mockJobDeleter{}
	h := setupJobsHandler(reader, deleter, &mockFileStore{})
	e := newJobsEcho(h, orgID)

	req := httptest.NewRequest(http.MethodDelete, "/v1/jobs/"+jobID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, uuid.Nil, deleter.deletedID) // Delete should not be called.
}
