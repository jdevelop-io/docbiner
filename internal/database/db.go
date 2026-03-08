package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB holds the connection pool and all repository instances.
type DB struct {
	Pool          *pgxpool.Pool
	Users         *UserRepo
	Organizations *OrgRepo
	Plans         *PlanRepo
	APIKeys       *APIKeyRepo
	Jobs          *JobRepo
	Templates     *TemplateRepo
}

// New creates a new DB instance with the given DSN, connects to PostgreSQL
// and initialises all repositories.
func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	db := &DB{Pool: pool}
	db.Users = &UserRepo{pool: pool}
	db.Organizations = &OrgRepo{pool: pool}
	db.Plans = &PlanRepo{pool: pool}
	db.APIKeys = &APIKeyRepo{pool: pool}
	db.Jobs = &JobRepo{pool: pool}
	db.Templates = &TemplateRepo{pool: pool}

	return db, nil
}

// Close releases the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}
