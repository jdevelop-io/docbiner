.PHONY: migrate-up migrate-down migrate-create db-reset dev-infra dev-infra-down dev-api dev-worker dev-dashboard dev-setup

DATABASE_URL ?= postgresql://docbiner:docbiner_dev@localhost:5434/docbiner?sslmode=disable

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

# Dev infrastructure (just the services, no app containers)
dev-infra:
	docker compose up -d postgres redis nats minio

dev-infra-down:
	docker compose down

# Run API locally (requires dev-infra)
dev-api:
	cd services/api && go run .

# Run Worker locally (requires dev-infra + chromium installed)
dev-worker:
	cd services/worker && go run .

# Run Dashboard locally
dev-dashboard:
	cd services/dashboard && npm run dev

# Full dev setup
dev-setup: dev-infra
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@$(MAKE) migrate-up
	@echo "Dev infrastructure ready!"
	@echo "Run 'make dev-api' and 'make dev-worker' in separate terminals"
