# LFX v2 Access Check Service

## Quick Overview

- **Purpose**: Bulk access checks for resource-action pairs
- **Framework**: Go with GOA v3 (API-first design)  
- **Authentication**: JWT tokens from Heimdall
- **Message Queue**: NATS for async processing
- **Deployment**: Kubernetes with Helm charts

## Architecture

```
Client → Traefik → Heimdall → Access Check Service → NATS
```

## Project Structure

```
lfx-v2-access-check/
├── cmd/lfx-access-check/           # Application entry point
│   ├── main.go                     # Application bootstrap
│   └── server.go                   # HTTP server setup
│
├── design/                         # GOA API design definitions
│   ├── access-svc.go              # Service design & endpoints
│   └── types.go                   # Shared type definitions
│
├── gen/                           # Generated code (GOA)
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
## Development Setup

### Prerequisites
- Go 1.24.0+
- Docker
- NATS server
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

## API

### Access Check
```
POST /access-check?v=1
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
  "version": "1",
  "requests": ["project:read:proj-123", "committee:write:comm-456"]
}
```

### Health Checks
- `GET /livez` - Liveness probe
- `GET /readyz` - Readiness probe

## Deployment

### Docker
```bash
make docker-build
docker run -p 8080:8080 -e JWKS_URL=... -e NATS_URL=... lfx-access-check
```

### Kubernetes
```bash
helm upgrade --install lfx-v2-access-check ./charts/lfx-v2-access-check
```

## Service Architecture

### Core Components

1. **HTTP Server** (`cmd/lfx-access-check/`)
   - GOA-based REST API server
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
- **Integration Tests**: API endpoints, NATS integration, JWT authentication
- **Benchmark Tests**: Performance testing for critical paths

### Running Tests
```bash
# Unit tests
make test

# Integration tests
make test-integration

# Specific package tests
go test ./internal/service/

# Test coverage
make test-coverage
```

## Deployment

### Docker Deployment
```bash
# Build image
make docker-build

# Run container
docker run -p 8080:8080 \
  -e JWKS_URL=http://heimdall:4457/.well-known/jwks \
  -e NATS_URL=nats://nats:4222 \
  linuxfoundation/lfx-access-check:latest
```

### Kubernetes Deployment
```bash
helm upgrade --install lfx-v2-access-check ./charts/lfx-v2-access-check \
  --set image.tag=v1.0.0 \
  --set config.jwksUrl=http://heimdall:4457/.well-known/jwks \
  --set config.natsUrl=nats://nats:4222 \
  --namespace lfx
```

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
