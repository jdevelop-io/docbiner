package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestUserRepo_Create(t *testing.T) {
	pool := testDB(t)
	repo := &UserRepo{pool: pool}
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	email := "alice-" + suffix + "@example.com"
	username := "alice" + suffix

	user, err := repo.Create(ctx, email, "hashed_pw_123", username, "Alice Wonderland")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID)
	})

	if user == nil {
		t.Fatal("expected user to be non-nil")
	}
	if user.Email != email {
		t.Errorf("email = %q, want %q", user.Email, email)
	}
	if user.PasswordHash != "hashed_pw_123" {
		t.Errorf("password_hash = %q, want %q", user.PasswordHash, "hashed_pw_123")
	}
	if user.Username != username {
		t.Errorf("username = %q, want %q", user.Username, username)
	}
	if user.DisplayName != "Alice Wonderland" {
		t.Errorf("display_name = %q, want %q", user.DisplayName, "Alice Wonderland")
	}
	if user.ID.String() == "" {
		t.Error("expected user ID to be set")
	}
	if user.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestUserRepo_GetByEmail(t *testing.T) {
	pool := testDB(t)
	repo := &UserRepo{pool: pool}
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	email := "bob-" + suffix + "@example.com"

	// Create a user first.
	created, err := repo.Create(ctx, email, "hashed_pw_456", "bob"+suffix, "Bob Builder")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", created.ID)
	})

	// Retrieve by email.
	found, err := repo.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if found == nil {
		t.Fatal("expected user to be non-nil")
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
	if found.Email != email {
		t.Errorf("email = %q, want %q", found.Email, email)
	}
}

func TestUserRepo_GetByEmail_NotFound(t *testing.T) {
	pool := testDB(t)
	repo := &UserRepo{pool: pool}
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nobody-"+uuid.NewString()[:8]+"@example.com")
	if err == nil {
		t.Fatal("expected error for non-existent email")
	}
}

func TestUserRepo_GetByID(t *testing.T) {
	pool := testDB(t)
	repo := &UserRepo{pool: pool}
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	created, err := repo.Create(ctx, "carol-"+suffix+"@example.com", "hashed_pw_789", "carol"+suffix, "Carol Danvers")
	if err != nil {
		t.Fatalf("Create user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", created.ID)
	})

	found, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found == nil {
		t.Fatal("expected user to be non-nil")
	}
	if found.ID != created.ID {
		t.Errorf("ID = %v, want %v", found.ID, created.ID)
	}
	if found.Username != "carol"+suffix {
		t.Errorf("username = %q, want %q", found.Username, "carol"+suffix)
	}
}
