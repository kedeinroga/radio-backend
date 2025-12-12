# Radio Backend

A production-ready radio streaming proxy backend built with Go, following Clean Architecture principles.

## Features

- ✅ **Clean Architecture** - Hexagonal architecture with clear separation of concerns
- ✅ **Authentication** - JWT-based authentication with guest/premium user support
- ✅ **Analytics** - Comprehensive behavior tracking with PostgreSQL + Redis
- ✅ **Logging** - Structured JSON logging with request tracing
- ✅ **Code Quality** - Enforced via golangci-lint with strict rules
- ✅ **CI/CD** - GitHub Actions for automated testing and deployment
- ✅ **Secrets Management** - Environment-based configuration
- ✅ **Error Codes** - Standardized error codes for frontend i18n
- ✅ **Docker** - Multi-stage builds with security best practices

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    External Interfaces                   │
│              (HTTP, CLI, Message Queues)                 │
└───────────────────┬─────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────┐
│                  Interface Adapters                      │
│         (Handlers, Presenters, Controllers)              │
└───────────────────┬─────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────┐
│                  Application Business Rules              │
│                  (Use Cases / Services)                  │
└───────────────────┬─────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────┐
│               Enterprise Business Rules                  │
│                  (Entities / Domain)                     │
└─────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Make (optional)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd radio-backend
```

2. Copy environment file:
```bash
cp .env.example .env
```

3. Generate JWT keys:
```bash
make generate-keys
```

4. Install dependencies:
```bash
go mod download
```

5. Run database migrations:
```bash
make migrate-up
```

6. Start the server:
```bash
make run
```

The server will start on `http://localhost:8080`

## API Endpoints

### Authentication

- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login user
- `POST /api/v1/auth/refresh` - Refresh access token
- `GET /api/v1/auth/me` - Get current user (requires auth)

### Stations

- `GET /api/v1/stations/popular` - Get popular stations
- `GET /api/v1/stations/search?q=jazz` - Search stations

### Analytics (Premium Only)

- `GET /api/v1/analytics/stations/popular` - Most played stations
- `GET /api/v1/analytics/searches/trending` - Trending searches
- `GET /api/v1/analytics/users/active` - Active users count

## Development

### Run with live reload

```bash
make dev
```

### Run tests

```bash
make test
```

### Run linters

```bash
make lint
```

### Format code

```bash
make fmt
```

### Security scan

```bash
make security
```

## Docker

### Build image

```bash
make docker-build
```

### Run container

```bash
make docker-run
```

## Configuration

All configuration is done via environment variables. See `.env.example` for all available options.

Key configuration:

- `SERVER_PORT` - Server port (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `JWT_PRIVATE_KEY_PATH` - Path to JWT private key
- `JWT_PUBLIC_KEY_PATH` - Path to JWT public key

## Project Structure

```
radio-backend/
├── cmd/                    # Application entry points
│   └── server/            # Main server application
├── internal/              # Private application code
│   ├── domain/           # Business entities and interfaces
│   ├── services/         # Business logic (use cases)
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── repositories/     # Data access implementations
│   ├── infrastructure/   # External concerns (DB, cache, logger)
│   ├── config/          # Configuration management
│   └── server/          # Server setup and routing
├── migrations/          # Database migrations
├── locales/            # i18n translation files
├── .github/workflows/  # CI/CD pipelines
└── docs/               # Documentation
```

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for coding standards and contribution guidelines.

## License

MIT License - see LICENSE file for details
