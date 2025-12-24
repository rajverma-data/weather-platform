.PHONY: help build test clean run-server run-ingester migrate-up migrate-down docker-up docker-down

help:
	@echo "Weather Platform - Available Commands"
	@echo "======================================"
	@echo "build            - Build all binaries"
	@echo "test             - Run all tests"
	@echo "clean            - Clean build artifacts"
	@echo "run-server       - Run API server"
	@echo "run-ingester     - Run data ingester"
	@echo "migrate-up       - Run database migrations"
	@echo "migrate-down     - Rollback database migrations"
	@echo "docker-up        - Start Docker containers"
	@echo "docker-down      - Stop Docker containers"

build:
	@echo "Building binaries..."
	@mkdir -p bin
	@go build -o bin/weather-api ./cmd/server
	@go build -o bin/weather-ingester ./cmd/ingester
	@go build -o bin/weather-migrate ./cmd/migrate
	@echo "Build complete!"

test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Test complete! Coverage report: coverage.html"

clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "Clean complete!"

run-server: build
	@echo "Starting API server..."
	@./bin/weather-api

run-ingester: build
	@echo "Starting data ingester..."
	@./bin/weather-ingester -data-dir=./wx_data -batch-size=1000 -calculate-stats

migrate-up: build
	@echo "Running migrations..."
	@./bin/weather-migrate -direction=up

migrate-down: build
	@echo "Rolling back migrations..."
	@./bin/weather-migrate -direction=down

docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down

docker-logs:
	@docker-compose logs -f

# Database setup
db-setup: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 5
	@$(MAKE) migrate-up
	@echo "Database setup complete!"
