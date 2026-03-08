# Docbiner — Design Document

**Date** : 2026-03-04
**Auteur** : Jean-Denis VIDOT
**Statut** : Validé

---

## Vision

Docbiner est un concurrent sérieux à PDFShift et DocRaptor sur le marché de la conversion HTML → PDF/images via API. Le positionnement repose sur deux piliers : **qualité de rendu** (Chromium headless) et **Developer Experience** (playground, SDKs, CLI).

Le pricing n'est pas nécessairement agressif — il y a suffisamment de place sur le marché pour capter une part significative avec un produit supérieur.

---

## Formats supportés

- **MVP** : HTML → PDF + images (PNG, JPEG, WebP)
- **Futur** : DOCX, XLSX, et autres formats

## Modèles d'input

| Mode | Description |
|---|---|
| **URL** | Docbiner charge l'URL et la convertit |
| **HTML brut** | HTML envoyé dans le body de la requête |
| **Templates** | Templates serveur (Handlebars/Liquid) avec injection de données |

## Modèles d'output

| Mode | Description |
|---|---|
| **Sync** | Le fichier est retourné directement dans la réponse HTTP |
| **Webhook** | Notification async via callback URL quand la conversion est terminée |
| **S3 upload** | Upload direct vers le bucket S3/GCS/R2 du client |

---

## Architecture

### Stack technique

| Composant | Tech |
|---|---|
| API + Workers | Go (Full Go) |
| Moteur de rendu | Chromium headless via CDP (chromedp/rod) |
| Dashboard | Next.js 15 (App Router) + Tailwind + shadcn/ui |
| Base de données | PostgreSQL 17 |
| Cache / Sessions | Redis 7 |
| Message Queue | NATS JetStream |
| Stockage temporaire | Minio (S3-compatible) |
| Reverse proxy | Caddy (auto-TLS) |
| Infra | Docker Compose sur VPS-3 OVHCloud (8 vCPU, 24 Go RAM, 200 Go NVMe) |

### Architecture microservices

```
                    ┌─────────────────┐
                    │   Next.js App   │
                    │   (Dashboard)   │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │     Caddy       │
                    │  Reverse Proxy  │
                    │  + Auto-TLS     │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
    ┌─────────▼──────┐      │     ┌────────▼────────┐
    │   API Service  │      │     │   Dashboard BFF  │
    │   (Go - Echo)  │      │     │   (Next.js API)  │
    └─────────┬──────┘      │     └────────┬────────┘
              │              │              │
              │      ┌───────▼───────┐     │
              │      │     NATS      │     │
              │      │  JetStream    │     │
              │      └───────┬───────┘     │
              │              │              │
              │    ┌─────────▼─────────┐   │
              │    │  Worker Service   │   │
              │    │  (Go + chromedp)  │   │
              │    │  x N replicas     │   │
              │    └─────────┬─────────┘   │
              │              │              │
    ┌─────────▼──────────────▼──────────────▼──┐
    │              PostgreSQL 17                │
    └──────────────────────────────────────────┘
              │              │
    ┌─────────▼──┐  ┌───────▼───────┐
    │   Redis    │  │    Minio      │
    └────────────┘  └───────────────┘
```

### Flux de conversion

1. Client → API : `POST /v1/convert` avec HTML/URL + options
2. API valide l'auth (API key), le quota, crée un job en DB
3. API publie le job sur NATS JetStream
4. Worker prend le job, lance Chromium, convertit
5. Worker upload le résultat sur Minio (ou S3 client)
6. Worker notifie l'API via NATS que c'est terminé
7. API retourne le PDF (sync) ou envoie le webhook (async)

---

## Modèle de données

### Organizations (tenant principal)

```sql
organizations
├── id                  UUID PK
├── name                VARCHAR
├── slug                VARCHAR UNIQUE
├── plan_id             FK → plans
├── stripe_customer_id  VARCHAR
├── created_at          TIMESTAMPTZ
└── updated_at          TIMESTAMPTZ
```

### Users

```sql
users
├── id              UUID PK
├── email           VARCHAR UNIQUE
├── password_hash   VARCHAR
├── username        VARCHAR UNIQUE
├── display_name    VARCHAR
├── avatar_url      VARCHAR NULL
├── created_at      TIMESTAMPTZ
└── updated_at      TIMESTAMPTZ
```

### Organization Members

```sql
org_members
├── id          UUID PK
├── org_id      FK → organizations
├── user_id     FK → users
├── role        ENUM (owner, admin, member)
├── invited_by  FK → users NULL
├── created_at  TIMESTAMPTZ
└── UNIQUE(org_id, user_id)
```

### Plans

```sql
plans
├── id              UUID PK
├── name            VARCHAR (free, starter, pro, business)
├── monthly_quota   INT
├── overage_price   DECIMAL
├── price_monthly   DECIMAL
├── price_yearly    DECIMAL
├── max_file_size   INT (Mo)
├── timeout_seconds INT
├── features        JSONB
└── active          BOOLEAN
```

### API Keys

```sql
api_keys
├── id          UUID PK
├── org_id      FK → organizations
├── created_by  FK → users
├── key_hash    VARCHAR (SHA-256)
├── key_prefix  VARCHAR (8 chars : "db_live_xxx")
├── name        VARCHAR
├── environment ENUM (live, test)
├── permissions JSONB
├── last_used_at TIMESTAMPTZ
├── expires_at  TIMESTAMPTZ NULL
└── created_at  TIMESTAMPTZ
```

### Jobs

```sql
jobs
├── id              UUID PK
├── org_id          FK → organizations
├── api_key_id      FK → api_keys
├── status          ENUM (pending, processing, completed, failed)
├── input_type      ENUM (url, html, template)
├── input_source    TEXT
├── input_data      JSONB NULL
├── output_format   ENUM (pdf, png, jpeg, webp)
├── options         JSONB
├── delivery_method ENUM (sync, webhook, s3)
├── delivery_config JSONB
├── result_url      VARCHAR NULL
├── result_size     INT NULL
├── pages_count     INT NULL
├── duration_ms     INT NULL
├── error_message   TEXT NULL
├── is_test         BOOLEAN
├── created_at      TIMESTAMPTZ
└── completed_at    TIMESTAMPTZ NULL
```

### Usage Monthly

```sql
usage_monthly
├── id                  UUID PK
├── org_id              FK → organizations
├── month               DATE
├── conversions         INT
├── test_conversions    INT
├── overage_amount      DECIMAL
└── UNIQUE(org_id, month)
```

### Templates

```sql
templates
├── id           UUID PK
├── org_id       FK → organizations
├── created_by   FK → users
├── name         VARCHAR
├── engine       ENUM (handlebars, liquid)
├── html_content TEXT
├── css_content  TEXT NULL
├── sample_data  JSONB NULL
├── created_at   TIMESTAMPTZ
└── updated_at   TIMESTAMPTZ
```

**Points clés :**
- Clés API hashées (SHA-256), jamais stockées en clair
- Séparation live/test au niveau de la clé API
- Multi-tenant : tout est lié à l'organisation, pas à l'utilisateur
- Un user solo = 1 org avec 1 membre (owner)

---

## API Design

### Base URL & Auth

```
Base: https://api.docbiner.com/v1
Auth: Header Authorization: Bearer db_live_xxxxx
```

### Endpoints

**Conversion**

```
POST   /v1/convert              Conversion sync
POST   /v1/convert/async        Conversion async
GET    /v1/jobs/{id}            Statut d'un job
GET    /v1/jobs/{id}/download   Télécharger le résultat
GET    /v1/jobs                 Lister les jobs
DELETE /v1/jobs/{id}            Supprimer un job
```

**Templates**

```
POST   /v1/templates            Créer
GET    /v1/templates            Lister
GET    /v1/templates/{id}       Détail
PUT    /v1/templates/{id}       Mettre à jour
DELETE /v1/templates/{id}       Supprimer
POST   /v1/templates/{id}/preview   Preview
```

**Merge**

```
POST   /v1/merge                Fusionner plusieurs sources en un PDF
```

**Usage & Account**

```
GET    /v1/usage                Usage du mois courant
GET    /v1/usage/history        Historique mensuel
GET    /v1/account              Infos du compte/org
```

### Exemple : Conversion sync

```bash
curl -X POST https://api.docbiner.com/v1/convert \
  -H "Authorization: Bearer db_live_abc123" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "https://example.com/invoice/42",
    "format": "pdf",
    "options": {
      "landscape": false,
      "page_size": "A4",
      "margin": { "top": "20mm", "bottom": "20mm" },
      "header": {
        "html": "<div style=\"font-size:10px\">Facture #42</div>"
      },
      "footer": {
        "html": "<div style=\"font-size:9px\">Page <span class=\"pageNumber\"></span>/<span class=\"totalPages\"></span></div>"
      },
      "watermark": { "text": "CONFIDENTIEL", "opacity": 0.1 },
      "encrypt": { "password": "secret123", "restrict": ["print", "copy"] },
      "css": "body { font-family: Inter, sans-serif; }",
      "js": "document.querySelectorAll('.no-print').forEach(e => e.remove())",
      "wait_for": "#content-loaded",
      "delay_ms": 500
    }
  }'
```

### Codes d'erreur

| Code | Signification |
|---|---|
| 401 | API key invalide ou manquante |
| 402 | Quota dépassé |
| 422 | Payload invalide |
| 429 | Rate limit atteint |
| 500 | Erreur interne |
| 504 | Timeout de conversion |

---

## Dashboard

### Pages

| Page | Description |
|---|---|
| Login / Register | Auth email+password, OAuth Google/GitHub |
| Onboarding | Création d'org, choix du plan, première clé API |
| Overview | Stats du mois, graphe d'usage, derniers jobs |
| API Keys | CRUD des clés live/test, copie, rotation, expiration |
| Jobs / History | Liste paginée, filtres, détail+download |
| Templates | Éditeur avec preview live du PDF |
| Playground | Éditeur HTML/CSS live → génère PDF en temps réel |
| Usage & Billing | Usage mensuel, historique, Stripe Customer Portal |
| Settings | Profil, org, membres, webhooks/S3 par défaut |

### Stack frontend

- Next.js 15 (App Router)
- Tailwind CSS + shadcn/ui
- NextAuth.js
- TanStack Query
- Monaco Editor (playground + éditeur de templates)
- Recharts (graphes d'usage)
- Stripe Customer Portal + Stripe Elements

### Playground

Feature DX phare :
- Panneau gauche : Monaco Editor (HTML/CSS)
- Panneau droit : Preview PDF intégrée
- Barre d'options : format, taille, orientation, headers/footers
- Bouton "Generate" : appelle l'API avec clé test interne
- Bouton "Copy cURL" : génère la commande correspondante
- Conversions gratuites et watermarkées

---

## SDKs & CLI

### SDKs cibles

| Langage | Package | Priorité |
|---|---|---|
| Node.js / TypeScript | `@docbiner/sdk` | P0 |
| Python | `docbiner` | P0 |
| Go | `github.com/docbiner/docbiner-go` | P1 |
| PHP | `docbiner/docbiner-php` | P1 |
| Ruby | `docbiner` (gem) | P2 |

### Principes SDK

- Wrapper mince autour de l'API REST
- Typage complet (TypeScript, Python type hints, Go structs)
- Retry automatique avec backoff exponentiel sur 5xx
- Streaming du résultat pour les gros fichiers
- Erreurs typées et documentées

### CLI

```bash
docbiner auth login / status
docbiner convert file.html -o file.pdf
docbiner convert https://example.com -o page.png --format png
docbiner convert template.hbs -d data.json -o report.pdf
docbiner templates list / push / preview
docbiner merge page1.html page2.html -o combined.pdf
docbiner usage / usage --history
```

Installation : `npm install -g @docbiner/cli` ou `brew install docbiner`

---

## Pricing

### Modèle

Plans avec quota mensuel + overage + documents de test gratuits (watermarkés).

### Benchmark concurrence

| | PDFShift | DocRaptor | Docbiner (cible) |
|---|---|---|---|
| Free | 50 conv/mois | 5 docs/mois | À définir |
| Premier plan | 9$/mois (500) | 15$/mois (125) | À définir |
| Coût min/conv | ~0.004$ | ~0.025$ | À définir |
| Tests gratuits | Non | Oui (watermark) | Oui (watermark) |

Les grilles tarifaires exactes seront définies avant le launch.

---

## Déploiement

### Infrastructure

- **VPS-3 OVHCloud** : 8 vCPU, 24 Go RAM, 200 Go NVMe
- **Orchestration** : Docker Compose
- **CI/CD** : GitHub Actions → build → push images → SSH deploy

### Services Docker Compose

| Service | Replicas | RAM estimée |
|---|---|---|
| API (Go + Echo) | 2 | 200 Mo |
| Workers (Go + chromedp) | 4 | 1.6 Go |
| Dashboard (Next.js) | 1 | 200 Mo |
| PostgreSQL 17 | 1 | 1 Go |
| Redis 7 | 1 | 256 Mo |
| NATS JetStream | 1 | 64 Mo |
| Minio | 1 | 256 Mo |
| Caddy | 1 | 64 Mo |
| **Total** | — | **~4 Go** |

Marge restante : ~20 Go RAM, possibilité de monter à 10-15 workers.

### Monitoring

- Prometheus (métriques)
- Grafana (dashboards ops)
- Loki (logs)
- Alertmanager (alertes)

### Backup

- PostgreSQL : pg_dump quotidien → S3 distant
- Minio : fichiers temporaires (TTL 24h), pas de backup

---

## Features MVP

| Feature | Description |
|---|---|
| HTML → PDF | Conversion via Chromium headless |
| HTML → Images | Screenshots PNG/JPEG/WebP |
| URL / HTML / Templates | Trois modes d'input |
| Sync / Webhook / S3 | Trois modes de delivery |
| Headers & Footers | HTML custom pour les en-têtes/pieds de page |
| Watermark | Texte ou image en filigrane |
| Encryption | Protection par mot de passe, restriction print/copy |
| CSS/JS Injection | Injection de CSS et JavaScript avant conversion |
| PDF Merge | Combiner plusieurs sources en un seul PDF |
| Playground interactif | Éditeur live HTML → PDF |
| SDKs (Node, Python) | P0 au lancement |
| CLI | Outil en ligne de commande |
| Dashboard complet | Clés API, historique, analytics, billing |
| Multi-tenant orgs | Support des organisations avec rôles |
| Docs de test | Conversions gratuites watermarkées |
