package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeyRepo handles API key persistence.
type APIKeyRepo struct {
	pool *pgxpool.Pool
}

// Create inserts a new API key and returns it.
func (r *APIKeyRepo) Create(ctx context.Context, orgID, createdBy uuid.UUID, keyHash, keyPrefix, name string, env domain.APIKeyEnvironment) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.pool.QueryRow(ctx,
		`INSERT INTO api_keys (org_id, created_by, key_hash, key_prefix, name, environment)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, org_id, created_by, key_hash, key_prefix, name, environment, last_used_at, expires_at, created_at`,
		orgID, createdBy, keyHash, keyPrefix, name, env,
	).Scan(&k.ID, &k.OrgID, &k.CreatedBy, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Environment, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &k, nil
}

// GetByHash retrieves an API key by its hash.
func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var k domain.APIKey
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, created_by, key_hash, key_prefix, name, environment, last_used_at, expires_at, created_at
		 FROM api_keys WHERE key_hash = $1`,
		keyHash,
	).Scan(&k.ID, &k.OrgID, &k.CreatedBy, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Environment, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return &k, nil
}

// UpdateLastUsed sets the last_used_at timestamp to now.
func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("update api key last used: %w", err)
	}
	return nil
}

// Delete removes an API key by its ID.
func (r *APIKeyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM api_keys WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}
	return nil
}

// ListByOrg returns all API keys for the given organization.
func (r *APIKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, created_by, key_hash, key_prefix, name, environment, last_used_at, expires_at, created_at
		 FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys by org: %w", err)
	}
	defer rows.Close()

	var keys []domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.OrgID, &k.CreatedBy, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Environment, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate api keys: %w", err)
	}
	return keys, nil
}
