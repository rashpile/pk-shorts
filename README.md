# PK Shorts - Simple Link Shortener

A lightweight, fast, and simple link shortener service written in Go with an embedded BoltDB database.

## Features

- 🔗 Simple URL shortening with random 8-character IDs
- 🔐 Optional secure mode with 16-character IDs (resistant to guessing attacks)
- ✏️ Custom ID support - choose your own memorable short links
- 📊 Click tracking for each shortened link
- 🗑️ Delete functionality for managing links
- 🎨 Clean, responsive web UI (no JavaScript frameworks)
- 🗄️ Embedded BoltDB database (no external dependencies)
- 🐳 Small Docker image (~37MB)
- 🚀 Fast and lightweight
- 🔄 RESTful API endpoints
- 📦 Multi-platform Docker support (linux/amd64, linux/arm64)

## Quick Start

### Using Docker

```bash
docker run -d \
  --name pk-shorts \
  -p 8080:8080 \
  -v $(pwd)/data:/app/data \
  -e SHORT_PREFIX=/s \
  -e UI_PREFIX=/sui \
  ghcr.io/rashpile/pk-shorts:latest
```

### Using Go

```bash
# Install dependencies
go mod download

# Run the service
go run main.go

# Or build and run
make build
./pk-shorts
```

## API Endpoints

- **Web UI**: `http://localhost:8080/sui`
- **Create short link**: `POST /sui/api/create`
  - Standard: `{"url": "https://example.com"}`
  - Secure: `{"url": "https://example.com", "secure": true}`
  - Custom ID: `{"url": "https://example.com", "custom_id": "my-link"}`
- **List all links**: `GET /sui/api/list`
- **Delete link**: `DELETE /sui/api/delete/{shortcode}`
- **Redirect**: `GET /s/{shortcode}`
- **Health check**: `GET /health`

## Custom IDs

When creating custom IDs, follow these rules:
- 3-50 characters long
- Only letters (a-z, A-Z), numbers (0-9), dashes (-), and underscores (_)
- Must be unique (not already in use)
- Cannot use reserved words: api, admin, health, static, assets, js, css
- Secure mode is disabled when using custom IDs

## Configuration

Environment variables:
- `PORT`: Server port (default: 8080)
- `SHORT_PREFIX`: URL prefix for short links (default: /s)
- `UI_PREFIX`: URL prefix for UI (default: /sui)

## Development

```bash
# Build
make build

# Run tests
make test

# Format code
make fmt

# Build Docker image
make docker-build

# Run with hot reload
make dev
```

## GitHub Actions

The repository includes GitHub Actions workflow for:
- Running tests
- Building multi-platform Docker images
- Publishing to GitHub Container Registry (ghcr.io)
- Generating SBOM (Software Bill of Materials)

## Repository

```bash
git@github.com:rashpile/pk-shorts.git
```

## License

MIT