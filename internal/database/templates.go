package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

// TemplateRepo handles template persistence.
type TemplateRepo struct {
	pool *pgxpool.Pool
}

// Create inserts a new template and returns it.
func (r *TemplateRepo) Create(ctx context.Context, params CreateTemplateParams) (*domain.Template, error) {
	var t domain.Template
	err := r.pool.QueryRow(ctx,
		`INSERT INTO templates (org_id, created_by, name, engine, html_content, css_content, sample_data)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, org_id, created_by, name, engine, html_content, css_content, sample_data, created_at, updated_at`,
		params.OrgID, params.CreatedBy, params.Name, params.Engine, params.HTMLContent, params.CSSContent, params.SampleData,
	).Scan(
		&t.ID, &t.OrgID, &t.CreatedBy, &t.Name, &t.Engine, &t.HTMLContent, &t.CSSContent, &t.SampleData, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return &t, nil
}

// GetByID retrieves a template by its ID.
func (r *TemplateRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Template, error) {
	var t domain.Template
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, created_by, name, engine, html_content, css_content, sample_data, created_at, updated_at
		 FROM templates WHERE id = $1`,
		id,
	).Scan(
		&t.ID, &t.OrgID, &t.CreatedBy, &t.Name, &t.Engine, &t.HTMLContent, &t.CSSContent, &t.SampleData, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get template by id: %w", err)
	}
	return &t, nil
}

// ListByOrg retrieves all templates for the given organization.
func (r *TemplateRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Template, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, created_by, name, engine, html_content, css_content, sample_data, created_at, updated_at
		 FROM templates WHERE org_id = $1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list templates by org: %w", err)
	}
	defer rows.Close()

	var templates []*domain.Template
	for rows.Next() {
		var t domain.Template
		if err := rows.Scan(
			&t.ID, &t.OrgID, &t.CreatedBy, &t.Name, &t.Engine, &t.HTMLContent, &t.CSSContent, &t.SampleData, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, &t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}

	return templates, nil
}

// Update updates a template and returns the updated version.
func (r *TemplateRepo) Update(ctx context.Context, id uuid.UUID, params UpdateTemplateParams) (*domain.Template, error) {
	var t domain.Template

	// Build a dynamic update: use COALESCE to keep existing values when params are nil.
	err := r.pool.QueryRow(ctx,
		`UPDATE templates
		 SET name = COALESCE($2, name),
		     engine = COALESCE($3, engine),
		     html_content = COALESCE($4, html_content),
		     css_content = COALESCE($5, css_content),
		     sample_data = COALESCE($6, sample_data),
		     updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, org_id, created_by, name, engine, html_content, css_content, sample_data, created_at, updated_at`,
		id, params.Name, params.Engine, params.HTMLContent, params.CSSContent, params.SampleData,
	).Scan(
		&t.ID, &t.OrgID, &t.CreatedBy, &t.Name, &t.Engine, &t.HTMLContent, &t.CSSContent, &t.SampleData, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return &t, nil
}

// Delete removes a template by its ID.
func (r *TemplateRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.pool.Exec(ctx,
		`DELETE FROM templates WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("delete template: not found")
	}
	return nil
}
