# =============================================================================
# Nafer Backend Makefile
# =============================================================================

.PHONY: help up down build logs ps clean infra

# Default target
help:
	@echo ""
	@echo "  Nafer Backend — Available Commands"
	@echo "  -----------------------------------"
	@echo "  make up        Start all services (build if needed)"
	@echo "  make down      Stop all services"
	@echo "  make build     Build all service images"
	@echo "  make logs      Tail logs from all services"
	@echo "  make ps        Show running containers"
	@echo "  make infra     Start infrastructure only (postgres, redis, minio, meilisearch)"
	@echo "  make clean     Stop services and remove volumes (DANGER: deletes data)"
	@echo ""

# Start everything
up:
	docker compose up -d --build

# Start infrastructure only (useful during development)
infra:
	docker compose up -d postgres redis minio meilisearch

# Stop all services
down:
	docker compose down

# Build all images without starting
build:
	docker compose build

# Stream logs
logs:
	docker compose logs -f

# Show container status
ps:
	docker compose ps

# Remove everything including volumes — USE WITH CAUTION
clean:
	docker compose down -v --remove-orphans
