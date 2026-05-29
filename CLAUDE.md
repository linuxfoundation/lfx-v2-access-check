# LFX Access Check Service

> **Central LFX skills:**
> - `lfx-skills:lfx` for cross-repo tasks, "where does X live" questions, owner/peer repo routing, and missing checkouts.
> - `lfx-skills:lfx-platform-architecture` for platform composition, V2 service classes (this service is a proxy/consumer), write/read/access-check flows, NATS/KV ownership, and handoffs across FGA, indexer, query, Heimdall, OpenFGA, Helm, ArgoCD.
> - The repo-local `access-check-dev` skill auto-attaches on Go, design, `gen/`, `Makefile`, chart, docs, and local Claude guidance paths. It owns repo-local Go conventions and access-check implementation truth (HTTP-over-NATS wrapper, centralized subject constants, unordered-response semantics, my-grants direct tuple reads, and the do-not-publish-index/access-mutation rule).
> - Local rule `.claude/rules/access-check-boundaries.md` codifies the consumer-service boundary. Keep it aligned with code and chart behavior.
> - If the central plugin is missing, install with `/plugin marketplace add linuxfoundation/lfx-skills` then `/plugin install lfx-skills@lfx-skills`.

## Quick Overview

- **Purpose**: Bulk access checks for resource-action pairs
- **Framework**: Go with Goa v3 (API-first design)
- **Authentication**: Heimdall-issued JWT validation through JWKS
- **Messaging**: NATS request/reply against `lfx-v2-fga-sync`; this service never publishes index or tuple-mutation messages
- **Deployment**: Kubernetes with Helm charts

## Agent Guidance

This service is a **consumer wrapper** over the access-check contract owned by `lfx-v2-fga-sync`. It owns its HTTP surface (see `docs/access-check-contract.md`) and nothing else. Tuple semantics, the OpenFGA model, cache behavior, and the canonical request/reply envelope live in `lfx-v2-fga-sync/docs/fga-sync-contract.md`. Deployed values and chart pins live in `lfx-v2-argocd`.

Repo-owned guidance:

- `.claude/skills/access-check-dev/SKILL.md` for repo-local Go, Goa, chart, contract-doc, and access-check implementation truth.
- `.claude/skills/access-check-dev/references/goa-patterns.md` for access-check Goa methods and local deviations.
- `.claude/skills/access-check-dev/references/nats-messaging.md` for the subject set and request/reply call shape.
- `.claude/rules/access-check-boundaries.md` for the consumer-service boundary.
- `docs/access-check-contract.md` for the authoritative HTTP API contract.
- `docs/service-helm-chart.md` for this repo's chart interface.

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
├── gen/                           # Generated code (Goa), do not edit
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

The full API contract (request/response shape, unordered-response caveat,
error mapping, timeout semantics, health checks, OpenAPI spec paths) lives in
[`docs/access-check-contract.md`](docs/access-check-contract.md). Read that
file before changing the HTTP surface.

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

## Testing

### Test Structure
- **Unit Tests**: Service layer, infrastructure, configuration, middleware
- **Integration Tests**: API endpoints with mock dependencies, no external services required

### Running Tests
```bash
# Unit tests
make test

# Integration tests (uses mocks, no external services needed)
go test -v ./test/integration/

# Specific package tests
go test ./internal/service/

# Test coverage
make test-coverage
```

### Integration Test Files
- `access_check_test.go` - Tests access check endpoint with JWT validation
- `my_grants_test.go` - Tests my-grants endpoint with JWT validation
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

Repo-owned logger discipline (slog, OpenTelemetry-aware fields, request-id and principal context propagation, log levels) lives in path-scoped `access-check-dev` guidance.

### Health Checks
- **Liveness Probe**: `/livez` - Service basic health
- **Readiness Probe**: `/readyz` - Service + dependencies health
