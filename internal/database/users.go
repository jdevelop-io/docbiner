package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo handles user persistence.
type UserRepo struct {
	pool *pgxpool.Pool
}

// Create inserts a new user and returns it.
func (r *UserRepo) Create(ctx context.Context, email, passwordHash, username, displayName string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, username, display_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, password_hash, username, display_name, avatar_url, created_at, updated_at`,
		email, passwordHash, username, displayName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &u, nil
}

// GetByEmail retrieves a user by email address.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, username, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, username, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.DisplayName, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}
