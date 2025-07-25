# LFX v2 Access Check Service

A access check service for the LFX v2 platform, providing centralized authorization and permission management across LFX services.

## 🏗️ Architecture Overview

```mermaid
graph TB
    subgraph "Client Applications"
        A[Web UI]
        B[CLI Tools]
        C[Mobile Apps]
    end

    subgraph "LFX v2 Access Check Service"
        D[HTTP Server<br/>:8080]
        E[JWT Auth<br/>Middleware]
        F[Access Service<br/>Core Logic]
        G[Health Check<br/>Endpoints]
    end

    subgraph "External Dependencies"
        H[Heimdall<br/>JWT Provider]
        I[NATS<br/>Message Queue]
        J[Downstream<br/>Services]
    end

    A --> D
    B --> D
    C --> D
    
    D --> E
    E --> F
    F --> I
    I --> J
    
    E <--> H
    
    D --> G
    
    style D fill:#e1f5fe
    style F fill:#f3e5f5
    style H fill:#fff3e0
    style I fill:#e8f5e8
```

## 🔄 Request Flow

```mermaid
sequenceDiagram
    participant Client
    participant Gateway as API Gateway
    participant Access as Access Check Service
    participant Heimdall as Heimdall Auth
    participant NATS as NATS Queue
    participant Service as Target Service

    Client->>Gateway: POST /check-access<br/>Bearer: JWT
    Gateway->>Access: Forward request
    
    Access->>Heimdall: Validate JWT token
    Heimdall-->>Access: Return claims & principal
    
    Access->>Access: Extract resource-action pairs
    Access->>NATS: Publish access check request
    NATS->>Service: Route to appropriate service
    Service-->>NATS: Return allow/deny decisions
    NATS-->>Access: Aggregate responses
    
    Access-->>Gateway: JSON response with results
    Gateway-->>Client: Access decisions
    
    Note over Access: All requests logged<br/>with request ID
```

## 🚀 Quick Start

### Prerequisites

- **Go**: 1.24.0 
- **Docker**: For containerized deployment
- **NATS**: Message queue for service communication
- **Heimdall**: JWT authentication provider

### Local Development

1. **Clone the repository**
   ```bash
   git clone https://github.com/linuxfoundation/lfx-v2-access-check.git
   cd lfx-v2-access-check
   ```

2. **Install dependencies**
   ```bash
   make deps
   ```

3. **Generate API code** (if needed)
   ```bash
   make apigen
   ```

4. **Build the service**
   ```bash
   make build
   ```

5. **Run tests**
   ```bash
   make test
   ```

6. **Start the service**
   ```bash
   ./bin/lfx-access-check
   ```

### Configuration

The service is configured via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `HOST` | Server host address | `0.0.0.0` |
| `PORT` | Server port | `8080` |
| `DEBUG` | Enable debug logging | `false` |
| `JWKS_URL` | Heimdall JWKS endpoint | `http://heimdall:4457/.well-known/jwks` |
| `AUDIENCE` | JWT audience | `access-check` |
| `ISSUER` | JWT issuer | `heimdall` |
| `NATS_URL` | NATS server URL | `nats://nats:4222` |

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

### Health Endpoints

- **Liveness**: `GET /livez` - Basic service health
- **Readiness**: `GET /readyz` - Service + dependencies health

## 🏛️ Architecture Details

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

### Project Structure

```
├── cmd/lfx-access-check/    # Application entry point
├── design/                  # Goa API design definitions
├── gen/                     # Generated API code (Goa)
├── internal/
│   ├── container/          # Dependency injection
│   ├── domain/contracts/   # Domain models & interfaces
│   ├── infrastructure/     # External service adapters
│   ├── middleware/         # HTTP middleware
│   ├── service/           # Core business logic
│   └── mocks/             # Test mocks
├── pkg/
│   ├── constants/         # Application constants
│   └── log/              # Structured logging utilities
├── test/integration/      # Integration tests
└── charts/               # Helm deployment charts
```

## 🚢 Deployment

### Kubernetes with Helm

```bash
# Install/upgrade with Helm
helm upgrade --install lfx-v2-access-check ./charts/lfx-v2-access-check \
  --set image.tag=latest \
  --set config.jwksUrl=http://heimdall:4457/.well-known/jwks \
  --set config.natsUrl=nats://nats:4222
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

