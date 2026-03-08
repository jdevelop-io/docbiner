-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Plans
CREATE TABLE plans (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name           VARCHAR(50) NOT NULL UNIQUE,
    monthly_quota  INT NOT NULL,
    overage_price  DECIMAL(10,4) NOT NULL DEFAULT 0,
    price_monthly  DECIMAL(10,2) NOT NULL DEFAULT 0,
    price_yearly   DECIMAL(10,2) NOT NULL DEFAULT 0,
    max_file_size  INT NOT NULL DEFAULT 10485760,
    timeout_seconds INT NOT NULL DEFAULT 30,
    features       JSONB NOT NULL DEFAULT '{}',
    active         BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Users
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email          VARCHAR(255) NOT NULL UNIQUE,
    password_hash  VARCHAR(255) NOT NULL,
    username       VARCHAR(50) NOT NULL UNIQUE,
    display_name   VARCHAR(100) NOT NULL,
    avatar_url     VARCHAR(500) NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organizations
CREATE TABLE organizations (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                VARCHAR(100) NOT NULL,
    slug                VARCHAR(100) NOT NULL UNIQUE,
    plan_id             UUID NOT NULL REFERENCES plans(id),
    stripe_customer_id  VARCHAR(255) NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Organization Members
CREATE TABLE org_members (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        VARCHAR(20) NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
    invited_by  UUID NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, user_id)
);

-- API Keys
CREATE TABLE api_keys (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by   UUID NOT NULL REFERENCES users(id),
    key_hash     VARCHAR(64) NOT NULL UNIQUE,
    key_prefix   VARCHAR(20) NOT NULL,
    name         VARCHAR(100) NOT NULL,
    environment  VARCHAR(10) NOT NULL CHECK (environment IN ('live', 'test')),
    permissions  JSONB NOT NULL DEFAULT '{}',
    last_used_at TIMESTAMPTZ NULL,
    expires_at   TIMESTAMPTZ NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_org_id ON api_keys(org_id);

-- Jobs
CREATE TABLE jobs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    api_key_id      UUID NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')) DEFAULT 'pending',
    input_type      VARCHAR(20) NOT NULL CHECK (input_type IN ('url', 'html', 'template')),
    output_format   VARCHAR(10) NOT NULL CHECK (output_format IN ('pdf', 'png', 'jpeg', 'webp')),
    delivery_method VARCHAR(20) NOT NULL CHECK (delivery_method IN ('sync', 'webhook', 's3')) DEFAULT 'sync',
    input_source    TEXT NOT NULL,
    input_data      JSONB NULL,
    options         JSONB NOT NULL DEFAULT '{}',
    delivery_config JSONB NULL,
    result_url      TEXT NULL,
    result_size     BIGINT NULL,
    pages_count     INT NULL,
    duration_ms     INT NULL,
    error_message   TEXT NULL,
    is_test         BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ NULL
);

CREATE INDEX idx_jobs_org_id ON jobs(org_id);
CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_created_at ON jobs(created_at DESC);

-- Usage Monthly
CREATE TABLE usage_monthly (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id            UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    month             DATE NOT NULL,
    conversions       INT NOT NULL DEFAULT 0,
    test_conversions  INT NOT NULL DEFAULT 0,
    overage_amount    DECIMAL(10,4) NOT NULL DEFAULT 0,
    UNIQUE(org_id, month)
);

CREATE INDEX idx_usage_monthly_org_month ON usage_monthly(org_id, month);

-- Templates
CREATE TABLE templates (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by   UUID NOT NULL REFERENCES users(id),
    name         VARCHAR(100) NOT NULL,
    engine       VARCHAR(20) NOT NULL CHECK (engine IN ('handlebars', 'liquid')),
    html_content TEXT NOT NULL,
    css_content  TEXT NULL,
    sample_data  JSONB NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_templates_org_id ON templates(org_id);

-- Seed data: Plans
INSERT INTO plans (name, monthly_quota, overage_price, price_monthly, price_yearly, max_file_size, timeout_seconds, features) VALUES
(
    'free',
    100,
    0,
    0,
    0,
    5242880,
    15,
    '{"templates": false, "webhooks": false, "custom_headers": false, "priority_queue": false}'
),
(
    'starter',
    2500,
    0.0080,
    19.00,
    190.00,
    10485760,
    30,
    '{"templates": true, "webhooks": false, "custom_headers": true, "priority_queue": false}'
),
(
    'pro',
    15000,
    0.0050,
    49.00,
    490.00,
    26214400,
    60,
    '{"templates": true, "webhooks": true, "custom_headers": true, "priority_queue": true}'
),
(
    'business',
    100000,
    0.0030,
    149.00,
    1490.00,
    52428800,
    120,
    '{"templates": true, "webhooks": true, "custom_headers": true, "priority_queue": true}'
);
