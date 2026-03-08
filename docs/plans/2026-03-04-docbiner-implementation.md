# Docbiner Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready HTML-to-PDF/images conversion API service with dashboard, SDKs, and CLI.

**Architecture:** Microservices in Go (API + Worker) communicating via NATS JetStream, with a Next.js dashboard. Chromium headless via chromedp for rendering. PostgreSQL for data, Redis for cache, Minio for temp file storage.

**Tech Stack:** Go 1.22+, Echo, chromedp, NATS JetStream, PostgreSQL 17, Redis 7, Minio, Next.js 15, Tailwind, shadcn/ui, Stripe, Docker Compose.

**Reference:** See `docs/plans/2026-03-04-docbiner-design.md` for full design document.

---

## Phase 1: Project Scaffolding & Infrastructure (Tasks 1-5)

### Task 1: Go Monorepo Setup

**Files:**
- Create: `go.mod`
- Create: `go.work`
- Create: `services/api/main.go`
- Create: `services/api/go.mod`
- Create: `services/worker/main.go`
- Create: `services/worker/go.mod`
- Create: `internal/domain/models.go`
- Create: `internal/domain/go.mod`
- Create: `.gitignore`

**Step 1: Initialize Go workspace**

```bash
cd ~/Development/Projects/docbiner
go work init
```

**Step 2: Create shared domain module**

```bash
mkdir -p internal/domain
cd internal/domain
go mod init github.com/docbiner/docbiner/internal/domain
```

`internal/domain/models.go`:
```go
package domain

import (
	"time"

	"github.com/google/uuid"
)

type Organization struct {
	ID               uuid.UUID `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Slug             string    `json:"slug" db:"slug"`
	PlanID           uuid.UUID `json:"plan_id" db:"plan_id"`
	StripeCustomerID string    `json:"stripe_customer_id,omitempty" db:"stripe_customer_id"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Username     string    `json:"username" db:"username"`
	DisplayName  string    `json:"display_name" db:"display_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty" db:"avatar_url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

type OrgMember struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	OrgID     uuid.UUID  `json:"org_id" db:"org_id"`
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	Role      OrgRole    `json:"role" db:"role"`
	InvitedBy *uuid.UUID `json:"invited_by,omitempty" db:"invited_by"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
}

type APIKeyEnvironment string

const (
	APIKeyEnvLive APIKeyEnvironment = "live"
	APIKeyEnvTest APIKeyEnvironment = "test"
)

type APIKey struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	OrgID       uuid.UUID         `json:"org_id" db:"org_id"`
	CreatedBy   uuid.UUID         `json:"created_by" db:"created_by"`
	KeyHash     string            `json:"-" db:"key_hash"`
	KeyPrefix   string            `json:"key_prefix" db:"key_prefix"`
	Name        string            `json:"name" db:"name"`
	Environment APIKeyEnvironment `json:"environment" db:"environment"`
	LastUsedAt  *time.Time        `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
}

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

type InputType string

const (
	InputTypeURL      InputType = "url"
	InputTypeHTML     InputType = "html"
	InputTypeTemplate InputType = "template"
)

type OutputFormat string

const (
	OutputFormatPDF  OutputFormat = "pdf"
	OutputFormatPNG  OutputFormat = "png"
	OutputFormatJPEG OutputFormat = "jpeg"
	OutputFormatWebP OutputFormat = "webp"
)

type DeliveryMethod string

const (
	DeliverySync    DeliveryMethod = "sync"
	DeliveryWebhook DeliveryMethod = "webhook"
	DeliveryS3      DeliveryMethod = "s3"
)

type Job struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	OrgID          uuid.UUID      `json:"org_id" db:"org_id"`
	APIKeyID       uuid.UUID      `json:"api_key_id" db:"api_key_id"`
	Status         JobStatus      `json:"status" db:"status"`
	InputType      InputType      `json:"input_type" db:"input_type"`
	InputSource    string         `json:"input_source" db:"input_source"`
	InputData      []byte         `json:"input_data,omitempty" db:"input_data"`
	OutputFormat   OutputFormat   `json:"output_format" db:"output_format"`
	Options        []byte         `json:"options" db:"options"`
	DeliveryMethod DeliveryMethod `json:"delivery_method" db:"delivery_method"`
	DeliveryConfig []byte         `json:"delivery_config,omitempty" db:"delivery_config"`
	ResultURL      *string        `json:"result_url,omitempty" db:"result_url"`
	ResultSize     *int64         `json:"result_size,omitempty" db:"result_size"`
	PagesCount     *int           `json:"pages_count,omitempty" db:"pages_count"`
	DurationMs     *int64         `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage   *string        `json:"error_message,omitempty" db:"error_message"`
	IsTest         bool           `json:"is_test" db:"is_test"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
}

type TemplateEngine string

const (
	TemplateEngineHandlebars TemplateEngine = "handlebars"
	TemplateEngineLiquid     TemplateEngine = "liquid"
)

type Template struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	OrgID       uuid.UUID      `json:"org_id" db:"org_id"`
	CreatedBy   uuid.UUID      `json:"created_by" db:"created_by"`
	Name        string         `json:"name" db:"name"`
	Engine      TemplateEngine `json:"engine" db:"engine"`
	HTMLContent string         `json:"html_content" db:"html_content"`
	CSSContent  *string        `json:"css_content,omitempty" db:"css_content"`
	SampleData  []byte         `json:"sample_data,omitempty" db:"sample_data"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

type Plan struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	MonthlyQuota   int       `json:"monthly_quota" db:"monthly_quota"`
	OveragePrice   float64   `json:"overage_price" db:"overage_price"`
	PriceMonthly   float64   `json:"price_monthly" db:"price_monthly"`
	PriceYearly    float64   `json:"price_yearly" db:"price_yearly"`
	MaxFileSize    int       `json:"max_file_size" db:"max_file_size"`
	TimeoutSeconds int       `json:"timeout_seconds" db:"timeout_seconds"`
	Features       []byte    `json:"features" db:"features"`
	Active         bool      `json:"active" db:"active"`
}

type UsageMonthly struct {
	ID              uuid.UUID `json:"id" db:"id"`
	OrgID           uuid.UUID `json:"org_id" db:"org_id"`
	Month           time.Time `json:"month" db:"month"`
	Conversions     int       `json:"conversions" db:"conversions"`
	TestConversions int       `json:"test_conversions" db:"test_conversions"`
	OverageAmount   float64   `json:"overage_amount" db:"overage_amount"`
}
```

**Step 3: Create API service skeleton**

```bash
mkdir -p services/api
cd services/api
go mod init github.com/docbiner/docbiner/services/api
```

`services/api/main.go`:
```go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(e.Start(fmt.Sprintf(":%s", port)))
}
```

**Step 4: Create Worker service skeleton**

```bash
mkdir -p services/worker
cd services/worker
go mod init github.com/docbiner/docbiner/services/worker
```

`services/worker/main.go`:
```go
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	log.Println("Docbiner Worker starting...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Worker shutting down...")
}
```

**Step 5: Setup Go workspace and .gitignore**

```bash
cd ~/Development/Projects/docbiner
go work use ./internal/domain ./services/api ./services/worker
```

`.gitignore`:
```
# Binaries
/bin/
*.exe
*.dll
*.so
*.dylib

# Test
*.test
*.out
coverage.txt

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Env
.env
.env.local
.env.*.local

# Docker
docker-compose.override.yml

# Node (dashboard)
node_modules/
.next/
out/

# Minio data
minio_data/

# Tmp
tmp/
```

**Step 6: Verify everything compiles**

```bash
cd ~/Development/Projects/docbiner
go work sync
cd services/api && go build ./... && cd ../..
cd services/worker && go build ./... && cd ../..
```

Expected: No errors.

**Step 7: Commit**

```bash
git add -A
git commit -m ":tada: feat: scaffold Go monorepo with API and Worker services"
```

---

### Task 2: Docker Compose Infrastructure

**Files:**
- Create: `docker-compose.yml`
- Create: `services/api/Dockerfile`
- Create: `services/worker/Dockerfile`
- Create: `.env.example`

**Step 1: Create .env.example**

`.env.example`:
```bash
# PostgreSQL
POSTGRES_USER=docbiner
POSTGRES_PASSWORD=docbiner_dev
POSTGRES_DB=docbiner

# Redis
REDIS_URL=redis://redis:6379

# NATS
NATS_URL=nats://nats:4222

# Minio
MINIO_ROOT_USER=docbiner
MINIO_ROOT_PASSWORD=docbiner_dev_secret
MINIO_ENDPOINT=minio:9000
MINIO_BUCKET=docbiner-files

# API
API_PORT=8080
DATABASE_URL=postgresql://docbiner:docbiner_dev@postgres:5432/docbiner?sslmode=disable

# Worker
CHROMIUM_PATH=/usr/bin/chromium-browser
```

**Step 2: Create API Dockerfile**

`services/api/Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/api .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/api /usr/local/bin/api
EXPOSE 8080
CMD ["api"]
```

**Step 3: Create Worker Dockerfile**

`services/worker/Dockerfile`:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/worker .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates chromium
ENV CHROMIUM_PATH=/usr/bin/chromium-browser
COPY --from=builder /app/worker /usr/local/bin/worker
CMD ["worker"]
```

**Step 4: Create docker-compose.yml**

`docker-compose.yml`:
```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-docbiner}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-docbiner_dev}
      POSTGRES_DB: ${POSTGRES_DB:-docbiner}
    ports:
      - "5433:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U docbiner"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6380:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  nats:
    image: nats:2-alpine
    command: ["--jetstream", "--store_dir", "/data"]
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - nats_data:/data

  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER:-docbiner}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD:-docbiner_dev_secret}
    ports:
      - "9000:9000"
      - "9001:9001"
    volumes:
      - minio_data:/data

  api:
    build:
      context: ./services/api
    ports:
      - "8080:8080"
    environment:
      PORT: "8080"
      DATABASE_URL: postgresql://${POSTGRES_USER:-docbiner}:${POSTGRES_PASSWORD:-docbiner_dev}@postgres:5432/${POSTGRES_DB:-docbiner}?sslmode=disable
      REDIS_URL: redis://redis:6379
      NATS_URL: nats://nats:4222
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ROOT_USER:-docbiner}
      MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD:-docbiner_dev_secret}
      MINIO_BUCKET: ${MINIO_BUCKET:-docbiner-files}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      nats:
        condition: service_started
    deploy:
      replicas: 1

  worker:
    build:
      context: ./services/worker
    environment:
      DATABASE_URL: postgresql://${POSTGRES_USER:-docbiner}:${POSTGRES_PASSWORD:-docbiner_dev}@postgres:5432/${POSTGRES_DB:-docbiner}?sslmode=disable
      NATS_URL: nats://nats:4222
      MINIO_ENDPOINT: minio:9000
      MINIO_ACCESS_KEY: ${MINIO_ROOT_USER:-docbiner}
      MINIO_SECRET_KEY: ${MINIO_ROOT_PASSWORD:-docbiner_dev_secret}
      MINIO_BUCKET: ${MINIO_BUCKET:-docbiner-files}
      REDIS_URL: redis://redis:6379
    depends_on:
      postgres:
        condition: service_healthy
      nats:
        condition: service_started
    deploy:
      replicas: 2

volumes:
  pg_data:
  nats_data:
  minio_data:
```

**Step 5: Verify compose config**

```bash
docker compose config
```

Expected: Valid YAML output, no errors.

**Step 6: Commit**

```bash
git add -A
git commit -m ":whale: feat: add Docker Compose with PG, Redis, NATS, Minio"
```

---

### Task 3: Database Migrations

**Files:**
- Create: `migrations/001_initial_schema.up.sql`
- Create: `migrations/001_initial_schema.down.sql`
- Create: `Makefile`

**Step 1: Create Makefile with migration commands**

`Makefile`:
```makefile
.PHONY: migrate-up migrate-down migrate-create db-reset

DATABASE_URL ?= postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

db-reset:
	migrate -path migrations -database "$(DATABASE_URL)" drop -f
	migrate -path migrations -database "$(DATABASE_URL)" up
```

**Step 2: Create initial migration (up)**

`migrations/001_initial_schema.up.sql`:
```sql
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Plans
CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    monthly_quota INT NOT NULL DEFAULT 0,
    overage_price DECIMAL(10, 4) NOT NULL DEFAULT 0,
    price_monthly DECIMAL(10, 2) NOT NULL DEFAULT 0,
    price_yearly DECIMAL(10, 2) NOT NULL DEFAULT 0,
    max_file_size INT NOT NULL DEFAULT 5,
    timeout_seconds INT NOT NULL DEFAULT 30,
    features JSONB NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    avatar_url VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organizations
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    plan_id UUID NOT NULL REFERENCES plans(id),
    stripe_customer_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization Members
CREATE TABLE org_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'admin', 'member')),
    invited_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, user_id)
);

-- API Keys
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_prefix VARCHAR(20) NOT NULL,
    name VARCHAR(100) NOT NULL,
    environment VARCHAR(10) NOT NULL DEFAULT 'live' CHECK (environment IN ('live', 'test')),
    permissions JSONB NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_org_id ON api_keys(org_id);

-- Jobs
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    api_key_id UUID NOT NULL REFERENCES api_keys(id),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    input_type VARCHAR(20) NOT NULL CHECK (input_type IN ('url', 'html', 'template')),
    input_source TEXT NOT NULL,
    input_data JSONB,
    output_format VARCHAR(10) NOT NULL CHECK (output_format IN ('pdf', 'png', 'jpeg', 'webp')),
    options JSONB NOT NULL DEFAULT '{}',
    delivery_method VARCHAR(10) NOT NULL DEFAULT 'sync' CHECK (delivery_method IN ('sync', 'webhook', 's3')),
    delivery_config JSONB,
    result_url VARCHAR(500),
    result_size BIGINT,
    pages_count INT,
    duration_ms BIGINT,
    error_message TEXT,
    is_test BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_jobs_org_id ON jobs(org_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- Usage Monthly
CREATE TABLE usage_monthly (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    month DATE NOT NULL,
    conversions INT NOT NULL DEFAULT 0,
    test_conversions INT NOT NULL DEFAULT 0,
    overage_amount DECIMAL(10, 4) NOT NULL DEFAULT 0,
    UNIQUE(org_id, month)
);

CREATE INDEX idx_usage_monthly_org_month ON usage_monthly(org_id, month);

-- Templates
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by UUID NOT NULL REFERENCES users(id),
    name VARCHAR(100) NOT NULL,
    engine VARCHAR(20) NOT NULL DEFAULT 'handlebars' CHECK (engine IN ('handlebars', 'liquid')),
    html_content TEXT NOT NULL,
    css_content TEXT,
    sample_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_templates_org_id ON templates(org_id);

-- Seed free plan
INSERT INTO plans (name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features, active)
VALUES ('free', 50, 0, 0, 0, 2, 30, '{"merge": false, "encryption": false, "templates": true, "watermark_removal": false}', true);

INSERT INTO plans (name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features, active)
VALUES ('starter', 500, 0.04, 9, 86.40, 5, 60, '{"merge": true, "encryption": false, "templates": true, "watermark_removal": true}', true);

INSERT INTO plans (name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features, active)
VALUES ('pro', 5000, 0.025, 39, 374.40, 20, 120, '{"merge": true, "encryption": true, "templates": true, "watermark_removal": true}', true);

INSERT INTO plans (name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features, active)
VALUES ('business', 25000, 0.02, 99, 950.40, 50, 300, '{"merge": true, "encryption": true, "templates": true, "watermark_removal": true}', true);
```

**Step 3: Create initial migration (down)**

`migrations/001_initial_schema.down.sql`:
```sql
DROP TABLE IF EXISTS templates;
DROP TABLE IF EXISTS usage_monthly;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS plans;
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
```

**Step 4: Start postgres and run migration**

```bash
docker compose up -d postgres
sleep 3
make migrate-up
```

Expected: Migrations applied successfully.

**Step 5: Verify tables exist**

```bash
psql "postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable" -c "\dt"
```

Expected: All tables listed (plans, users, organizations, org_members, api_keys, jobs, usage_monthly, templates).

**Step 6: Commit**

```bash
git add -A
git commit -m ":card_file_box: feat: add initial database schema with migrations"
```

---

### Task 4: Database Repository Layer

**Files:**
- Create: `internal/database/go.mod`
- Create: `internal/database/db.go`
- Create: `internal/database/users.go`
- Create: `internal/database/users_test.go`
- Create: `internal/database/organizations.go`
- Create: `internal/database/organizations_test.go`
- Create: `internal/database/apikeys.go`
- Create: `internal/database/apikeys_test.go`
- Create: `internal/database/jobs.go`
- Create: `internal/database/jobs_test.go`
- Create: `internal/database/testhelper_test.go`

**Step 1: Write test helper for database tests**

`internal/database/testhelper_test.go`:
```go
package database_test

import (
	"context"
	"os"
	"testing"

	"github.com/docbiner/docbiner/internal/database"
)

func testDB(t *testing.T) *database.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable"
	}
	db, err := database.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("failed to connect to test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
```

**Step 2: Write failing test for user creation**

`internal/database/users_test.go`:
```go
package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	user, err := db.Users.Create(ctx, "test@example.com", "hashedpw", "testuser", "Test User")
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "Test User", user.DisplayName)
	assert.NotEmpty(t, user.ID)

	// Cleanup
	t.Cleanup(func() {
		db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})
}

func TestGetUserByEmail(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()

	user, _ := db.Users.Create(ctx, "lookup@example.com", "hashedpw", "lookupuser", "Lookup User")
	t.Cleanup(func() {
		db.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
	})

	found, err := db.Users.GetByEmail(ctx, "lookup@example.com")
	require.NoError(t, err)
	assert.Equal(t, user.ID, found.ID)
}
```

**Step 3: Run tests to verify they fail**

```bash
cd internal/database && go test ./... -v
```

Expected: FAIL — packages/types don't exist yet.

**Step 4: Implement database connection and user repository**

`internal/database/db.go`:
```go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool          *pgxpool.Pool
	Users         *UserRepo
	Organizations *OrgRepo
	APIKeys       *APIKeyRepo
	Jobs          *JobRepo
}

func New(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	db := &DB{Pool: pool}
	db.Users = &UserRepo{pool: pool}
	db.Organizations = &OrgRepo{pool: pool}
	db.APIKeys = &APIKeyRepo{pool: pool}
	db.Jobs = &JobRepo{pool: pool}
	return db, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
```

`internal/database/users.go`:
```go
package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash, username, displayName string) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, username, display_name)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, password_hash, username, display_name, avatar_url, created_at, updated_at`,
		email, passwordHash, username, displayName,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Username, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, username, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Username, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &user, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, username, display_name, avatar_url, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Username, &user.DisplayName, &user.AvatarURL, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}
```

**Step 5: Implement org, apikey, jobs repos (same pattern)**

`internal/database/organizations.go`:
```go
package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrgRepo struct {
	pool *pgxpool.Pool
}

func (r *OrgRepo) Create(ctx context.Context, name, slug string, planID uuid.UUID) (*domain.Organization, error) {
	var org domain.Organization
	err := r.pool.QueryRow(ctx,
		`INSERT INTO organizations (name, slug, plan_id)
		 VALUES ($1, $2, $3)
		 RETURNING id, name, slug, plan_id, stripe_customer_id, created_at, updated_at`,
		name, slug, planID,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.PlanID, &org.StripeCustomerID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create org: %w", err)
	}
	return &org, nil
}

func (r *OrgRepo) AddMember(ctx context.Context, orgID, userID uuid.UUID, role domain.OrgRole, invitedBy *uuid.UUID) (*domain.OrgMember, error) {
	var member domain.OrgMember
	err := r.pool.QueryRow(ctx,
		`INSERT INTO org_members (org_id, user_id, role, invited_by)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, org_id, user_id, role, invited_by, created_at`,
		orgID, userID, role, invitedBy,
	).Scan(&member.ID, &member.OrgID, &member.UserID, &member.Role, &member.InvitedBy, &member.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("add org member: %w", err)
	}
	return &member, nil
}

func (r *OrgRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	var org domain.Organization
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, plan_id, stripe_customer_id, created_at, updated_at
		 FROM organizations WHERE slug = $1`,
		slug,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.PlanID, &org.StripeCustomerID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get org by slug: %w", err)
	}
	return &org, nil
}
```

`internal/database/apikeys.go`:
```go
package database

import (
	"context"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyRepo struct {
	pool *pgxpool.Pool
}

func (r *APIKeyRepo) Create(ctx context.Context, orgID, createdBy uuid.UUID, keyHash, keyPrefix, name string, env domain.APIKeyEnvironment) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.pool.QueryRow(ctx,
		`INSERT INTO api_keys (org_id, created_by, key_hash, key_prefix, name, environment)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, org_id, created_by, key_hash, key_prefix, name, environment, last_used_at, expires_at, created_at`,
		orgID, createdBy, keyHash, keyPrefix, name, env,
	).Scan(&key.ID, &key.OrgID, &key.CreatedBy, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Environment, &key.LastUsedAt, &key.ExpiresAt, &key.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &key, nil
}

func (r *APIKeyRepo) GetByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	var key domain.APIKey
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, created_by, key_hash, key_prefix, name, environment, last_used_at, expires_at, created_at
		 FROM api_keys WHERE key_hash = $1`,
		keyHash,
	).Scan(&key.ID, &key.OrgID, &key.CreatedBy, &key.KeyHash, &key.KeyPrefix, &key.Name, &key.Environment, &key.LastUsedAt, &key.ExpiresAt, &key.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get api key by hash: %w", err)
	}
	return &key, nil
}

func (r *APIKeyRepo) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *APIKeyRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]domain.APIKey, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, org_id, created_by, key_hash, key_prefix, name, environment, last_used_at, expires_at, created_at
		 FROM api_keys WHERE org_id = $1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()
	var keys []domain.APIKey
	for rows.Next() {
		var k domain.APIKey
		if err := rows.Scan(&k.ID, &k.OrgID, &k.CreatedBy, &k.KeyHash, &k.KeyPrefix, &k.Name, &k.Environment, &k.LastUsedAt, &k.ExpiresAt, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}
```

`internal/database/jobs.go`:
```go
package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docbiner/docbiner/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepo struct {
	pool *pgxpool.Pool
}

type CreateJobParams struct {
	OrgID          uuid.UUID
	APIKeyID       uuid.UUID
	InputType      domain.InputType
	InputSource    string
	InputData      json.RawMessage
	OutputFormat   domain.OutputFormat
	Options        json.RawMessage
	DeliveryMethod domain.DeliveryMethod
	DeliveryConfig json.RawMessage
	IsTest         bool
}

func (r *JobRepo) Create(ctx context.Context, p CreateJobParams) (*domain.Job, error) {
	var job domain.Job
	err := r.pool.QueryRow(ctx,
		`INSERT INTO jobs (org_id, api_key_id, input_type, input_source, input_data, output_format, options, delivery_method, delivery_config, is_test)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, org_id, api_key_id, status, input_type, input_source, input_data, output_format, options,
		           delivery_method, delivery_config, result_url, result_size, pages_count, duration_ms, error_message, is_test, created_at, completed_at`,
		p.OrgID, p.APIKeyID, p.InputType, p.InputSource, p.InputData, p.OutputFormat, p.Options, p.DeliveryMethod, p.DeliveryConfig, p.IsTest,
	).Scan(&job.ID, &job.OrgID, &job.APIKeyID, &job.Status, &job.InputType, &job.InputSource, &job.InputData, &job.OutputFormat, &job.Options,
		&job.DeliveryMethod, &job.DeliveryConfig, &job.ResultURL, &job.ResultSize, &job.PagesCount, &job.DurationMs, &job.ErrorMessage, &job.IsTest, &job.CreatedAt, &job.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return &job, nil
}

func (r *JobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Job, error) {
	var job domain.Job
	err := r.pool.QueryRow(ctx,
		`SELECT id, org_id, api_key_id, status, input_type, input_source, input_data, output_format, options,
		        delivery_method, delivery_config, result_url, result_size, pages_count, duration_ms, error_message, is_test, created_at, completed_at
		 FROM jobs WHERE id = $1`, id,
	).Scan(&job.ID, &job.OrgID, &job.APIKeyID, &job.Status, &job.InputType, &job.InputSource, &job.InputData, &job.OutputFormat, &job.Options,
		&job.DeliveryMethod, &job.DeliveryConfig, &job.ResultURL, &job.ResultSize, &job.PagesCount, &job.DurationMs, &job.ErrorMessage, &job.IsTest, &job.CreatedAt, &job.CompletedAt)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return &job, nil
}

func (r *JobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = $2 WHERE id = $1`, id, status)
	return err
}

func (r *JobRepo) Complete(ctx context.Context, id uuid.UUID, resultURL string, resultSize int64, pagesCount int, durationMs int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = 'completed', result_url = $2, result_size = $3, pages_count = $4, duration_ms = $5, completed_at = NOW()
		 WHERE id = $1`, id, resultURL, resultSize, pagesCount, durationMs)
	return err
}

func (r *JobRepo) Fail(ctx context.Context, id uuid.UUID, errMsg string, durationMs int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET status = 'failed', error_message = $2, duration_ms = $3, completed_at = NOW()
		 WHERE id = $1`, id, errMsg, durationMs)
	return err
}
```

**Step 6: Run tests**

```bash
cd internal/database && go test ./... -v -count=1
```

Expected: PASS.

**Step 7: Commit**

```bash
git add -A
git commit -m ":card_file_box: feat: add database repository layer with tests"
```

---

### Task 5: Configuration & Shared Packages

**Files:**
- Create: `internal/config/go.mod`
- Create: `internal/config/config.go`
- Create: `internal/apikey/go.mod`
- Create: `internal/apikey/apikey.go`
- Create: `internal/apikey/apikey_test.go`

**Step 1: Write failing test for API key generation**

`internal/apikey/apikey_test.go`:
```go
package apikey_test

import (
	"testing"

	"github.com/docbiner/docbiner/internal/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	key, err := apikey.Generate("live")
	require.NoError(t, err)
	assert.True(t, len(key.Raw) > 20)
	assert.Equal(t, "db_live_", key.Raw[:8])
	assert.NotEmpty(t, key.Hash)
	assert.Equal(t, "db_live_", key.Prefix[:8])
}

func TestGenerateTestKey(t *testing.T) {
	key, err := apikey.Generate("test")
	require.NoError(t, err)
	assert.Equal(t, "db_test_", key.Raw[:8])
}

func TestHashAndVerify(t *testing.T) {
	key, _ := apikey.Generate("live")
	hash := apikey.Hash(key.Raw)
	assert.Equal(t, key.Hash, hash)
}
```

**Step 2: Run test to verify it fails**

```bash
cd internal/apikey && go test ./... -v
```

Expected: FAIL.

**Step 3: Implement API key generation**

`internal/apikey/apikey.go`:
```go
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type GeneratedKey struct {
	Raw    string // The full key (shown to user once): db_live_xxxxx
	Hash   string // SHA-256 hash (stored in DB)
	Prefix string // First 12 chars for identification: db_live_xxxx
}

func Generate(env string) (*GeneratedKey, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	raw := fmt.Sprintf("db_%s_%s", env, hex.EncodeToString(bytes))
	return &GeneratedKey{
		Raw:    raw,
		Hash:   Hash(raw),
		Prefix: raw[:12],
	}, nil
}

func Hash(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
```

**Step 4: Run tests**

```bash
cd internal/apikey && go test ./... -v
```

Expected: PASS.

**Step 5: Create config package**

`internal/config/config.go`:
```go
package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
	NatsURL     string
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	JWTSecret   string
}

func Load() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseURL:    getEnv("DATABASE_URL", "postgresql://docbiner:docbiner_dev@localhost:5433/docbiner?sslmode=disable"),
		RedisURL:       getEnv("REDIS_URL", "redis://localhost:6380"),
		NatsURL:        getEnv("NATS_URL", "nats://localhost:4222"),
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "docbiner"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "docbiner_dev_secret"),
		MinioBucket:    getEnv("MINIO_BUCKET", "docbiner-files"),
		JWTSecret:      getEnv("JWT_SECRET", "dev-secret-change-in-prod"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

**Step 6: Commit**

```bash
git add -A
git commit -m ":wrench: feat: add config and API key generation with tests"
```

---

## Phase 2: Core Conversion Engine (Tasks 6-9)

### Task 6: Chromedp Renderer — PDF

**Files:**
- Create: `internal/renderer/go.mod`
- Create: `internal/renderer/renderer.go`
- Create: `internal/renderer/pdf.go`
- Create: `internal/renderer/pdf_test.go`
- Create: `internal/renderer/options.go`

**Step 1: Write failing test for basic HTML → PDF**

`internal/renderer/pdf_test.go`:
```go
package renderer_test

import (
	"context"
	"testing"

	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderHTMLToPDF(t *testing.T) {
	r, err := renderer.New()
	require.NoError(t, err)
	defer r.Close()

	result, err := r.HTMLToPDF(context.Background(), "<html><body><h1>Hello Docbiner</h1></body></html>", renderer.PDFOptions{
		PageSize: "A4",
	})
	require.NoError(t, err)
	assert.True(t, len(result) > 100, "PDF should have content")
	assert.Equal(t, "%PDF", string(result[:4]), "Should start with PDF magic bytes")
}

func TestRenderURLToPDF(t *testing.T) {
	r, err := renderer.New()
	require.NoError(t, err)
	defer r.Close()

	result, err := r.URLToPDF(context.Background(), "https://example.com", renderer.PDFOptions{
		PageSize: "A4",
	})
	require.NoError(t, err)
	assert.True(t, len(result) > 100)
	assert.Equal(t, "%PDF", string(result[:4]))
}
```

**Step 2: Run test to verify it fails**

```bash
cd internal/renderer && go test ./... -v -timeout 30s
```

Expected: FAIL.

**Step 3: Implement renderer**

`internal/renderer/options.go`:
```go
package renderer

type PDFOptions struct {
	PageSize      string  `json:"page_size"`       // A4, Letter, etc.
	Landscape     bool    `json:"landscape"`
	MarginTop     string  `json:"margin_top"`      // e.g. "20mm"
	MarginBottom  string  `json:"margin_bottom"`
	MarginLeft    string  `json:"margin_left"`
	MarginRight   string  `json:"margin_right"`
	HeaderHTML    string  `json:"header_html"`
	FooterHTML    string  `json:"footer_html"`
	CSS           string  `json:"css"`
	JS            string  `json:"js"`
	WaitFor       string  `json:"wait_for"`        // CSS selector to wait for
	DelayMs       int     `json:"delay_ms"`
	Scale         float64 `json:"scale"`
	PrintBG       bool    `json:"print_background"`
}

type ScreenshotOptions struct {
	Format   string `json:"format"`    // png, jpeg, webp
	Quality  int    `json:"quality"`   // 0-100 for jpeg/webp
	FullPage bool   `json:"full_page"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	CSS      string `json:"css"`
	JS       string `json:"js"`
	WaitFor  string `json:"wait_for"`
	DelayMs  int    `json:"delay_ms"`
}
```

`internal/renderer/renderer.go`:
```go
package renderer

import (
	"context"

	"github.com/chromedp/chromedp"
)

type Renderer struct {
	allocCtx context.Context
	cancel   context.CancelFunc
}

func New() (*Renderer, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	return &Renderer{allocCtx: allocCtx, cancel: cancel}, nil
}

func (r *Renderer) Close() {
	r.cancel()
}
```

`internal/renderer/pdf.go`:
```go
package renderer

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func (r *Renderer) HTMLToPDF(ctx context.Context, html string, opts PDFOptions) ([]byte, error) {
	taskCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	var buf []byte
	actions := []chromedp.Action{
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
	}
	actions = append(actions, r.buildPDFActions(opts, &buf)...)

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("render HTML to PDF: %w", err)
	}
	return buf, nil
}

func (r *Renderer) URLToPDF(ctx context.Context, url string, opts PDFOptions) ([]byte, error) {
	taskCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	var buf []byte
	actions := []chromedp.Action{
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	}

	if opts.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(opts.WaitFor))
	}
	if opts.DelayMs > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(opts.DelayMs)*time.Millisecond))
	}

	actions = append(actions, r.buildPDFActions(opts, &buf)...)

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("render URL to PDF: %w", err)
	}
	return buf, nil
}

func (r *Renderer) buildPDFActions(opts PDFOptions, buf *[]byte) []chromedp.Action {
	var actions []chromedp.Action

	if opts.CSS != "" {
		actions = append(actions, chromedp.Evaluate(fmt.Sprintf(
			`(function(){var s=document.createElement('style');s.textContent=%q;document.head.appendChild(s)})()`, opts.CSS), nil))
	}
	if opts.JS != "" {
		actions = append(actions, chromedp.Evaluate(opts.JS, nil))
	}
	if opts.DelayMs > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(opts.DelayMs)*time.Millisecond))
	}

	printParams := page.PrintToPDF()
	if opts.Landscape {
		printParams = printParams.WithLandscape(true)
	}
	if opts.Scale > 0 {
		printParams = printParams.WithScale(opts.Scale)
	}
	printParams = printParams.WithPrintBackground(opts.PrintBG)

	if opts.MarginTop != "" || opts.MarginBottom != "" || opts.MarginLeft != "" || opts.MarginRight != "" {
		printParams = printParams.
			WithMarginTop(parseMM(opts.MarginTop)).
			WithMarginBottom(parseMM(opts.MarginBottom)).
			WithMarginLeft(parseMM(opts.MarginLeft)).
			WithMarginRight(parseMM(opts.MarginRight))
	}

	if opts.HeaderHTML != "" || opts.FooterHTML != "" {
		printParams = printParams.WithDisplayHeaderFooter(true)
		if opts.HeaderHTML != "" {
			printParams = printParams.WithHeaderTemplate(opts.HeaderHTML)
		}
		if opts.FooterHTML != "" {
			printParams = printParams.WithFooterTemplate(opts.FooterHTML)
		}
	}

	actions = append(actions, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		*buf, _, err = printParams.Do(ctx)
		return err
	}))

	return actions
}

func parseMM(s string) float64 {
	if s == "" {
		return 0
	}
	var val float64
	fmt.Sscanf(s, "%fmm", &val)
	return val / 25.4 // Convert mm to inches (Chrome expects inches)
}
```

**Step 4: Run tests**

```bash
cd internal/renderer && go test ./... -v -timeout 60s
```

Expected: PASS (requires Chromium installed locally or in CI).

**Step 5: Commit**

```bash
git add -A
git commit -m ":sparkles: feat: add Chromium-based PDF renderer with chromedp"
```

---

### Task 7: Chromedp Renderer — Screenshots

**Files:**
- Create: `internal/renderer/screenshot.go`
- Create: `internal/renderer/screenshot_test.go`

**Step 1: Write failing test**

`internal/renderer/screenshot_test.go`:
```go
package renderer_test

import (
	"context"
	"testing"

	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderHTMLToScreenshot(t *testing.T) {
	r, err := renderer.New()
	require.NoError(t, err)
	defer r.Close()

	result, err := r.HTMLToScreenshot(context.Background(), "<html><body><h1>Screenshot Test</h1></body></html>", renderer.ScreenshotOptions{
		Format: "png",
		Width:  1280,
		Height: 720,
	})
	require.NoError(t, err)
	assert.True(t, len(result) > 100)
	// PNG magic bytes
	assert.Equal(t, byte(0x89), result[0])
	assert.Equal(t, byte(0x50), result[1]) // P
}
```

**Step 2: Run test to verify it fails**

```bash
cd internal/renderer && go test -run TestRenderHTMLToScreenshot -v
```

Expected: FAIL.

**Step 3: Implement screenshot rendering**

`internal/renderer/screenshot.go`:
```go
package renderer

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func (r *Renderer) HTMLToScreenshot(ctx context.Context, html string, opts ScreenshotOptions) ([]byte, error) {
	taskCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	width := opts.Width
	if width == 0 {
		width = 1280
	}
	height := opts.Height
	if height == 0 {
		height = 720
	}

	var buf []byte
	actions := []chromedp.Action{
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1, false),
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		chromedp.WaitReady("body"),
	}

	if opts.CSS != "" {
		actions = append(actions, chromedp.Evaluate(fmt.Sprintf(
			`(function(){var s=document.createElement('style');s.textContent=%q;document.head.appendChild(s)})()`, opts.CSS), nil))
	}
	if opts.JS != "" {
		actions = append(actions, chromedp.Evaluate(opts.JS, nil))
	}
	if opts.DelayMs > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(opts.DelayMs)*time.Millisecond))
	}

	if opts.FullPage {
		actions = append(actions, chromedp.FullScreenshot(&buf, screenshotQuality(opts)))
	} else {
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("render HTML to screenshot: %w", err)
	}
	return buf, nil
}

func (r *Renderer) URLToScreenshot(ctx context.Context, url string, opts ScreenshotOptions) ([]byte, error) {
	taskCtx, cancel := chromedp.NewContext(r.allocCtx)
	defer cancel()

	width := opts.Width
	if width == 0 {
		width = 1280
	}
	height := opts.Height
	if height == 0 {
		height = 720
	}

	var buf []byte
	actions := []chromedp.Action{
		emulation.SetDeviceMetricsOverride(int64(width), int64(height), 1, false),
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
	}

	if opts.WaitFor != "" {
		actions = append(actions, chromedp.WaitVisible(opts.WaitFor))
	}
	if opts.CSS != "" {
		actions = append(actions, chromedp.Evaluate(fmt.Sprintf(
			`(function(){var s=document.createElement('style');s.textContent=%q;document.head.appendChild(s)})()`, opts.CSS), nil))
	}
	if opts.JS != "" {
		actions = append(actions, chromedp.Evaluate(opts.JS, nil))
	}
	if opts.DelayMs > 0 {
		actions = append(actions, chromedp.Sleep(time.Duration(opts.DelayMs)*time.Millisecond))
	}

	if opts.FullPage {
		actions = append(actions, chromedp.FullScreenshot(&buf, screenshotQuality(opts)))
	} else {
		actions = append(actions, chromedp.CaptureScreenshot(&buf))
	}

	if err := chromedp.Run(taskCtx, actions...); err != nil {
		return nil, fmt.Errorf("render URL to screenshot: %w", err)
	}
	return buf, nil
}

func screenshotQuality(opts ScreenshotOptions) int {
	if opts.Quality > 0 {
		return opts.Quality
	}
	return 90
}
```

**Step 4: Run tests**

```bash
cd internal/renderer && go test ./... -v -timeout 60s
```

Expected: PASS.

**Step 5: Commit**

```bash
git add -A
git commit -m ":sparkles: feat: add screenshot renderer (PNG/JPEG/WebP)"
```

---

### Task 8: Watermark & PDF Encryption

**Files:**
- Create: `internal/renderer/watermark.go`
- Create: `internal/renderer/watermark_test.go`
- Create: `internal/pdfutil/go.mod`
- Create: `internal/pdfutil/encrypt.go`
- Create: `internal/pdfutil/encrypt_test.go`
- Create: `internal/pdfutil/merge.go`
- Create: `internal/pdfutil/merge_test.go`

Note: Watermark is applied via CSS/JS injection before rendering. PDF encryption and merge use a Go PDF library (pdfcpu or unipdf).

**Step 1: Write failing test for watermark injection**

`internal/renderer/watermark_test.go`:
```go
package renderer_test

import (
	"context"
	"testing"

	"github.com/docbiner/docbiner/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatermarkPDF(t *testing.T) {
	r, err := renderer.New()
	require.NoError(t, err)
	defer r.Close()

	result, err := r.HTMLToPDF(context.Background(), "<html><body><h1>Test</h1></body></html>", renderer.PDFOptions{
		PageSize:      "A4",
		WatermarkText: "DRAFT",
		WatermarkOpacity: 0.1,
	})
	require.NoError(t, err)
	assert.Equal(t, "%PDF", string(result[:4]))
}
```

**Step 2: Add watermark fields to PDFOptions and implement**

Add to `options.go`:
```go
// Add to PDFOptions struct:
WatermarkText    string  `json:"watermark_text"`
WatermarkOpacity float64 `json:"watermark_opacity"`
```

Add watermark CSS injection in `buildPDFActions`:
```go
if opts.WatermarkText != "" {
    opacity := opts.WatermarkOpacity
    if opacity == 0 {
        opacity = 0.1
    }
    watermarkCSS := fmt.Sprintf(`
        body::after {
            content: '%s';
            position: fixed;
            top: 50%%;
            left: 50%%;
            transform: translate(-50%%, -50%%) rotate(-45deg);
            font-size: 80px;
            color: rgba(0,0,0,%.2f);
            pointer-events: none;
            z-index: 9999;
        }`, opts.WatermarkText, opacity)
    actions = append(actions, chromedp.Evaluate(fmt.Sprintf(
        `(function(){var s=document.createElement('style');s.textContent=%q;document.head.appendChild(s)})()`, watermarkCSS), nil))
}
```

**Step 3: Implement PDF encryption (using pdfcpu)**

`internal/pdfutil/encrypt.go`:
```go
package pdfutil

import (
	"bytes"
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type EncryptOptions struct {
	UserPassword  string   // Password to open the PDF
	OwnerPassword string   // Password for full permissions
	Restrict      []string // "print", "copy", "modify"
}

func Encrypt(pdfData []byte, opts EncryptOptions) ([]byte, error) {
	conf := model.NewAESConfiguration(opts.UserPassword, opts.OwnerPassword, 256)

	for _, r := range opts.Restrict {
		switch r {
		case "print":
			conf.Permissions.PrintAllowed = false
		case "copy":
			conf.Permissions.CopyAllowed = false
		case "modify":
			conf.Permissions.ModifyAllowed = false
		}
	}

	reader := bytes.NewReader(pdfData)
	var out bytes.Buffer
	if err := api.Encrypt(reader, &out, conf); err != nil {
		return nil, fmt.Errorf("encrypt PDF: %w", err)
	}
	return out.Bytes(), nil
}
```

**Step 4: Implement PDF merge**

`internal/pdfutil/merge.go`:
```go
package pdfutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func Merge(pdfs [][]byte) ([]byte, error) {
	if len(pdfs) == 0 {
		return nil, fmt.Errorf("no PDFs to merge")
	}
	if len(pdfs) == 1 {
		return pdfs[0], nil
	}

	readers := make([]io.ReadSeeker, len(pdfs))
	for i, p := range pdfs {
		readers[i] = bytes.NewReader(p)
	}

	var out bytes.Buffer
	conf := model.NewDefaultConfiguration()
	if err := api.MergeRaw(readers, &out, false, conf); err != nil {
		return nil, fmt.Errorf("merge PDFs: %w", err)
	}
	return out.Bytes(), nil
}
```

**Step 5: Tests, verify, commit**

```bash
cd internal/pdfutil && go test ./... -v
cd internal/renderer && go test ./... -v -timeout 60s
git add -A
git commit -m ":sparkles: feat: add watermark, PDF encryption, and merge"
```

---

### Task 9: NATS JetStream Job Queue

**Files:**
- Create: `internal/queue/go.mod`
- Create: `internal/queue/nats.go`
- Create: `internal/queue/nats_test.go`

**Step 1: Write failing test**

`internal/queue/nats_test.go`:
```go
package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/docbiner/docbiner/internal/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishAndConsume(t *testing.T) {
	q, err := queue.New("nats://localhost:4222")
	require.NoError(t, err)
	defer q.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	msg := queue.JobMessage{
		JobID: "test-job-123",
		Type:  "convert",
	}
	err = q.Publish(ctx, msg)
	require.NoError(t, err)

	received := make(chan queue.JobMessage, 1)
	go func() {
		q.Subscribe(ctx, func(m queue.JobMessage) error {
			received <- m
			return nil
		})
	}()

	select {
	case got := <-received:
		assert.Equal(t, "test-job-123", got.JobID)
	case <-ctx.Done():
		t.Fatal("timeout waiting for message")
	}
}
```

**Step 2: Implement NATS queue**

`internal/queue/nats.go`:
```go
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StreamName  = "DOCBINER"
	SubjectJobs = "docbiner.jobs"
)

type JobMessage struct {
	JobID string `json:"job_id"`
	Type  string `json:"type"` // "convert", "merge"
}

type Queue struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
}

func New(url string) (*Queue, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("create JetStream: %w", err)
	}
	ctx := context.Background()
	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     StreamName,
		Subjects: []string{"docbiner.>"},
	})
	if err != nil {
		return nil, fmt.Errorf("create stream: %w", err)
	}
	return &Queue{nc: nc, js: js, stream: stream}, nil
}

func (q *Queue) Close() {
	q.nc.Close()
}

func (q *Queue) Publish(ctx context.Context, msg JobMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = q.js.Publish(ctx, SubjectJobs, data)
	return err
}

func (q *Queue) Subscribe(ctx context.Context, handler func(JobMessage) error) error {
	cons, err := q.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "worker",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}

	iter, err := cons.Messages()
	if err != nil {
		return err
	}
	defer iter.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := iter.Next()
			if err != nil {
				continue
			}
			var job JobMessage
			if err := json.Unmarshal(msg.Data(), &job); err != nil {
				msg.Nak()
				continue
			}
			if err := handler(job); err != nil {
				msg.Nak()
				continue
			}
			msg.Ack()
		}
	}
}
```

**Step 3: Run tests (requires NATS running)**

```bash
cd internal/queue && go test ./... -v -timeout 15s
```

Expected: PASS (requires `docker compose up -d nats` first).

**Step 4: Commit**

```bash
git add -A
git commit -m ":sparkles: feat: add NATS JetStream job queue"
```

---

## Phase 3: API Service (Tasks 10-15)

### Task 10: API Authentication Middleware

**Files:**
- Create: `services/api/middleware/auth.go`
- Create: `services/api/middleware/auth_test.go`

Implement Bearer token middleware that:
1. Extracts API key from `Authorization: Bearer db_xxx_...`
2. Hashes it with SHA-256
3. Looks up in DB via `api_keys.key_hash`
4. Sets org_id and api_key on request context
5. Returns 401 if invalid

TDD: Write test with mock DB → implement → verify.

**Commit:** `:lock: feat: add API key authentication middleware`

---

### Task 11: Rate Limiting Middleware

**Files:**
- Create: `services/api/middleware/ratelimit.go`
- Create: `services/api/middleware/ratelimit_test.go`

Implement Redis-based rate limiting:
1. Per API key, sliding window
2. Default: 60 req/min for free, 300 for paid plans
3. Returns 429 with `Retry-After` header

**Commit:** `:lock: feat: add Redis-based rate limiting middleware`

---

### Task 12: Convert Endpoint (Sync)

**Files:**
- Create: `services/api/handlers/convert.go`
- Create: `services/api/handlers/convert_test.go`

Implement `POST /v1/convert`:
1. Parse request body (source, format, options)
2. Validate input (source required, format valid)
3. Determine input type (URL starts with http, else HTML)
4. Create job in DB with status=processing
5. Call renderer directly (sync)
6. Return file bytes with correct Content-Type
7. Update job status to completed

TDD: Write handler test with httptest → implement → verify.

**Commit:** `:sparkles: feat: add sync conversion endpoint POST /v1/convert`

---

### Task 13: Convert Async Endpoint

**Files:**
- Create: `services/api/handlers/convert_async.go`
- Create: `services/api/handlers/convert_async_test.go`

Implement `POST /v1/convert/async`:
1. Parse request body (same as sync + delivery config)
2. Create job in DB with status=pending
3. Publish job to NATS
4. Return job ID + status

**Commit:** `:sparkles: feat: add async conversion endpoint POST /v1/convert/async`

---

### Task 14: Jobs CRUD Endpoints

**Files:**
- Create: `services/api/handlers/jobs.go`
- Create: `services/api/handlers/jobs_test.go`

Implement:
- `GET /v1/jobs` — list jobs for org (paginated, filtered)
- `GET /v1/jobs/:id` — get job status
- `GET /v1/jobs/:id/download` — download result (redirect to Minio signed URL)
- `DELETE /v1/jobs/:id` — delete job + file

**Commit:** `:sparkles: feat: add jobs CRUD endpoints`

---

### Task 15: Templates & Merge Endpoints

**Files:**
- Create: `services/api/handlers/templates.go`
- Create: `services/api/handlers/templates_test.go`
- Create: `services/api/handlers/merge.go`
- Create: `services/api/handlers/merge_test.go`
- Create: `internal/tmpl/go.mod`
- Create: `internal/tmpl/engine.go`

Implement:
- Templates CRUD (`/v1/templates`)
- Template preview (`/v1/templates/:id/preview`)
- Merge endpoint (`/v1/merge`)
- Template engine: Handlebars rendering using `aymerick/raymond`

**Commit:** `:sparkles: feat: add templates CRUD, preview, and merge endpoints`

---

## Phase 4: Worker Service (Tasks 16-18)

### Task 16: Worker Main Loop

**Files:**
- Modify: `services/worker/main.go`
- Create: `services/worker/handler.go`
- Create: `services/worker/handler_test.go`

Implement worker that:
1. Connects to NATS, DB, Minio
2. Subscribes to `docbiner.jobs`
3. On message: load job from DB, process based on type
4. Convert HTML/URL → PDF/image using renderer
5. Apply watermark if test key
6. Apply encryption if requested
7. Upload result to Minio
8. Update job status in DB
9. If webhook delivery: send webhook
10. If S3 delivery: upload to client S3

TDD: Write handler test with mock renderer → implement → verify.

**Commit:** `:sparkles: feat: implement worker job processing loop`

---

### Task 17: Webhook Delivery

**Files:**
- Create: `internal/delivery/webhook.go`
- Create: `internal/delivery/webhook_test.go`

Implement webhook delivery:
1. POST to callback URL with job result metadata
2. Include custom headers from delivery_config
3. Retry 3 times with exponential backoff on failure
4. Sign payload with HMAC-SHA256 if secret provided

**Commit:** `:sparkles: feat: add webhook delivery with retry`

---

### Task 18: S3 Delivery

**Files:**
- Create: `internal/delivery/s3.go`
- Create: `internal/delivery/s3_test.go`
- Create: `internal/storage/minio.go`
- Create: `internal/storage/minio_test.go`

Implement:
1. Minio client for temp storage (upload, signed URL, delete)
2. S3 upload to client's bucket using their credentials
3. Support S3, GCS, R2 (all S3-compatible)

**Commit:** `:sparkles: feat: add S3 delivery and Minio temp storage`

---

## Phase 5: Usage & Billing (Tasks 19-21)

### Task 19: Usage Tracking

**Files:**
- Create: `internal/usage/tracker.go`
- Create: `internal/usage/tracker_test.go`
- Create: `services/api/handlers/usage.go`

Implement:
1. Increment usage counter after each conversion (in worker)
2. Separate live and test conversions
3. Check quota before allowing conversion (in API)
4. `GET /v1/usage` and `GET /v1/usage/history` endpoints

**Commit:** `:sparkles: feat: add usage tracking and quota enforcement`

---

### Task 20: Stripe Integration

**Files:**
- Create: `internal/billing/stripe.go`
- Create: `internal/billing/stripe_test.go`
- Create: `services/api/handlers/billing.go`
- Create: `services/api/handlers/webhooks_stripe.go`

Implement:
1. Create Stripe customer on org creation
2. Checkout session for plan upgrade
3. Customer portal redirect for billing management
4. Stripe webhook handler for payment events
5. Overage billing at end of month

**Commit:** `:sparkles: feat: add Stripe billing integration`

---

### Task 21: User Auth (Register/Login)

**Files:**
- Create: `services/api/handlers/auth.go`
- Create: `services/api/handlers/auth_test.go`
- Create: `internal/auth/jwt.go`

Implement:
1. `POST /v1/auth/register` — create user + org + free plan
2. `POST /v1/auth/login` — email/password → JWT token
3. JWT middleware for dashboard API routes
4. Password hashing with bcrypt

Note: This is separate from API key auth. API keys = programmatic access. JWT = dashboard access.

**Commit:** `:lock: feat: add user registration, login, and JWT auth`

---

## Phase 6: Dashboard (Tasks 22-30)

### Task 22: Next.js Project Setup

**Files:**
- Create: `services/dashboard/` (entire Next.js project)

```bash
cd services
npx create-next-app@latest dashboard --typescript --tailwind --eslint --app --src-dir --import-alias "@/*"
cd dashboard
npx shadcn@latest init
npm install @tanstack/react-query recharts @monaco-editor/react next-auth
```

**Commit:** `:tada: feat: scaffold Next.js dashboard with shadcn/ui`

---

### Task 23: Auth Pages (Login/Register)

**Files:**
- Create: `services/dashboard/src/app/(auth)/login/page.tsx`
- Create: `services/dashboard/src/app/(auth)/register/page.tsx`
- Create: `services/dashboard/src/lib/auth.ts`

Implement login/register pages with NextAuth.js, calling the Go API.

**Commit:** `:sparkles: feat: add login and registration pages`

---

### Task 24: Dashboard Layout & Overview

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/layout.tsx`
- Create: `services/dashboard/src/app/(dashboard)/page.tsx`
- Create: `services/dashboard/src/components/sidebar.tsx`
- Create: `services/dashboard/src/components/usage-chart.tsx`

Implement dashboard shell with sidebar navigation and overview page with usage stats chart (Recharts).

**Commit:** `:sparkles: feat: add dashboard layout and overview page`

---

### Task 25: API Keys Page

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/api-keys/page.tsx`
- Create: `services/dashboard/src/components/api-key-table.tsx`
- Create: `services/dashboard/src/components/create-key-dialog.tsx`

Implement API key management: list, create (live/test), copy, delete, show prefix.

**Commit:** `:sparkles: feat: add API keys management page`

---

### Task 26: Jobs History Page

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/jobs/page.tsx`
- Create: `services/dashboard/src/components/jobs-table.tsx`
- Create: `services/dashboard/src/components/job-detail-dialog.tsx`

Implement job history with pagination, filters (status, format, date), detail modal, download button.

**Commit:** `:sparkles: feat: add jobs history page with filters`

---

### Task 27: Templates Page

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/templates/page.tsx`
- Create: `services/dashboard/src/app/(dashboard)/templates/[id]/page.tsx`
- Create: `services/dashboard/src/components/template-editor.tsx`

Implement template management with Monaco Editor for HTML editing and live preview.

**Commit:** `:sparkles: feat: add templates page with editor`

---

### Task 28: Playground

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/playground/page.tsx`
- Create: `services/dashboard/src/components/playground-editor.tsx`
- Create: `services/dashboard/src/components/pdf-viewer.tsx`

Implement interactive playground:
1. Monaco Editor (left panel) for HTML/CSS
2. PDF preview (right panel) via iframe
3. Options bar: format, size, orientation
4. "Generate" button: calls API with internal test key
5. "Copy cURL" button: generates equivalent cURL command
6. All conversions watermarked and free

**Commit:** `:sparkles: feat: add interactive playground with live PDF preview`

---

### Task 29: Usage & Billing Page

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/billing/page.tsx`
- Create: `services/dashboard/src/components/plan-card.tsx`
- Create: `services/dashboard/src/components/usage-table.tsx`

Implement billing page with current plan display, usage history table, upgrade/downgrade via Stripe Checkout, and "Manage Billing" → Stripe Customer Portal.

**Commit:** `:sparkles: feat: add billing and usage page`

---

### Task 30: Settings Page

**Files:**
- Create: `services/dashboard/src/app/(dashboard)/settings/page.tsx`
- Create: `services/dashboard/src/app/(dashboard)/settings/members/page.tsx`
- Create: `services/dashboard/src/components/member-invite.tsx`

Implement settings: profile edit, org settings, member management (invite, roles, remove).

**Commit:** `:sparkles: feat: add settings and member management pages`

---

## Phase 7: SDKs & CLI (Tasks 31-34)

### Task 31: Node.js/TypeScript SDK

**Files:**
- Create: `sdks/node/` (entire npm package)

Create `@docbiner/sdk` with:
- TypeScript types for all API payloads
- `Docbiner` client class with methods: `convert()`, `convertAsync()`, `jobs.get()`, `jobs.list()`, `templates.*`, `merge()`, `usage()`
- Retry with exponential backoff on 5xx
- Streaming support for large files

```bash
mkdir -p sdks/node && cd sdks/node
npm init -y
npm install typescript @types/node -D
```

**Commit:** `:sparkles: feat: add Node.js/TypeScript SDK (@docbiner/sdk)`

---

### Task 32: Python SDK

**Files:**
- Create: `sdks/python/` (entire pip package)

Create `docbiner` Python package with:
- Type hints for all payloads
- `Docbiner` client class with sync methods
- `AsyncDocbiner` for async/await
- Retry with backoff on 5xx
- Streaming support

```bash
mkdir -p sdks/python && cd sdks/python
python -m venv .venv
```

**Commit:** `:sparkles: feat: add Python SDK (docbiner)`

---

### Task 33: CLI Tool

**Files:**
- Create: `cmd/cli/main.go`
- Create: `cmd/cli/commands/auth.go`
- Create: `cmd/cli/commands/convert.go`
- Create: `cmd/cli/commands/templates.go`
- Create: `cmd/cli/commands/merge.go`
- Create: `cmd/cli/commands/usage.go`

Create Go CLI using `cobra`:
- `docbiner auth login` — store API key in `~/.docbiner/config.json`
- `docbiner convert` — local file or URL conversion
- `docbiner templates` — list/push/preview
- `docbiner merge` — merge multiple files
- `docbiner usage` — check quota

**Commit:** `:sparkles: feat: add Docbiner CLI tool`

---

### Task 34: SDK & CLI Tests

Write integration tests for Node SDK, Python SDK, and CLI against a running API instance.

**Commit:** `:white_check_mark: test: add SDK and CLI integration tests`

---

## Phase 8: Deployment & Ops (Tasks 35-38)

### Task 35: Production Docker Compose

**Files:**
- Create: `docker-compose.prod.yml`
- Create: `Caddyfile`

Production compose with:
- Caddy for auto-TLS
- 2 API replicas
- 4 worker replicas
- Proper resource limits
- Health checks on all services
- Named volumes for persistence

**Commit:** `:whale: feat: add production Docker Compose with Caddy`

---

### Task 36: GitHub Actions CI/CD

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/deploy.yml`

CI pipeline:
1. Run Go tests
2. Run linter (golangci-lint)
3. Build Docker images
4. Push to Docker Hub / GHCR

Deploy pipeline:
1. SSH to VPS
2. Pull latest images
3. `docker compose -f docker-compose.prod.yml up -d`
4. Health check

**Commit:** `:construction_worker: feat: add CI/CD with GitHub Actions`

---

### Task 37: Monitoring Stack

**Files:**
- Create: `monitoring/prometheus.yml`
- Create: `monitoring/grafana/dashboards/docbiner.json`
- Create: `monitoring/docker-compose.monitoring.yml`

Setup Prometheus + Grafana + Loki:
- Metrics: API latency, conversion duration, queue depth, error rate
- Dashboard: overview, per-endpoint, worker health
- Alerts: error rate > 5%, queue depth > 100, disk > 80%

**Commit:** `:chart_with_upwards_trend: feat: add monitoring with Prometheus and Grafana`

---

### Task 38: Landing Page & Documentation

**Files:**
- Create: `services/dashboard/src/app/(marketing)/page.tsx`
- Create: `services/dashboard/src/app/(marketing)/pricing/page.tsx`
- Create: `services/dashboard/src/app/(marketing)/docs/page.tsx`

Create marketing pages:
1. Landing page with hero, features, pricing preview, playground demo
2. Pricing page with plan comparison
3. API documentation (or link to separate docs site)

**Commit:** `:sparkles: feat: add landing page, pricing, and docs`

---

## Execution Order Summary

| Phase | Tasks | Duration est. |
|---|---|---|
| 1. Scaffolding & Infra | 1-5 | Foundation |
| 2. Core Engine | 6-9 | Core value |
| 3. API Service | 10-15 | Public API |
| 4. Worker Service | 16-18 | Async processing |
| 5. Usage & Billing | 19-21 | Monetization |
| 6. Dashboard | 22-30 | User interface |
| 7. SDKs & CLI | 31-34 | Developer tools |
| 8. Deployment & Ops | 35-38 | Production readiness |

**Critical path:** Phase 1 → 2 → 3 → 4 (parallel with 5) → 6 → 7 → 8

**Dependencies:**
- Tasks 10-15 depend on 1-5 (need DB + domain models)
- Tasks 16-18 depend on 6-9 (need renderer + queue)
- Tasks 22-30 depend on 10-15 (need API endpoints)
- Tasks 31-34 depend on 10-15 (need stable API)
- Tasks 35-38 can start after Phase 3
