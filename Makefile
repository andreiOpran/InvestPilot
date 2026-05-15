make coverage.PHONY: help test-operational test-decisional test-all coverage coverage-decisional check clean build-local frontend frontend-dev frontend-kill lint lint-operational lint-decisional lint-frontend up down rebuild restart-operational restart-decisional logs logs-operational logs-decisional shell-operational shell-decisional adminer rabbitmq ps prune k8s-status k8s-pods k8s-logs-operational k8s-logs-decisional k8s-logs-frontend k8s-restart-operational k8s-restart-decisional k8s-restart-frontend k8s-restart-all k8s-apply k8s-describe k8s-secrets k8s-apply-secret-go k8s-apply-secret-python k8s-apply-secret-frontend k8s-apply-secrets

NS := investpilot

.DEFAULT_GOAL := help

# packages relative to operational-node/
PACKAGES := ./internal/handlers/... ./internal/services/... ./internal/repositories/...

help:
	@echo "Available commands:"
	@echo "\n-> Testing & Local"
	@echo "  make test-operational     - run operational-node unit tests"
	@echo "  make test-decisional      - run decisional-node unit tests"
	@echo "  make test-all             - run all tests (operational + decisional)"
	@echo "  make coverage-operational - run operational-node tests and generate html coverage report"
	@echo "  make coverage-decisional  - run decisional-node tests with coverage summary"
	@echo "  make check                - run test-all + lint (pre-push gate)"
	@echo "  make build-local          - quickly compile the operational-node binary locally"
	@echo "  make clean                - remove temporary files"
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
	@echo "\n-> Kubernetes (namespace: investpilot)"
	@echo "  make k8s-status                - show deployments + services + ingress"
	@echo "  make k8s-pods                  - list all pods with status"
	@echo "  make k8s-logs-operational      - tail operational-node pod logs"
	@echo "  make k8s-logs-decisional       - tail decisional-node pod logs"
	@echo "  make k8s-logs-frontend         - tail nginx-frontend pod logs"
	@echo "  make k8s-restart-operational   - rollout restart operational-node"
	@echo "  make k8s-restart-decisional    - rollout restart decisional-node"
	@echo "  make k8s-restart-frontend      - rollout restart nginx-frontend"
	@echo "  make k8s-restart-all           - rollout restart all deployments"
	@echo "  make k8s-apply                 - apply all manifests in k8s/"
	@echo "  make k8s-describe              - describe all pods (events + state)"
	@echo "  make k8s-secrets               - list all secrets in namespace"
	@echo "  make k8s-apply-secret-go       - apply go-secrets.yaml (copy from go-secrets.example)"
	@echo "  make k8s-apply-secret-python   - apply python-secrets.yaml (copy from python-secrets.example)"
	@echo "  make k8s-apply-secret-frontend - apply frontend-secrets.yaml (copy from frontend-secrets.example)"
	@echo "  make k8s-apply-secrets         - apply all secret manifests"

# TESTING & LOCAL
test-operational:
	@echo "Running operational-node unit tests..."
	cd operational-node && go test -v $(PACKAGES)

test-decisional:
	@echo "Running decisional-node unit tests..."
	cd decisional-node && python3 -m pytest tests/ -v

test-all: test-operational test-decisional

coverage-operational:
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
	rm -f decisional-node/.coverage
	rm -rf decisional-node/.ruff_cache
	rm -f frontend/vite.log

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

# KUBERNETES
k8s-status:
	kubectl get deployments,services,ingress -n $(NS)

k8s-pods:
	kubectl get pods -n $(NS) -o wide

k8s-logs-operational:
	kubectl logs -n $(NS) -l app=operational-node --tail=100 -f

k8s-logs-decisional:
	kubectl logs -n $(NS) -l app=decisional-node --tail=100 -f

k8s-logs-frontend:
	kubectl logs -n $(NS) -l app=nginx-frontend --tail=100 -f

k8s-restart-operational:
	kubectl rollout restart deployment/operational-node -n $(NS)

k8s-restart-decisional:
	kubectl rollout restart deployment/decisional-node -n $(NS)

k8s-restart-frontend:
	kubectl rollout restart deployment/nginx-frontend -n $(NS)

k8s-restart-all: k8s-restart-operational k8s-restart-decisional k8s-restart-frontend

k8s-apply:
	kubectl apply -f k8s/

k8s-describe:
	kubectl describe pods -n $(NS)

k8s-secrets:
	kubectl get secrets -n $(NS)

k8s-apply-secret-go:
	kubectl apply -f k8s/go-secrets.yaml

k8s-apply-secret-python:
	kubectl apply -f k8s/python-secrets.yaml

k8s-apply-secret-frontend:
	kubectl apply -f k8s/frontend-secrets.yaml

k8s-apply-secrets: k8s-apply-secret-go k8s-apply-secret-python k8s-apply-secret-frontend

