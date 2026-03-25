.PHONY: help test coverage clean build-local up down rebuild logs

.DEFAULT_GOAL := help

# go packages relative to operational-node/
PACKAGES := ./internal/handlers/... ./internal/services/... ./internal/repositories/...

help:
	@echo "Available commands:"
	@echo "\n-> Testing & Local"
	@echo "  make test         - run standard go unit tests"
	@echo "  make coverage     - run tests and generate html coverage report"
	@echo "  make build-local  - quickly compile the go binary locally"
	@echo "  make clean        - remove temporary files"
	@echo "\n-> Docker Environment"
	@echo "  make up           - start the entire infrastructure in background"
	@echo "  make down         - stop and remove all containers"
	@echo "  make rebuild      - force rebuild docker images and restart"
	@echo "  make logs         - tail logs for all services"

# TESTING & LOCAL (needs "cd operational-node")
test:
	@echo "Running unit tests..."
	cd operational-node && go test -v $(PACKAGES)

coverage:
	@echo "Calculating coverage..."
	cd operational-node && go test -coverprofile=coverage.out $(PACKAGES)
	@echo "Coverage summary:"
	cd operational-node && go tool cover -func=coverage.out
	@echo "Generating html report..."
	cd operational-node && go tool cover -html=coverage.out -o coverage_report.html
	@echo "Done. Saved to operational-node/coverage_report.html"

build-local:
	@echo "Building go binary locally..."
	cd operational-node && go build -o tmp/main ./cmd/api/main.go

clean:
	@echo "Cleaning up..."
	cd operational-node && rm -f coverage.out coverage_report.html
	cd operational-node && rm -rf tmp/

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

logs:
	docker compose logs -f