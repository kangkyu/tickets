# Makefile for tickets-by-uma backend

.PHONY: test test-integration test-unit test-coverage test-setup clean build run

# Test commands
test: test-unit test-integration

test-unit:
	@echo "Running unit tests..."
	go test -v -short ./apphandlers/...

test-integration: test-setup
	@echo "Running integration tests..."
	go test -v -run "^Test" -timeout 30s .

test-coverage: test-setup
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out -covermode=atomic .
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-setup:
	@echo "Setting up test database..."
	@if ! psql -lqt | cut -d \| -f 1 | grep -qw tickets_uma_test; then \
		createdb tickets_uma_test || true; \
	fi
	@echo "Running migrations on test database..."
	@PGPASSWORD=password psql -h localhost -U postgres -d tickets_uma_test -f db/schema.sql > /dev/null 2>&1 || echo "Schema already applied"

clean:
	@echo "Cleaning up test artifacts..."
	rm -f coverage.out coverage.html
	@echo "Dropping test database..."
	@dropdb tickets_uma_test || true

build:
	@echo "Building application..."
	go build -o bin/tickets-uma .

run:
	@echo "Starting application..."
	go run .

# Development helpers
dev-deps:
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go mod tidy

lint:
	@echo "Running linter..."
	golangci-lint run

format:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Database helpers
db-reset:
	@echo "Resetting main database..."
	@PGPASSWORD=password psql -h localhost -U postgres -c "DROP DATABASE IF EXISTS tickets_uma;"
	@PGPASSWORD=password psql -h localhost -U postgres -c "CREATE DATABASE tickets_uma;"
	@PGPASSWORD=password psql -h localhost -U postgres -d tickets_uma -f db/schema.sql
	@PGPASSWORD=password psql -h localhost -U postgres -d tickets_uma -f db/seed.sql

db-migrate:
	@echo "Running database migrations..."
	@PGPASSWORD=password psql -h localhost -U postgres -d tickets_uma -f db/schema.sql

# Help
help:
	@echo "Available commands:"
	@echo "  test              - Run all tests"
	@echo "  test-unit         - Run unit tests only"
	@echo "  test-integration  - Run integration tests only"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  test-setup        - Set up test database"
	@echo "  clean             - Clean test artifacts and databases"
	@echo "  build             - Build the application"
	@echo "  run               - Run the application"
	@echo "  dev-deps          - Install development dependencies"
	@echo "  lint              - Run linter"
	@echo "  format            - Format code"
	@echo "  db-reset          - Reset main database"
	@echo "  db-migrate        - Run database migrations"
	@echo "  help              - Show this help"