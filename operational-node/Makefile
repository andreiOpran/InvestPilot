.PHONY: test coverage clean

# set help as the default action
.DEFAULT_GOAL := help

# core packages to maintain >80% coverage
PACKAGES := ./internal/handlers/... ./internal/services/... ./internal/repositories/...

# show available commands
help:
	@echo "Available commands:"
	@echo "  make test      - run standard unit tests"
	@echo "  make coverage  - run tests and generate html coverage report"
	@echo "  make clean     - remove temporary coverage files"
	@echo "  make help      - show this help message"

# run standard unit tests
test:
	@echo "Running unit tests..."
	go test -v $(PACKAGES)

# run tests and generate html coverage report
coverage:
	@echo "Calculating coverage..."
	go test -coverprofile=coverage.out $(PACKAGES)
	@echo "Coverage summary:"
	go tool cover -func=coverage.out
	@echo "Generating HTML report..."
	go tool cover -html=coverage.out -o coverage_report.html
	@echo "Done. Saved to coverage_report.html"

# clean temporary files
clean:
	@echo "Cleaning up..."
	rm -f coverage.out coverage_report.html