package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlanRepo handles plan persistence.
type PlanRepo struct {
	pool *pgxpool.Pool
}

// GetByName retrieves a plan by its name.
func (r *PlanRepo) GetByName(ctx context.Context, name string) (*domain.Plan, error) {
	var p domain.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features, active
		 FROM plans WHERE name = $1`,
		name,
	).Scan(&p.ID, &p.Name, &p.MonthlyQuota, &p.OveragePrice, &p.PriceMonthly, &p.PriceYearly, &p.MaxFileSize, &p.TimeoutSeconds, &p.Features, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get plan by name: %w", err)
	}
	return &p, nil
}

// GetByID retrieves a plan by its ID.
func (r *PlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Plan, error) {
	var p domain.Plan
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features, active
		 FROM plans WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.MonthlyQuota, &p.OveragePrice, &p.PriceMonthly, &p.PriceYearly, &p.MaxFileSize, &p.TimeoutSeconds, &p.Features, &p.Active)
	if err != nil {
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	return &p, nil
}
