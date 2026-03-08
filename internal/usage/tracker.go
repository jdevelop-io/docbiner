package usage

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MonthlyUsage represents usage data for a single month.
type MonthlyUsage struct {
	Month           time.Time `json:"month"`
	Conversions     int       `json:"conversions"`
	TestConversions int       `json:"test_conversions"`
	OverageAmount   float64   `json:"overage_amount"`
}

// QuotaStatus represents the current quota state for an organization.
type QuotaStatus struct {
	Allowed   bool `json:"allowed"`
	Used      int  `json:"used"`
	Limit     int  `json:"limit"`
	Remaining int  `json:"remaining"`
}

// Tracker handles usage tracking and quota enforcement.
type Tracker struct {
	pool *pgxpool.Pool
}

// New creates a new Tracker backed by the given connection pool.
func New(pool *pgxpool.Pool) *Tracker {
	return &Tracker{pool: pool}
}

// currentMonth returns the first day of the current month (UTC) truncated to date.
func currentMonth() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// Increment increments the usage counter for the current month.
// If isTest is true, test_conversions is incremented; otherwise conversions is incremented.
func (t *Tracker) Increment(ctx context.Context, orgID uuid.UUID, isTest bool) error {
	month := currentMonth()

	var query string
	if isTest {
		query = `
			INSERT INTO usage_monthly (org_id, month, test_conversions)
			VALUES ($1, $2, 1)
			ON CONFLICT (org_id, month)
			DO UPDATE SET test_conversions = usage_monthly.test_conversions + 1`
	} else {
		query = `
			INSERT INTO usage_monthly (org_id, month, conversions)
			VALUES ($1, $2, 1)
			ON CONFLICT (org_id, month)
			DO UPDATE SET conversions = usage_monthly.conversions + 1`
	}

	_, err := t.pool.Exec(ctx, query, orgID, month)
	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}

	return nil
}

// GetCurrent returns current month usage for an organization.
// If no usage record exists for the current month, zero values are returned.
func (t *Tracker) GetCurrent(ctx context.Context, orgID uuid.UUID) (*MonthlyUsage, error) {
	month := currentMonth()

	var u MonthlyUsage
	err := t.pool.QueryRow(ctx, `
		SELECT month, conversions, test_conversions, overage_amount
		FROM usage_monthly
		WHERE org_id = $1 AND month = $2`,
		orgID, month,
	).Scan(&u.Month, &u.Conversions, &u.TestConversions, &u.OverageAmount)

	if err == pgx.ErrNoRows {
		return &MonthlyUsage{Month: month}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get current usage: %w", err)
	}

	return &u, nil
}

// GetHistory returns usage history for the last N months, ordered by month DESC.
func (t *Tracker) GetHistory(ctx context.Context, orgID uuid.UUID, months int) ([]*MonthlyUsage, error) {
	rows, err := t.pool.Query(ctx, `
		SELECT month, conversions, test_conversions, overage_amount
		FROM usage_monthly
		WHERE org_id = $1
		ORDER BY month DESC
		LIMIT $2`,
		orgID, months,
	)
	if err != nil {
		return nil, fmt.Errorf("get usage history: %w", err)
	}
	defer rows.Close()

	var history []*MonthlyUsage
	for rows.Next() {
		var u MonthlyUsage
		if err := rows.Scan(&u.Month, &u.Conversions, &u.TestConversions, &u.OverageAmount); err != nil {
			return nil, fmt.Errorf("scan usage row: %w", err)
		}
		history = append(history, &u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage rows: %w", err)
	}

	return history, nil
}

// CheckQuota checks if the organization is within their plan quota.
// It joins with organizations and plans tables to get the monthly_quota,
// then compares with the current month's conversions count.
func (t *Tracker) CheckQuota(ctx context.Context, orgID uuid.UUID) (bool, int, error) {
	month := currentMonth()

	var quota int
	var conversions int

	err := t.pool.QueryRow(ctx, `
		SELECT p.monthly_quota, COALESCE(u.conversions, 0)
		FROM organizations o
		JOIN plans p ON o.plan_id = p.id
		LEFT JOIN usage_monthly u ON u.org_id = o.id AND u.month = $2
		WHERE o.id = $1`,
		orgID, month,
	).Scan(&quota, &conversions)

	if err != nil {
		return false, 0, fmt.Errorf("check quota: %w", err)
	}

	remaining := quota - conversions
	if remaining < 0 {
		remaining = 0
	}
	allowed := conversions < quota

	return allowed, remaining, nil
}

// GetQuotaStatus returns a full QuotaStatus struct for the organization.
func (t *Tracker) GetQuotaStatus(ctx context.Context, orgID uuid.UUID) (*QuotaStatus, error) {
	month := currentMonth()

	var quota int
	var conversions int

	err := t.pool.QueryRow(ctx, `
		SELECT p.monthly_quota, COALESCE(u.conversions, 0)
		FROM organizations o
		JOIN plans p ON o.plan_id = p.id
		LEFT JOIN usage_monthly u ON u.org_id = o.id AND u.month = $2
		WHERE o.id = $1`,
		orgID, month,
	).Scan(&quota, &conversions)

	if err != nil {
		return nil, fmt.Errorf("get quota status: %w", err)
	}

	remaining := quota - conversions
	if remaining < 0 {
		remaining = 0
	}

	return &QuotaStatus{
		Allowed:   conversions < quota,
		Used:      conversions,
		Limit:     quota,
		Remaining: remaining,
	}, nil
}
