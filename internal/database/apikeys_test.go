package database

import (
	"context"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
)

func TestAPIKeyRepo_Create(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	ctx := context.Background()

	user, org := setupAPIKeyTestData(t, pool, userRepo, orgRepo)

	key, err := apiKeyRepo.Create(ctx, org.ID, user.ID, "sha256_hash_abc", "db_live_", "Production Key", domain.APIKeyEnvLive)
	if err != nil {
		t.Fatalf("Create API key: %v", err)
	}

	if key == nil {
		t.Fatal("expected API key to be non-nil")
	}
	if key.OrgID != org.ID {
		t.Errorf("org_id = %v, want %v", key.OrgID, org.ID)
	}
	if key.CreatedBy != user.ID {
		t.Errorf("created_by = %v, want %v", key.CreatedBy, user.ID)
	}
	if key.KeyHash != "sha256_hash_abc" {
		t.Errorf("key_hash = %q, want %q", key.KeyHash, "sha256_hash_abc")
	}
	if key.KeyPrefix != "db_live_" {
		t.Errorf("key_prefix = %q, want %q", key.KeyPrefix, "db_live_")
	}
	if key.Name != "Production Key" {
		t.Errorf("name = %q, want %q", key.Name, "Production Key")
	}
	if key.Environment != domain.APIKeyEnvLive {
		t.Errorf("environment = %q, want %q", key.Environment, domain.APIKeyEnvLive)
	}
	if key.LastUsedAt != nil {
		t.Errorf("last_used_at should be nil, got %v", key.LastUsedAt)
	}
}

func TestAPIKeyRepo_GetByHash(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	ctx := context.Background()

	user, org := setupAPIKeyTestData(t, pool, userRepo, orgRepo)

	hash := "sha256_hash_getbyhash_" + uuid.NewString()[:8]
	created, err := apiKeyRepo.Create(ctx, org.ID, user.ID, hash, "db_test_", "Test Key", domain.APIKeyEnvTest)
	if err != nil {
		t.Fatalf("Create API key: %v", err)
	}

	found, err := apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByHash: %v", err)
	}
	if found == nil {
		t.Fatal("expected API key to be non-nil")
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
}

func TestAPIKeyRepo_GetByHash_NotFound(t *testing.T) {
	pool := testDB(t)
	apiKeyRepo := &APIKeyRepo{pool: pool}
	ctx := context.Background()

	_, err := apiKeyRepo.GetByHash(ctx, "nonexistent_hash")
	if err == nil {
		t.Fatal("expected error for non-existent hash")
	}
}

func TestAPIKeyRepo_UpdateLastUsed(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	ctx := context.Background()

	user, org := setupAPIKeyTestData(t, pool, userRepo, orgRepo)

	hash := "sha256_hash_lastused_" + uuid.NewString()[:8]
	key, err := apiKeyRepo.Create(ctx, org.ID, user.ID, hash, "db_live_", "Update Test", domain.APIKeyEnvLive)
	if err != nil {
		t.Fatalf("Create API key: %v", err)
	}

	before := time.Now().Add(-time.Second)
	err = apiKeyRepo.UpdateLastUsed(ctx, key.ID)
	if err != nil {
		t.Fatalf("UpdateLastUsed: %v", err)
	}

	// Re-fetch and verify.
	updated, err := apiKeyRepo.GetByHash(ctx, hash)
	if err != nil {
		t.Fatalf("GetByHash after update: %v", err)
	}
	if updated.LastUsedAt == nil {
		t.Fatal("expected last_used_at to be set after update")
	}
	if updated.LastUsedAt.Before(before) {
		t.Errorf("last_used_at = %v, expected after %v", updated.LastUsedAt, before)
	}
}

func TestAPIKeyRepo_ListByOrg(t *testing.T) {
	pool := testDB(t)
	userRepo := &UserRepo{pool: pool}
	orgRepo := &OrgRepo{pool: pool}
	apiKeyRepo := &APIKeyRepo{pool: pool}
	ctx := context.Background()

	user, org := setupAPIKeyTestData(t, pool, userRepo, orgRepo)

	// Create two keys.
	hash1 := "sha256_hash_list1_" + uuid.NewString()[:8]
	hash2 := "sha256_hash_list2_" + uuid.NewString()[:8]
	_, err := apiKeyRepo.Create(ctx, org.ID, user.ID, hash1, "db_live_", "Key 1", domain.APIKeyEnvLive)
	if err != nil {
		t.Fatalf("Create key 1: %v", err)
	}
	_, err = apiKeyRepo.Create(ctx, org.ID, user.ID, hash2, "db_test_", "Key 2", domain.APIKeyEnvTest)
	if err != nil {
		t.Fatalf("Create key 2: %v", err)
	}

	keys, err := apiKeyRepo.ListByOrg(ctx, org.ID)
	if err != nil {
		t.Fatalf("ListByOrg: %v", err)
	}
	if len(keys) < 2 {
		t.Errorf("expected at least 2 keys, got %d", len(keys))
	}

	// Verify all returned keys belong to this org.
	for _, k := range keys {
		if k.OrgID != org.ID {
			t.Errorf("key %v has org_id %v, want %v", k.ID, k.OrgID, org.ID)
		}
	}
}

// setupAPIKeyTestData creates the user and org needed for API key tests.
func setupAPIKeyTestData(t *testing.T, pool interface{}, userRepo *UserRepo, orgRepo *OrgRepo) (*domain.User, *domain.Organization) {
	t.Helper()
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	user, err := userRepo.Create(ctx, "ak-"+suffix+"@example.com", "pw", "akuser"+suffix, "AK User")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	planID := freePlanID(t, userRepo.pool)
	planUUID, _ := uuid.Parse(planID)
	org, err := orgRepo.Create(ctx, "AK Org "+suffix, "ak-org-"+suffix, planUUID)
	if err != nil {
		t.Fatalf("Create org: %v", err)
	}

	t.Cleanup(func() {
		_, _ = userRepo.pool.Exec(context.Background(), "DELETE FROM organizations WHERE id = $1", org.ID)
		_, _ = userRepo.pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})

	return user, org
}
