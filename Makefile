.PHONY: help test-operational test-decisional test-all coverage coverage-decisional check clean build-local frontend frontend-dev frontend-kill lint lint-operational lint-decisional lint-frontend up down rebuild restart-operational restart-decisional logs logs-operational logs-decisional shell-operational shell-decisional adminer rabbitmq ps prune

.DEFAULT_GOAL := help

# packages relative to operational-node/
PACKAGES := ./internal/handlers/... ./internal/services/... ./internal/repositories/...

help:
	@echo "Available commands:"
	@echo "\n-> Testing & Local"
	@echo "  make test-operational    - run operational-node unit tests"
	@echo "  make test-decisional     - run decisional-node unit tests"
	@echo "  make test-all            - run all tests (operational + decisional)"
	@echo "  make coverage            - run operational-node tests and generate html coverage report"
	@echo "  make coverage-decisional - run decisional-node tests with coverage summary"
	@echo "  make check               - run test-all + lint (pre-push gate)"
	@echo "  make build-local         - quickly compile the operational-node binary locally"
	@echo "  make clean               - remove temporary files"
	@echo "\n-> Linting"
	@echo "  make lint                - lint all services (operational vet + decisional ruff + eslint)"
	@echo "  make lint-operational    - vet operational-node"
	@echo "  make lint-decisional     - ruff check decisional-node"
	@echo "  make lint-frontend       - eslint frontend"
	@echo "\n-> Frontend"
	@echo "  make frontend            - start Vite dev server detached (logs -> frontend/vite.log)"
	@echo "  make frontend-dev        - start Vite dev server in foreground"
	@echo "  make frontend-kill       - kill detached Vite dev server"
	@echo "\n-> Docker Environment"
	@echo "  make up                  - start the entire infrastructure in background"
	@echo "  make down                - stop and remove all containers"
	@echo "  make rebuild             - force rebuild docker images and restart"
	@echo "  make restart-operational - restart operational-node container"
	@echo "  make restart-decisional  - restart decisional-node container"
	@echo "  make logs                - tail logs for all services"
	@echo "  make logs-operational    - tail operational-node logs only"
	@echo "  make logs-decisional     - tail decisional-node logs only"
	@echo "  make shell-operational   - exec shell into operational-node container"
	@echo "  make shell-decisional    - exec shell into decisional-node container"
	@echo "  make adminer             - open Adminer in browser (port 8082)"
	@echo "  make rabbitmq            - open RabbitMQ management UI in browser (port 15672)"
	@echo "  make ps                  - show status of all containers"
	@echo "  make prune               - remove containers + local images (keeps DB/RabbitMQ volumes)"

# TESTING & LOCAL
test-operational:
	@echo "Running operational-node unit tests..."
	cd operational-node && go test -v $(PACKAGES)

test-decisional:
	@echo "Running decisional-node unit tests..."
	cd decisional-node && python3 -m pytest tests/ -v

test-all: test-operational test-decisional

coverage:
	@echo "Calculating coverage..."
	cd operational-node && go test -coverprofile=coverage.out $(PACKAGES)
	@echo "Coverage summary:"
	cd operational-node && go tool cover -func=coverage.out
	@echo "Generating html report..."
	cd operational-node && go tool cover -html=coverage.out -o coverage_report.html
	@echo "Done. Saved to operational-node/coverage_report.html"

coverage-decisional:
	@echo "Running decisional-node tests with coverage..."
	cd decisional-node && python3 -m pytest tests/ -v --cov=. --cov-report=term-missing

check: test-all lint

build-local:
	@echo "Building operational-node binary locally..."
	cd operational-node && go build -o tmp/main ./cmd/api/main.go

clean:
	@echo "Cleaning up..."
	cd operational-node && rm -f coverage.out coverage_report.html
	cd operational-node && rm -rf tmp/
	find decisional-node -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null; true
	find decisional-node -type d -name .pytest_cache -exec rm -rf {} + 2>/dev/null; true

# LINTING
lint-operational:
	@echo "Linting operational-node..."
	cd operational-node && go vet ./...

lint-decisional:
	@echo "Linting decisional-node..."
	cd decisional-node && ruff check .

lint-frontend:
	@echo "Linting frontend..."
	cd frontend && npm run lint

lint: lint-operational lint-decisional lint-frontend

# FRONTEND
frontend:
	@echo "Starting Vite dev server (detached)... logs -> frontend/vite.log"
	cd frontend && nohup npm run dev > vite.log 2>&1 &

frontend-dev:
	@echo "Starting Vite dev server..."
	cd frontend && npm run dev

frontend-kill:
	@echo "Killing Vite dev server..."
	@pkill -f "vite" || echo "No Vite process found"

# DOCKER ENVIRONMENT (runs from root)
up:
	@echo "Starting docker infrastructure..."
	docker compose up -d

down:
	@echo "Stopping docker infrastructure..."
	docker compose down

rebuild:
	@echo "Rebuilding and restarting docker containers..."
	docker compose up --build -d

restart-operational:
	@echo "Restarting operational-node..."
	docker compose restart operational-node

restart-decisional:
	@echo "Restarting decisional-node..."
	docker compose restart decisional-node

logs:
	docker compose logs -f

logs-operational:
	docker compose logs -f operational-node

logs-decisional:
	docker compose logs -f decisional-node

shell-operational:
	docker compose exec operational-node sh

shell-decisional:
	docker compose exec decisional-node sh

ps:
	docker compose ps

prune:
	@echo "Removing project containers and local images (volumes preserved)..."
	docker compose down --remove-orphans --rmi local

adminer:
	xdg-open http://localhost:8082 > /dev/null 2>&1 &

rabbitmq:
	xdg-open http://localhost:15672 > /dev/null 2>&1 &
