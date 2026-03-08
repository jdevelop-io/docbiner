package usage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// --- Fake pool implementation for unit tests ---
// We build a minimal fake that satisfies the pgxpool method signatures used by Tracker.

// fakeRow implements pgx.Row for QueryRow results.
type fakeRow struct {
	scanFunc func(dest ...any) error
}

func (r *fakeRow) Scan(dest ...any) error {
	return r.scanFunc(dest...)
}

// fakeRows implements pgx.Rows for Query results.
type fakeRows struct {
	data    [][]any
	cursor  int
	closed  bool
	scanErr error
	iterErr error
}

func (r *fakeRows) Close()                                         { r.closed = true }
func (r *fakeRows) Err() error                                     { return r.iterErr }
func (r *fakeRows) CommandTag() pgconn.CommandTag                   { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription    { return nil }
func (r *fakeRows) RawValues() [][]byte                             { return nil }
func (r *fakeRows) Conn() *pgx.Conn                                { return nil }

func (r *fakeRows) Next() bool {
	if r.cursor >= len(r.data) {
		return false
	}
	r.cursor++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.cursor-1]
	for i, d := range dest {
		switch v := d.(type) {
		case *time.Time:
			*v = row[i].(time.Time)
		case *int:
			*v = row[i].(int)
		case *float64:
			*v = row[i].(float64)
		}
	}
	return nil
}

func (r *fakeRows) Values() ([]any, error) { return nil, nil }

// fakePool wraps behavior for Exec, QueryRow, and Query.
// We cannot embed *pgxpool.Pool, so instead Tracker is tested through
// a refactored interface approach. However, since the task requires
// pgxpool.Pool directly, we test with an integration-style approach
// using pgxmock or by testing the SQL logic patterns.
//
// For pure unit tests without a real DB, we test the logic by
// exercising the public API through a helper that sets up a
// test-scoped PostgreSQL connection when available, otherwise
// skips gracefully.

// TestIncrementLiveConversion verifies that Increment with isTest=false
// builds the correct UPSERT query for live conversions.
func TestIncrementLiveConversion(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool)

	err := tracker.Increment(ctx, orgID, false)
	if err != nil {
		t.Fatalf("Increment live: %v", err)
	}

	// Verify the counter.
	usage, err := tracker.GetCurrent(ctx, orgID)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if usage.Conversions != 1 {
		t.Errorf("expected 1 conversion, got %d", usage.Conversions)
	}
	if usage.TestConversions != 0 {
		t.Errorf("expected 0 test conversions, got %d", usage.TestConversions)
	}
}

// TestIncrementTestConversion verifies that Increment with isTest=true
// increments test_conversions only.
func TestIncrementTestConversion(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool)

	err := tracker.Increment(ctx, orgID, true)
	if err != nil {
		t.Fatalf("Increment test: %v", err)
	}

	usage, err := tracker.GetCurrent(ctx, orgID)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if usage.Conversions != 0 {
		t.Errorf("expected 0 conversions, got %d", usage.Conversions)
	}
	if usage.TestConversions != 1 {
		t.Errorf("expected 1 test conversion, got %d", usage.TestConversions)
	}
}

// TestGetCurrentNoData verifies that GetCurrent returns zero values
// when no usage record exists for the current month.
func TestGetCurrentNoData(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool)

	usage, err := tracker.GetCurrent(ctx, orgID)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if usage.Conversions != 0 {
		t.Errorf("expected 0 conversions, got %d", usage.Conversions)
	}
	if usage.TestConversions != 0 {
		t.Errorf("expected 0 test conversions, got %d", usage.TestConversions)
	}
	if usage.OverageAmount != 0 {
		t.Errorf("expected 0 overage, got %f", usage.OverageAmount)
	}
}

// TestCheckQuotaUnderLimit verifies that CheckQuota returns allowed=true
// when conversions are below the plan quota.
func TestCheckQuotaUnderLimit(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool) // uses 'free' plan with quota=100

	// Add a few conversions (under 100).
	for i := 0; i < 5; i++ {
		if err := tracker.Increment(ctx, orgID, false); err != nil {
			t.Fatalf("Increment: %v", err)
		}
	}

	allowed, remaining, err := tracker.CheckQuota(ctx, orgID)
	if err != nil {
		t.Fatalf("CheckQuota: %v", err)
	}
	if !allowed {
		t.Error("expected allowed=true")
	}
	if remaining != 95 {
		t.Errorf("expected remaining=95, got %d", remaining)
	}
}

// TestCheckQuotaOverLimit verifies that CheckQuota returns allowed=false
// when conversions reach or exceed the plan quota.
func TestCheckQuotaOverLimit(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool) // uses 'free' plan with quota=100

	// Fill quota by inserting directly.
	month := currentMonth()
	_, err := pool.Exec(ctx, `
		INSERT INTO usage_monthly (org_id, month, conversions)
		VALUES ($1, $2, 100)
		ON CONFLICT (org_id, month)
		DO UPDATE SET conversions = 100`,
		orgID, month,
	)
	if err != nil {
		t.Fatalf("seed usage: %v", err)
	}

	allowed, remaining, err := tracker.CheckQuota(ctx, orgID)
	if err != nil {
		t.Fatalf("CheckQuota: %v", err)
	}
	if allowed {
		t.Error("expected allowed=false")
	}
	if remaining != 0 {
		t.Errorf("expected remaining=0, got %d", remaining)
	}
}

// TestGetHistory verifies that GetHistory returns entries ordered by month DESC.
func TestGetHistory(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool)

	// Insert usage for 3 months.
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		m := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		_, err := pool.Exec(ctx, `
			INSERT INTO usage_monthly (org_id, month, conversions, test_conversions)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (org_id, month)
			DO UPDATE SET conversions = $3, test_conversions = $4`,
			orgID, m, (i+1)*10, (i+1)*2,
		)
		if err != nil {
			t.Fatalf("seed history month %d: %v", i, err)
		}
	}

	history, err := tracker.GetHistory(ctx, orgID, 12)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(history))
	}

	// Verify descending order.
	for i := 1; i < len(history); i++ {
		if !history[i-1].Month.After(history[i].Month) {
			t.Errorf("history not in DESC order: %v should be after %v",
				history[i-1].Month, history[i].Month)
		}
	}
}

// TestGetQuotaStatus verifies the full QuotaStatus struct.
func TestGetQuotaStatus(t *testing.T) {
	pool := getTestPool(t)
	tracker := New(pool)
	ctx := context.Background()
	orgID := setupTestOrg(t, pool)

	// Add some conversions.
	for i := 0; i < 30; i++ {
		if err := tracker.Increment(ctx, orgID, false); err != nil {
			t.Fatalf("Increment: %v", err)
		}
	}

	status, err := tracker.GetQuotaStatus(ctx, orgID)
	if err != nil {
		t.Fatalf("GetQuotaStatus: %v", err)
	}
	if !status.Allowed {
		t.Error("expected allowed=true")
	}
	if status.Used != 30 {
		t.Errorf("expected used=30, got %d", status.Used)
	}
	if status.Limit != 100 {
		t.Errorf("expected limit=100, got %d", status.Limit)
	}
	if status.Remaining != 70 {
		t.Errorf("expected remaining=70, got %d", status.Remaining)
	}
}

// --- Test helpers ---

// getTestPool returns a pgxpool.Pool connected to the test database.
// It skips the test if DATABASE_URL is not set.
func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := "postgresql://docbiner:docbiner@localhost:5432/docbiner_test"
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Skipf("skipping: cannot connect to test database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skipping: cannot ping test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// setupTestOrg creates a test plan and organization, returning the org ID.
// Uses the 'free' plan (quota=100) seeded in the migration.
func setupTestOrg(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()

	// Get the 'free' plan ID.
	var planID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM plans WHERE name = 'free'`).Scan(&planID)
	if err != nil {
		t.Fatalf("get free plan: %v", err)
	}

	// Create a unique test org.
	orgID := uuid.New()
	slug := fmt.Sprintf("test-org-%s", orgID.String()[:8])
	_, err = pool.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, plan_id)
		VALUES ($1, $2, $3, $4)`,
		orgID, "Test Org", slug, planID,
	)
	if err != nil {
		t.Fatalf("create test org: %v", err)
	}

	t.Cleanup(func() {
		// Clean up: delete usage and org.
		pool.Exec(ctx, `DELETE FROM usage_monthly WHERE org_id = $1`, orgID)
		pool.Exec(ctx, `DELETE FROM organizations WHERE id = $1`, orgID)
	})

	return orgID
}
