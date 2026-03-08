package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateJobParams holds parameters for creating a new job.
type CreateJobParams struct {
	OrgID          uuid.UUID
	APIKeyID       uuid.UUID
	InputType      domain.InputType
	InputSource    string
	InputData      []byte
	OutputFormat   domain.OutputFormat
	Options        []byte
	DeliveryMethod domain.DeliveryMethod
	DeliveryConfig []byte
	IsTest         bool
}

// JobRepo handles job persistence.
type JobRepo struct {
	pool *pgxpool.Pool
}

// Create inserts a new job and returns it.
func (r *JobRepo) Create(ctx context.Context, params CreateJobParams) (*domain.Job, error) {
	// Default options to empty JSON object to satisfy NOT NULL constraint.
	if params.Options == nil {
		params.Options = []byte("{}")
	}

	var j domain.Job
	err := r.pool.QueryRow(ctx,
		`INSERT INTO jobs (org_id, api_key_id, input_type, input_source, input_data, output_format, options, delivery_method, delivery_config, is_test)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, org_id, api_key_id, status, input_type, input_source, input_data, output_format, options, delivery_method, delivery_config,
		           result_url, result_size, pages_count, duration_ms, error_message, is_test, created_at, completed_at`,
		params.OrgID, params.APIKeyID, params.InputType, params.InputSource, params.InputData,
		params.OutputFormat, params.Options, params.DeliveryMethod, params.DeliveryConfig, params.IsTest,
	).Scan(
		&j.ID, &j.OrgID, &j.APIKeyID, &j.Status, &j.InputType, &j.InputSource, &j.InputData,
		&j.OutputFormat, &j.Options, &j.DeliveryMethod, &j.DeliveryConfig,
		&j.ResultURL, &j.ResultSize, &j.PagesCount, &j.DurationMs, &j.ErrorMessage, &j.IsTest, &j.CreatedAt, &j.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return &j, nil
}

// GetByID retrieves a job by its ID.
func (r *JobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	var j domain.Job
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, api_key_id, status, input_type, input_source, input_data, output_format, options, delivery_method, delivery_config,
		        result_url, result_size, pages_count, duration_ms, error_message, is_test, created_at, completed_at
		 FROM jobs WHERE id = $1`,
		id,
	).Scan(
		&j.ID, &j.OrgID, &j.APIKeyID, &j.Status, &j.InputType, &j.InputSource, &j.InputData,
		&j.OutputFormat, &j.Options, &j.DeliveryMethod, &j.DeliveryConfig,
		&j.ResultURL, &j.ResultSize, &j.PagesCount, &j.DurationMs, &j.ErrorMessage, &j.IsTest, &j.CreatedAt, &j.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get job by id: %w", err)
	}
	return &j, nil
}

// UpdateStatus updates the status of a job.
func (r *JobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $2, completed_at = NOW() WHERE id = $1`,
		id, status,
	)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}

// Complete marks a job as completed with its results.
func (r *JobRepo) Complete(ctx context.Context, id uuid.UUID, resultURL string, resultSize int64, pagesCount int, durationMs int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs
		 SET status = 'completed', result_url = $2, result_size = $3, pages_count = $4, duration_ms = $5, completed_at = NOW()
		 WHERE id = $1`,
		id, resultURL, resultSize, pagesCount, durationMs,
	)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

// Fail marks a job as failed with an error message.
func (r *JobRepo) Fail(ctx context.Context, id uuid.UUID, errMsg string, durationMs int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs
		 SET status = 'failed', error_message = $2, duration_ms = $3, completed_at = NOW()
		 WHERE id = $1`,
		id, errMsg, durationMs,
	)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}
