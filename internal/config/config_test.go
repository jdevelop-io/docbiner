package config_test

import (
	"os"
	"testing"

	"github.com/docbiner/docbiner/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	// Unset all relevant env vars to ensure defaults are used.
	for _, key := range []string{
		"PORT", "DATABASE_URL", "REDIS_URL", "NATS_URL",
		"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY",
		"MINIO_BUCKET", "JWT_SECRET",
	} {
		t.Setenv(key, "")
	}

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	if cfg.DatabaseURL != "postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want default", cfg.DatabaseURL)
	}
	if cfg.RedisURL != "redis://localhost:6380" {
		t.Errorf("RedisURL = %q, want default", cfg.RedisURL)
	}
	if cfg.NatsURL != "nats://localhost:4222" {
		t.Errorf("NatsURL = %q, want default", cfg.NatsURL)
	}
	if cfg.MinioEndpoint != "localhost:9000" {
		t.Errorf("MinioEndpoint = %q, want default", cfg.MinioEndpoint)
	}
	if cfg.MinioBucket != "docbiner-files" {
		t.Errorf("MinioBucket = %q, want default", cfg.MinioBucket)
	}
	if cfg.JWTSecret != "dev-secret-change-in-prod" {
		t.Errorf("JWTSecret = %q, want default", cfg.JWTSecret)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("PORT", "3000")
	t.Cleanup(func() { os.Unsetenv("PORT") })

	cfg := config.Load()

	if cfg.Port != "3000" {
		t.Errorf("Port = %q, want %q", cfg.Port, "3000")
	}
}
