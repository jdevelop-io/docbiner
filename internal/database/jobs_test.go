package database

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
)

// setupJobTestData creates user, org, org_member, and api_key needed for job tests.
func setupJobTestData(t *testing.T, userRepo *UserRepo, orgRepo *OrgRepo, apiKeyRepo *APIKeyRepo) (*domain.User, *domain.Organization, *domain.APIKey) {
	t.Helper()
	ctx := context.Background()
	suffix := uuid.NewString()[:8]

	user, err := userRepo.Create(ctx, "job-"+suffix+"@example.com", "pw", "jobuser"+suffix, "Job User")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	planID := freePlanID(t, userRepo.pool)
	planUUID, _ := uuid.Parse(planID)
	org, err := orgRepo.Create(ctx, "Job Org "+suffix, "job-org-"+suffix, planUUID)
	if err != nil {
		t.Fatalf("Create org: %v", err)
	}

	// Add user as member (required for FK on api_keys.created_by).
	_, err = orgRepo.AddMember(ctx, org.ID, user.ID, domain.OrgRoleOwner, nil)
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}

	hash := "sha256_job_key_" + suffix
	apiKey, err := apiKeyRepo.Create(ctx, org.ID, user.ID, hash, "db_live_", "Job Key", domain.APIKeyEnvLive)
	if err != nil {
		t.Fatalf("Create API key: %v", err)
	}

	t.Cleanup(func() {
		_, _ = userRepo.pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", org.ID)
		_, _ = userRepo.pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})

	return user, org, apiKey
}

func TestJobRepo_Create(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	jobRepo := &JobRepo{pool: pool}
	ctx := context.Background()

	_, org, apiKey := setupJobTestData(t, userRepo, orgRepo, apiKeyRepo)

	opts, _ := json.Marshal(map[string]any{"landscape": true})
	params := CreateJobParams{
		OrgID:          org.ID,
		APIKeyID:       apiKey.ID,
		InputType:      domain.InputTypeURL,
		InputSource:    "https://example.com",
		OutputFormat:   domain.OutputFormatPDF,
		Options:        opts,
		DeliveryMethod: domain.DeliverySync,
		IsTest:         false,
	}

	job, err := jobRepo.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	if job == nil {
		t.Fatal("expected job to be non-nil")
	}
	if job.OrgID != org.ID {
		t.Errorf("org_id = %v, want %v", job.OrgID, org.ID)
	}
	if job.APIKeyID != apiKey.ID {
		t.Errorf("api_key_id = %v, want %v", job.APIKeyID, apiKey.ID)
	}
	if job.Status != domain.JobStatusPending {
		t.Errorf("status = %q, want %q", job.Status, domain.JobStatusPending)
	}
	if job.InputType != domain.InputTypeURL {
		t.Errorf("input_type = %q, want %q", job.InputType, domain.InputTypeURL)
	}
	if job.InputSource != "https://example.com" {
		t.Errorf("input_source = %q, want %q", job.InputSource, "https://example.com")
	}
	if job.OutputFormat != domain.OutputFormatPDF {
		t.Errorf("output_format = %q, want %q", job.OutputFormat, domain.OutputFormatPDF)
	}
	if job.DeliveryMethod != domain.DeliverySync {
		t.Errorf("delivery_method = %q, want %q", job.DeliveryMethod, domain.DeliverySync)
	}
}

func TestJobRepo_GetByID(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	jobRepo := &JobRepo{pool: pool}
	ctx := context.Background()

	_, org, apiKey := setupJobTestData(t, userRepo, orgRepo, apiKeyRepo)

	created, err := jobRepo.Create(ctx, CreateJobParams{
		OrgID:          org.ID,
		APIKeyID:       apiKey.ID,
		InputType:      domain.InputTypeHTML,
		InputSource:    "<h1>Hello</h1>",
		OutputFormat:   domain.OutputFormatPNG,
		Options:        []byte("{}"),
		DeliveryMethod: domain.DeliverySync,
		IsTest:         true,
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	found, err := jobRepo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found == nil {
		t.Fatal("expected job to be non-nil")
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
	if found.InputSource != "<h1>Hello</h1>" {
		t.Errorf("input_source = %q, want %q", found.InputSource, "<h1>Hello</h1>")
	}
	if found.IsTest != true {
		t.Errorf("is_test = %v, want true", found.IsTest)
	}
}

func TestJobRepo_GetByID_NotFound(t *testing.T) {
	pool := testDB(t)
	jobRepo := &JobRepo{pool: pool}
	ctx := context.Background()

	_, err := jobRepo.GetByID(ctx, uuid.New())
	if err == nil {
		t.Fatal("expected error for non-existent job ID")
	}
}

func TestJobRepo_UpdateStatus(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	jobRepo := &JobRepo{pool: pool}
	ctx := context.Background()

	_, org, apiKey := setupJobTestData(t, userRepo, orgRepo, apiKeyRepo)

	job, err := jobRepo.Create(ctx, CreateJobParams{
		OrgID:          org.ID,
		APIKeyID:       apiKey.ID,
		InputType:      domain.InputTypeURL,
		InputSource:    "https://example.com/status",
		OutputFormat:   domain.OutputFormatPDF,
		Options:        []byte("{}"),
		DeliveryMethod: domain.DeliverySync,
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	err = jobRepo.UpdateStatus(ctx, job.ID, domain.JobStatusProcessing)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	updated, err := jobRepo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if updated.Status != domain.JobStatusProcessing {
		t.Errorf("status = %q, want %q", updated.Status, domain.JobStatusProcessing)
	}
}

func TestJobRepo_Complete(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	jobRepo := &JobRepo{pool: pool}
	ctx := context.Background()

	_, org, apiKey := setupJobTestData(t, userRepo, orgRepo, apiKeyRepo)

	job, err := jobRepo.Create(ctx, CreateJobParams{
		OrgID:          org.ID,
		APIKeyID:       apiKey.ID,
		InputType:      domain.InputTypeURL,
		InputSource:    "https://example.com/complete",
		OutputFormat:   domain.OutputFormatPDF,
		Options:        []byte("{}"),
		DeliveryMethod: domain.DeliverySync,
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	err = jobRepo.Complete(ctx, job.ID, "https://s3.example.com/result.pdf", 102400, 3, 1500)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	completed, err := jobRepo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID after complete: %v", err)
	}
	if completed.Status != domain.JobStatusCompleted {
		t.Errorf("status = %q, want %q", completed.Status, domain.JobStatusCompleted)
	}
	if completed.ResultURL == nil || *completed.ResultURL != "https://s3.example.com/result.pdf" {
		t.Errorf("result_url = %v, want %q", completed.ResultURL, "https://s3.example.com/result.pdf")
	}
	if completed.ResultSize == nil || *completed.ResultSize != 102400 {
		t.Errorf("result_size = %v, want %d", completed.ResultSize, 102400)
	}
	if completed.PagesCount == nil || *completed.PagesCount != 3 {
		t.Errorf("pages_count = %v, want %d", completed.PagesCount, 3)
	}
	if completed.DurationMs == nil || *completed.DurationMs != 1500 {
		t.Errorf("duration_ms = %v, want %d", completed.DurationMs, 1500)
	}
}

func TestJobRepo_Fail(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	jobRepo := &JobRepo{pool: pool}
	ctx := context.Background()

	_, org, apiKey := setupJobTestData(t, userRepo, orgRepo, apiKeyRepo)

	job, err := jobRepo.Create(ctx, CreateJobParams{
		OrgID:          org.ID,
		APIKeyID:       apiKey.ID,
		InputType:      domain.InputTypeURL,
		InputSource:    "https://example.com/fail",
		OutputFormat:   domain.OutputFormatPDF,
		Options:        []byte("{}"),
		DeliveryMethod: domain.DeliverySync,
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	err = jobRepo.Fail(ctx, job.ID, "timeout exceeded", 30000)
	if err != nil {
		t.Fatalf("Fail: %v", err)
	}

	failed, err := jobRepo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID after fail: %v", err)
	}
	if failed.Status != domain.JobStatusFailed {
		t.Errorf("status = %q, want %q", failed.Status, domain.JobStatusFailed)
	}
	if failed.ErrorMessage == nil || *failed.ErrorMessage != "timeout exceeded" {
		t.Errorf("error_message = %v, want %q", failed.ErrorMessage, "timeout exceeded")
	}
	if failed.DurationMs == nil || *failed.DurationMs != 30000 {
		t.Errorf("duration_ms = %v, want %d", failed.DurationMs, 30000)
	}
}
