.PHONY: build run clean docker-build docker-run docker-push test fmt lint deps

# Variables
APP_NAME = pk-shorts
DOCKER_IMAGE = rashpile/pk-shorts
DOCKER_TAG = latest
GO = go
GOFLAGS = -ldflags="-w -s"

# Build the application
build:
	@echo "Building $(APP_NAME)..."
	$(GO) build $(GOFLAGS) -o $(APP_NAME) .

# Run the application locally
run: build
	@echo "Running $(APP_NAME)..."
	./$(APP_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(APP_NAME)
	rm -f links.db
	$(GO) clean

# Install dependencies
deps:
	@echo "Installing dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -cover ./...

# Docker commands
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: docker-build
	@echo "Running Docker container..."
	docker run -d \
		--name $(APP_NAME) \
		-p 8080:8080 \
		-v $(PWD)/data:/app/data \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(APP_NAME) || true
	docker rm $(APP_NAME) || true

docker-push: docker-build
	@echo "Pushing Docker image to registry..."
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)

# Development with hot reload
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

# Initialize project
init: deps
	@echo "Initializing project..."
	mkdir -p templates static data

# Help
help:
	@echo "Available targets:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Build and run the application"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Install/update dependencies"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linter"
	@echo "  make test         - Run tests"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Build and run Docker container"
	@echo "  make docker-stop  - Stop and remove Docker container"
	@echo "  make docker-push  - Push Docker image to registry"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo "  make init         - Initialize project structure"
	@echo "  make help         - Show this help message"