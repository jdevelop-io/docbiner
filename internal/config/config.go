package config

import "os"

// Config holds the application configuration loaded from environment variables.
type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	NatsURL        string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	JWTSecret      string
}

// Load reads configuration from environment variables, falling back to sane
// development defaults when a variable is not set.
func Load() *Config {
	return &Config{
		Port:           envOrDefault("PORT", "8080"),
		DatabaseURL:    envOrDefault("DATABASE_URL", "postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable"),
		RedisURL:       envOrDefault("REDIS_URL", "redis://localhost:6380"),
		NatsURL:        envOrDefault("NATS_URL", "nats://localhost:4222"),
		MinioEndpoint:  envOrDefault("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: envOrDefault("MINIO_ACCESS_KEY", "docbiner"),
		MinioSecretKey: envOrDefault("MINIO_SECRET_KEY", "docbiner_dev_secret"),
		MinioBucket:    envOrDefault("MINIO_BUCKET", "docbiner-files"),
		JWTSecret:      envOrDefault("JWT_SECRET", "dev-secret-change-in-prod"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
