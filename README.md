# Radio Backend

A production-ready radio streaming proxy backend built with Go, following Clean Architecture principles.

## 🚀 Quick Start

### Local Development with Docker
```bash
# Start all services (PostgreSQL + Redis + App)
./docker.sh up

# The app will be available at http://localhost:8080
```

📖 **[Complete Docker Guide →](docs/DOCKER_GUIDE.md)**

### Production Deployment (Supabase + Upstash + Cloud Run)

#### Terraform + GitHub Actions (Automated CI/CD) 🏗️

Infrastructure as Code con deployment automático para gestionar toda la infraestructura:

```bash
# 1. Inicializar Terraform
make tf-init

# 2. Ver qué se va a crear
make tf-plan

# 3. Crear infraestructura (una sola vez)
make tf-apply

# 4. Configurar secretos en GitHub (una sola vez)
make tf-github-secrets

# 5. Deploy automático!
git push origin main  # GitHub Actions se encarga del resto
```

**Ventajas:**
- ✅ CI/CD completamente automatizado con GitHub Actions
- ✅ Infraestructura como código con Terraform
- ✅ Estado rastreado en GCS bucket
- ✅ Detecta cambios manuales (drift detection)
- ✅ Tests + Linting + Build + Deploy automáticos
- ✅ Migraciones de BD automáticas
- ✅ Versionado con Git
- ✅ Rollback fácil
- ✅ Multi-ambiente (staging/production)

📖 **[Terraform Quick Start →](terraform/QUICKSTART.md)**  
📖 **[Terraform Full Guide →](terraform/README.md)**  
📖 **[GitHub Actions Workflows →](.github/workflows/README.md)**  
📖 **[Ejemplos Prácticos →](terraform/EXAMPLES.md)**

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

### Option 1: Docker (Recommended) 🐳

El método más rápido para empezar. Incluye PostgreSQL, Redis y la aplicación.

```bash
# 1. Generar JWT keys
make generate-keys

# 2. Levantar todos los servicios
make docker-up

# 3. Verificar
make docker-health
```

**¡Listo!** → http://localhost:8080

📖 **[Guía completa de Docker →](docs/DOCKER_GUIDE.md)**

### Option 2: Local Development

### Prerequisites

- Go 1.24+
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
- `POST /api/v1/auth/logout` - **NEW** Logout user (blacklists JWT token)
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

### Admin Maintenance (Admin Only) 🔧 NEW

- `GET /api/v1/admin/maintenance/recommendations` - **NEW** Get intelligent maintenance recommendations
- `POST /api/v1/admin/maintenance/refresh-views?type=all|seo|analytics` - **NEW** Refresh materialized views
- `GET /api/v1/admin/maintenance/refresh-stats?days=7` - **NEW** View refresh statistics
- `POST /api/v1/admin/maintenance/cleanup-partitions?retention_months=12` - **NEW** Clean up old partitions
- `GET /api/v1/admin/maintenance/check-partitions?months_ahead=3` - **NEW** Verify future partitions exist
- `GET /api/v1/admin/maintenance/partition-status` - **NEW** Get partition sizes and row counts
- `POST /api/v1/admin/maintenance/full` - **NEW** Execute complete maintenance routine

📖 **[Complete Maintenance Guide →](docs/MAINTENANCE.md)**

### Admin Monitoring (Admin Only) 📊 NEW

- `GET /api/v1/admin/monitoring/health` - **NEW** Get comprehensive system health metrics
- `GET /api/v1/admin/monitoring/alerts` - **NEW** Get active system alerts and warnings

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

### Local Development

Build and test the Docker image locally:

```bash
# Build production image
make docker-build-prod

# Test locally
make docker-test

# Inspect image layers
make docker-inspect
```

### Cloud Run Deployment

This project is optimized for Google Cloud Run with a multi-stage Dockerfile.

#### Quick Deploy with GitHub Actions (Recommended)

1. **Setup GCP and GitHub Secrets**
   ```bash
   # Run the automated setup script
   chmod +x scripts/setup-github-actions.sh
   ./scripts/setup-github-actions.sh YOUR_PROJECT_ID YOUR_GITHUB_ORG YOUR_REPO_NAME
   ```

2. **Setup Secrets in Secret Manager**
   ```bash
   ./scripts/setup-secrets.sh YOUR_PROJECT_ID
   ```

3. **Deploy**
   ```bash
   # Deploy to staging
   git push origin staging

   # Deploy to production
   git push origin main

   # Or create a release tag
   git tag v1.0.0
   git push origin v1.0.0
   ```

#### Manual Deploy

```bash
# Setup and deploy everything
make cloudrun-all

# Or step by step:
make cloudrun-setup          # Setup GCP prerequisites
make cloudrun-secrets        # Configure secrets
make cloudrun-build          # Build and push image
make cloudrun-deploy         # Deploy to Cloud Run

# View logs
make cloudrun-logs
make cloudrun-logs-tail      # Real-time logs

# Get service URL
make cloudrun-url
```

📖 **[Complete Deployment Guide →](docs/CLOUDRUN_DEPLOYMENT.md)**  
📖 **[GitHub Actions Setup →](docs/GITHUB_ACTIONS_SETUP.md)**


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
