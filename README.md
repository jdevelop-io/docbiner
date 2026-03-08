# Docbiner

**HTML-to-document conversion API.** Convert HTML pages, URLs, and templates into PDF, PNG, JPEG, or WebP — synchronously or via an async job queue.

[![CI](https://github.com/jdevelop-io/docbiner/actions/workflows/ci.yml/badge.svg)](https://github.com/jdevelop-io/docbiner/actions/workflows/ci.yml)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)
![Next.js](https://img.shields.io/badge/Next.js-16-000000?logo=next.js)

## Overview

Docbiner is a self-hosted document generation platform built for developers. It exposes a REST API that renders HTML into documents using a headless Chromium engine, with support for:

- **Synchronous conversion** — send HTML or a URL, get a PDF/image back immediately
- **Asynchronous jobs** — queue conversions via NATS JetStream, get results via webhook or S3
- **Template engine** — define reusable Handlebars or Liquid templates, inject JSON data at render time
- **PDF merge** — combine multiple PDFs into a single document
- **Multi-format output** — PDF, PNG, JPEG, WebP
- **Usage tracking & billing** — per-organization quotas with Stripe integration

## Architecture

```
                          ┌──────────────┐
                          │   Dashboard  │
                          │  (Next.js)   │
                          │   :3000      │
                          └──────┬───────┘
                                 │ JWT auth
                                 ▼
┌──────────┐  API key   ┌──────────────┐        ┌──────────────┐
│  Client  │ ─────────► │     API      │ ─────► │    NATS      │
│  / SDK   │            │   (Go/Echo)  │  pub   │  JetStream   │
└──────────┘            │   :8080      │        └──────┬───────┘
                        └──────┬───────┘               │ sub
                               │ sync                  ▼
                               │ render       ┌──────────────┐
                               ▼              │   Worker(s)  │
                        ┌──────────────┐      │    (Go)      │
                        │  Chromium    │◄─────│  replicas: 2 │
                        │  (headless)  │      └──────┬───────┘
                        └──────────────┘             │
                                                     ▼
                 ┌────────────┐  ┌───────┐  ┌──────────────┐
                 │ PostgreSQL │  │ Redis │  │    Minio     │
                 │    :5434   │  │ :6381 │  │  (S3) :9000  │
                 └────────────┘  └───────┘  └──────────────┘
```

## Tech Stack

| Layer | Technology |
|-------|-----------|
| API | Go 1.25, Echo v4 |
| Worker | Go 1.25, NATS JetStream |
| Dashboard | Next.js 16, React 19, TailwindCSS 4, Radix UI |
| Database | PostgreSQL 17 |
| Cache | Redis 7 |
| Message Queue | NATS 2 (JetStream) |
| Object Storage | Minio (S3-compatible) |
| Rendering | Chromium (headless, via chromedp) |
| Payments | Stripe |
| CI/CD | GitHub Actions, Docker (ghcr.io) |

## Features

- Convert HTML or URL to PDF, PNG, JPEG, WebP
- Async job processing with webhook or S3 delivery
- Handlebars and Liquid template engine with live preview
- PDF merge and encryption
- API key authentication (live/test environments)
- Multi-organization support with role-based access
- Usage tracking with monthly quotas and overage billing
- Dashboard with job history, usage charts, and template editor
- Node.js and Python SDKs

## Project Structure

```
docbiner/
├── services/
│   ├── api/            # REST API (Go/Echo)
│   ├── worker/         # Async job processor (Go/NATS)
│   └── dashboard/      # Web UI (Next.js)
├── internal/
│   ├── auth/           # JWT authentication
│   ├── billing/        # Stripe integration
│   ├── config/         # Environment configuration
│   ├── database/       # PostgreSQL repositories
│   ├── delivery/       # Webhook & S3 delivery
│   ├── domain/         # Domain models
│   ├── pdfutil/        # PDF merge & encryption
│   ├── queue/          # NATS JetStream client
│   ├── renderer/       # Chromium rendering (chromedp)
│   ├── storage/        # Minio/S3 client
│   ├── tmpl/           # Template engines (Handlebars, Liquid)
│   └── usage/          # Usage tracking & quotas
├── migrations/         # PostgreSQL migrations
├── sdks/
│   ├── node/           # Node.js SDK
│   └── python/         # Python SDK
├── monitoring/         # Prometheus, Grafana, Loki, AlertManager
├── docker-compose.yml  # Development stack
└── Makefile            # Dev commands
```

## Getting Started

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and Docker Compose
- [Make](https://www.gnu.org/software/make/)
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI

### Setup

1. **Clone the repository**

```bash
git clone https://github.com/jdevelop-io/docbiner.git
cd docbiner
```

2. **Copy the environment file**

```bash
cp .env.example .env
```

The default values work out of the box for local development.

3. **Start all services**

```bash
docker compose up -d --build
```

This starts PostgreSQL, Redis, NATS, Minio, the API, and workers.

4. **Run database migrations**

```bash
make migrate-up
```

5. **Start the dashboard**

```bash
cd services/dashboard
npm install
npm run dev
```

### Access Points

| Service | URL |
|---------|-----|
| Dashboard | http://localhost:3000 |
| API | http://localhost:8080 |
| API Health | http://localhost:8080/health |
| Minio Console | http://localhost:9001 |
| NATS Monitoring | http://localhost:8222 |

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_USER` | `docbiner` | PostgreSQL username |
| `POSTGRES_PASSWORD` | `docbiner_dev` | PostgreSQL password |
| `POSTGRES_DB` | `docbiner` | PostgreSQL database name |
| `DATABASE_URL` | `postgresql://docbiner:docbiner_dev@postgres:5432/docbiner?sslmode=disable` | Full connection string |
| `REDIS_URL` | `redis://redis:6379` | Redis connection URL |
| `NATS_URL` | `nats://nats:4222` | NATS connection URL |
| `MINIO_ROOT_USER` | `docbiner` | Minio access key |
| `MINIO_ROOT_PASSWORD` | `docbiner_dev_secret` | Minio secret key |
| `MINIO_ENDPOINT` | `minio:9000` | Minio endpoint |
| `MINIO_BUCKET` | `docbiner-files` | Minio bucket name |
| `API_PORT` | `8080` | API server port |
| `CHROMIUM_PATH` | `/usr/bin/chromium-browser` | Path to Chromium binary (inside container) |

### Stripe (Optional)

Required only if you work on billing features.

| Variable | Description |
|----------|-------------|
| `STRIPE_SECRET_KEY` | Stripe API secret key |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret |

Set up a [Stripe CLI](https://stripe.com/docs/stripe-cli) webhook listener for local development:

```bash
stripe listen --forward-to localhost:8080/v1/webhooks/stripe
```

## API Overview

All endpoints are under `/v1`. Authentication is via API key (`X-API-Key` header) or JWT (`Authorization: Bearer <token>` header).

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/auth/register` | Create account |
| `POST` | `/v1/auth/login` | Get JWT token |
| `GET` | `/v1/auth/me` | Current user (JWT) |

### Conversion

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/convert` | Synchronous conversion |
| `POST` | `/v1/convert/async` | Async conversion (returns job ID) |

### Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/jobs` | List jobs |
| `GET` | `/v1/jobs/:id` | Get job details |
| `GET` | `/v1/jobs/:id/download` | Download result |
| `DELETE` | `/v1/jobs/:id` | Delete job |

### Templates

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/templates` | Create template |
| `GET` | `/v1/templates` | List templates |
| `GET` | `/v1/templates/:id` | Get template |
| `PUT` | `/v1/templates/:id` | Update template |
| `DELETE` | `/v1/templates/:id` | Delete template |
| `POST` | `/v1/templates/preview` | Preview inline template |
| `POST` | `/v1/templates/:id/preview` | Preview saved template |

### Other

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/merge` | Merge multiple PDFs |
| `GET` | `/v1/usage` | Current usage stats |
| `GET` | `/v1/usage/history` | Usage history |

## SDKs

### Node.js

```bash
cd sdks/node
npm install
npm run build
```

### Python

```bash
cd sdks/python
pip install -e ".[dev]"
pytest
```

## Monitoring

The `monitoring/` directory contains configuration for a full observability stack:

- **Prometheus** — metrics collection (`:9090`)
- **Grafana** — dashboards (`:3001`)
- **Loki** — log aggregation (`:3100`)
- **AlertManager** — alerting

Start the monitoring stack:

```bash
docker compose -f monitoring/docker-compose.yml up -d
```

## Makefile Reference

| Command | Description |
|---------|-------------|
| `make dev-infra` | Start infrastructure (PostgreSQL, Redis, NATS, Minio) |
| `make dev-infra-down` | Stop infrastructure |
| `make dev-setup` | Start infrastructure + run migrations |
| `make dev-api` | Run API locally (requires `dev-infra`) |
| `make dev-worker` | Run Worker locally (requires `dev-infra` + Chromium) |
| `make dev-dashboard` | Run Dashboard locally |
| `make migrate-up` | Apply all pending migrations |
| `make migrate-down` | Rollback last migration |
| `make migrate-create` | Create a new migration |
| `make db-reset` | Drop and recreate database |

## License

Proprietary. All rights reserved.
