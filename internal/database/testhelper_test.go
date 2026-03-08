package database

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTestDSN = "postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable"

// testDB returns a *pgxpool.Pool connected to the test database.
// It registers a cleanup function that closes the pool when the test ends.
// Tests that create rows should register their own cleanup to delete them.
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

// freePlanID returns the UUID of the 'free' plan from the seed data.
// It fails the test if the plan is not found.
func freePlanID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	var id string
	err := pool.QueryRow(context.Background(), "SELECT id FROM plans WHERE name = 'free'").Scan(&id)
	if err != nil {
		t.Fatalf("get free plan ID: %v", err)
	}
	return id
}
