package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
)

// ListJobsParams holds parameters for listing jobs with filtering and pagination.
type ListJobsParams struct {
	OrgID   uuid.UUID
	Status  string // optional filter
	Format  string // optional filter
	Page    int
	PerPage int
}

// ListByOrg returns paginated jobs for the given organisation with optional filters.
// It returns the matching jobs, total count, and any error.
func (r *JobRepo) ListByOrg(ctx context.Context, params ListJobsParams) ([]*domain.Job, int, error) {
	// Build WHERE clause dynamically.
	where := []string{"org_id = $1"}
	args := []any{params.OrgID}
	argIdx := 2

	if params.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	if params.Format != "" {
		where = append(where, fmt.Sprintf("output_format = $%d", argIdx))
		args = append(args, params.Format)
		argIdx++
	}

	whereClause := strings.Join(where, " AND ")

	// Count total matching rows.
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM jobs WHERE %s", whereClause)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count jobs: %w", err)
	}

	// Fetch the page.
	offset := (params.Page - 1) * params.PerPage
	selectQuery := fmt.Sprintf(
		`SELECT id, org_id, api_key_id, status, input_type, input_source, input_data, output_format, options, delivery_method, delivery_config,
		        result_url, result_size, pages_count, duration_ms, error_message, is_test, created_at, completed_at
		 FROM jobs WHERE %s
		 ORDER BY created_at DESC
		 LIMIT $%d OFFSET $%d`,
		whereClause, argIdx, argIdx+1,
	)
	args = append(args, params.PerPage, offset)

	rows, err := r.pool.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(
			&j.ID, &j.OrgID, &j.APIKeyID, &j.Status, &j.InputType, &j.InputSource, &j.InputData,
			&j.OutputFormat, &j.Options, &j.DeliveryMethod, &j.DeliveryConfig,
			&j.ResultURL, &j.ResultSize, &j.PagesCount, &j.DurationMs, &j.ErrorMessage, &j.IsTest, &j.CreatedAt, &j.CompletedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan job: %w", err)
		}
		jobs = append(jobs, &j)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate jobs: %w", err)
	}

	return jobs, total, nil
}

// Delete removes a job by its ID.
func (r *JobRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM jobs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete job: not found")
	}
	return nil
}
