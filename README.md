# Radio Backend

A production-ready radio streaming proxy backend built with Go, following Clean Architecture principles.

## 🆕 What's New in v2.0

### Authentication & Session Management
- ✅ **JWT with RFC 7519 Claims** - Standard claims (exp, iat, jti, iss, sub) for better security
- ✅ **Token Validation Endpoint** - Validate tokens without loading full user data
- ✅ **Token Revocation** - Revoke tokens individually, by session, or all at once
- ✅ **Session Management** - Track and manage user sessions across devices
- ✅ **Redis Blacklist** - Instant token revocation with Redis
- ✅ **Security Event Logging** - Complete audit trail of authentication events

### 🔐 Security Enhancements (v2.1)
- ✅ **Timing Attack Prevention** - Constant-time operations to prevent user enumeration
- ✅ **HTTPS Enforcement** - Automatic redirect to HTTPS in production
- ✅ **Account Lockout** - 10 failed attempts = 30 min lockout
- ✅ **Email Rate Limiting** - 10 attempts/hour per email (Redis-based)
- ✅ **Password Strength** - Special characters required + 47 common passwords blocked
- ✅ **Session Hijacking Protection** - User-Agent validation
- ✅ **Enhanced Security Logging** - Failed login attempts with detailed metadata

**Security Score**: 🎯 **100/100** (audited Dec 2025)

📖 **[Complete Authentication Guide →](docs/AUTHENTICATION.md)**  
📖 **[Security Improvements →](SECURITY_IMPROVEMENTS.md)**  
📖 **[Security Audit →](audit.md)**  
📖 **[Migration Guide →](docs/MIGRATION_GUIDE.md)**  
📖 **[Quick Reference →](QUICK_REFERENCE.md)**

## Features

- ✅ **Clean Architecture** - Hexagonal architecture with clear separation of concerns
- ✅ **Authentication** - JWT-based with RS256, token revocation, and session tracking
- ✅ **Security** - Account lockout, email rate limiting, timing attack prevention, HTTPS enforcement
- ✅ **Session Management** - Multi-device tracking with hijacking detection
- ✅ **Token Revocation** - Instant token invalidation with Redis blacklist
- ✅ **Rate Limiting** - IP-based (5/15min) + Email-based (10/hour) protection
- ✅ **Security Headers** - CORS, CSP, HSTS, and request size limits
- ✅ **Analytics** - Comprehensive behavior tracking with PostgreSQL + Redis
- ✅ **SEO Optimization** - Dynamic sitemap, URL slugs, and rich metadata for search engines
- ✅ **Internationalization** - Multi-language support (ES, EN, FR, DE)
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
- `POST /api/v1/auth/login` - Login user (returns enhanced response with session info)
- `POST /api/v1/auth/refresh` - Refresh access token (returns enhanced response)
- `GET /api/v1/auth/me` - Get current user (requires auth)
- `POST /api/v1/auth/validate` - **NEW** Validate token and get claims
- `POST /api/v1/auth/revoke` - **NEW** Revoke tokens (by token_id, session_id, or all)
- `GET /api/v1/auth/sessions` - **NEW** List active sessions
- `DELETE /api/v1/auth/sessions/:id` - **NEW** Terminate specific session

📖 **[Full API Documentation →](docs/AUTHENTICATION.md)**

### Stations

- `GET /api/v1/stations/popular` - Get popular stations
- `GET /api/v1/stations/search?q=jazz` - Search stations

### Analytics (Premium Only)

- `GET /api/v1/analytics/stations/popular` - Most played stations
- `GET /api/v1/analytics/searches/trending` - Trending searches
- `GET /api/v1/analytics/users/active` - Active users count

### Admin Security (Admin Only) 🔐 NEW

- `GET /api/v1/admin/security/metrics?period=7d` - **NEW** Security metrics and trends
- `GET /api/v1/admin/security/logs` - **NEW** Security event logs with filtering

📖 **[Security Admin Endpoints Documentation →](docs/SECURITY_ADMIN_ENDPOINTS.md)**

### SEO (Public)

- `GET /api/v1/seo/sitemap-data` - Data for generating dynamic sitemap
- `GET /api/v1/seo/popular-tags` - Top 100 genres/tags
- `GET /api/v1/seo/popular-countries` - Top 50 countries
- `POST /api/v1/seo/refresh-stats` - Manually refresh SEO statistics (admin)

## API Documentation

Interactive API documentation is available via Swagger UI:

**Development**: `http://localhost:8080/swagger/index.html`

The documentation is automatically generated from code annotations and includes:
- Complete endpoint descriptions
- Request/response schemas
- Authentication requirements
- Query parameters
- Example requests

To regenerate documentation after code changes:
```bash
make swagger-generate
```

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

### Database Migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create NAME=add_new_table
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
- `SERVER_BASE_URL` - Base URL for SEO canonical links (e.g., https://tudominio.com)

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
    ├── CONTRIBUTING.md     # Coding standards
    ├── SETUP.md            # Setup guide
    ├── ERROR_CODES.md      # Error code reference
    └── SEO_USAGE.md        # SEO implementation guide
```

## Contributing

See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for coding standards and contribution guidelines.

## License

MIT License - see LICENSE file for details
