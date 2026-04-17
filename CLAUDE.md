# LFX Access Check Service

## Quick Overview

- **Purpose**: Bulk access checks for resource-action pairs
- **Framework**: Go with Goa v3 (API-first design)
- **Authentication**: JWT tokens from Heimdall
- **Message Queue**: NATS for async processing; fga-sync evaluates permissions
- **Deployment**: Kubernetes with Helm charts

## Architecture

```text
Client → Traefik → Heimdall → Access Check Service → NATS → fga-sync
```

## Project Structure

```
lfx-v2-access-check/
├── cmd/lfx-access-check/           # Application entry point
│   ├── main.go                     # Application bootstrap
│   └── server.go                   # HTTP server setup
│
├── design/                         # Goa API design definitions
│   ├── access-svc.go              # Service design & endpoints
│   └── types.go                   # Shared type definitions
│
├── gen/                           # Generated code (Goa) — do not edit
│   ├── access_svc/                # Service interfaces
│   └── http/                      # HTTP transport layer
│
├── internal/                      # Private application code
│   ├── container/                 # Dependency injection
│   ├── domain/contracts/          # Domain models & interfaces
│   ├── infrastructure/            # External service adapters
│   │   ├── auth/                 # Heimdall JWT validation
│   │   ├── config/               # Configuration management
│   │   └── messaging/            # NATS integration
│   ├── middleware/                # HTTP middleware
│   ├── service/                   # Core business logic
│   └── mocks/                     # Test mocks
│
├── pkg/                           # Public packages
│   ├── constants/                 # Application constants
│   └── log/                       # Logging utilities
│
└── charts/                        # Helm deployment charts
```

## Development Setup

### Prerequisites
- Go 1.24.0+
- Docker
- NATS server
- fga-sync (evaluates permissions from NATS messages)
- Heimdall (JWT provider)

### Quick Start
```bash
git clone https://github.com/linuxfoundation/lfx-v2-access-check.git
cd lfx-v2-access-check
make deps
make build
make test
./bin/lfx-access-check
```

### Make Targets
Available development and build targets:

**Development:**
```bash
make setup-dev        # Install development tools (golangci-lint)
make setup            # Setup development environment
make deps             # Install Go dependencies
make apigen           # Generate API code using Goa
make fmt              # Format Go code
make vet              # Run go vet
make lint             # Run golangci-lint
make clean            # Clean build artifacts
```

**Building:**
```bash
make build            # Build for local OS
make build-linux      # Build for Linux (production)
make run              # Build and run locally
```

**Testing:**
```bash
make test             # Run unit tests with race detection
make test-coverage    # Run tests with HTML coverage report
```

**Docker:**
```bash
make docker-build     # Build Docker image
make docker-push      # Push image to registry
make docker-run       # Run container locally
```

**Helm/Kubernetes:**
```bash
make helm-install         # Install Helm chart using values.yaml
make helm-install-local   # Install Helm chart using values.local.yaml
make helm-templates       # Render Helm templates
make helm-templates-local # Render Helm templates using values.local.yaml
make helm-uninstall       # Uninstall Helm release
make helm-lint            # Lint Helm chart
```

**Utility:**
```bash
make help             # Show all available targets
```

## API

### Access Check

Version is passed as a query parameter (`?v=1`), not in the request body.

```http
POST /access-check?v=1
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "requests": ["project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor", "committee:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer"]
}
```

Response (results are unordered — match on `object#relation@user` prefix):

```json
{
  "results": ["project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue", "committee:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer@user:auth0|alice\tfalse"]
}
```

### Health Checks
- `GET /livez` - Liveness probe
- `GET /readyz` - Readiness probe

### OpenAPI Spec
Available at `/_access-check/openapi.json`, `openapi.yaml`, `openapi3.json`, `openapi3.yaml`.

## Deployment

### Docker
```bash
make docker-build
docker run -p 8080:8080 ghcr.io/linuxfoundation/lfx-v2-access-check/lfx-access-check:latest
```

### Kubernetes
```bash
make helm-install
```

## Service Architecture

### Core Components

1. **HTTP Server** (`cmd/lfx-access-check/`)
   - Goa-based REST API server
   - JWT authentication middleware
   - Request ID tracking
   - Structured logging

2. **Access Service** (`internal/service/`)
   - Core business logic
   - JWT token validation
   - NATS message publishing
   - Response aggregation

3. **Infrastructure Layer** (`internal/infrastructure/`)
   - **Auth Repository**: Heimdall JWT validation
   - **Messaging Repository**: NATS communication
   - **Config**: Environment-based configuration

4. **Domain Contracts** (`internal/domain/contracts/`)
   - Shared data structures
   - JWT claims modeling
   - Service interfaces

## Testing

### Test Structure
- **Unit Tests**: Service layer, infrastructure, configuration, middleware
- **Integration Tests**: API endpoints with mock dependencies — no external services required

### Running Tests
```bash
# Unit tests
make test

# Integration tests (uses mocks — no external services needed)
go test -v ./test/integration/

# Specific package tests
go test ./internal/service/

# Test coverage
make test-coverage
```

### Integration Test Files
- `access_check_test.go` - Tests access check endpoint with JWT validation
- `health_test.go` - Tests health check endpoints (/livez, /readyz)
- `plaintext_test.go` - Tests plaintext response handling
- `mocks.go` - Mock auth and messaging repositories

## Security

### Authentication & Authorization
- **JWT Validation**: All requests require valid JWT tokens
- **JWKS Integration**: Dynamic key rotation support
- **Audience Validation**: Ensures tokens are intended for this service
- **Issuer Validation**: Verifies tokens from trusted Heimdall

### Network Security
- **TLS Termination**: At gateway level (Traefik)
- **Internal Communication**: Service-to-service via Kubernetes networking
- **NATS Security**: Authenticated NATS connections

## Monitoring & Observability

### Logging
- **Structured Logging**: JSON format with consistent fields
- **Request Tracking**: Unique request IDs for correlation
- **Log Levels**: DEBUG, INFO, WARN, ERROR
- **Context Propagation**: Request context through service layers

### Health Checks
- **Liveness Probe**: `/livez` - Service basic health
- **Readiness Probe**: `/readyz` - Service + dependencies health
