# Radio Backend

A production-ready radio streaming proxy backend built with Go, designed for high performance, security, and scalability. It follows strict Clean Architecture principles to ensure maintainability and testability.

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

Infrastructure as Code with automated deployment to manage the entire stack:

```bash
# 1. Initialize Terraform
make tf-init

# 2. Preview changes
make tf-plan

# 3. Apply infrastructure (one-time setup)
make tf-apply

# 4. Configure GitHub Secrets (one-time setup)
make tf-github-secrets

# 5. Automatic Deploy!
git push origin main  # GitHub Actions handles the rest
```

**Benefits:**
- Fully automated CI/CD with GitHub Actions
- Infrastructure as Code (IaC) with Terraform
- State tracked in GCS bucket
- Drift detection
- Automated Tests + Linting + Build + Deploy
- Zero-downtime deployments
- Multi-environment support (staging/production)

📖 **[Terraform Quick Start →](terraform/QUICKSTART.md)**
📖 **[GitHub Actions Workflows →](.github/workflows/README.md)**

## 🌟 Key Features

### 🛡️ Core & Security
- **Clean Architecture**: Strictly layered design (Domain, Use Cases, Adapters, Infrastructure).
- **Advanced Authentication**: JWT with RFC 7519 claims, session management, and token revocation.
- **Security Hardening**:
  - Timing attack prevention.
  - Account lockout policies.
  - Rate limiting (IP & Email based).
  - Redis-based token blacklisting.
  - Strict security headers (CSP, HSTS).
- **Audio Proxy**: Securely proxies radio streams to hide source IPs and manage bandwidth/connections.

### 💰 Ad Platform & Monetization
- **Ad Campaign Management**: Create and schedule audio ad campaigns.
- **Dynamic Insertion**: Intelligent ad injection into streams.
- **Impression Tracking**: Accurate recording of ad plays with anti-fraud measures.
- **User Profiling**: Builds anonymous listener profiles for targeted advertising.
- **Analytics**: Comprehensive reporting on impressions, clicks, and revenue.

### 📻 Station Management
- **Aggregator Integration**: Seamlessly fetches stations from Radio Browser API.
- **Search & Discovery**: High-performance search with caching.
- **Favorites**: User-specific station collections.
- **Popularity Tracking**: Real-time tracking of trending stations and genres.

### 📊 Analytics & SEO
- **SEO Optimization**: Dynamic sitemaps, semantic URLs, and rich metadata for high search visibility.
- **Behavioral Analytics**: Tracks user listening habits, search trends, and geographic distribution.
- **Admin Dashboard**: Specialized endpoints for monitoring system health and business metrics.

## 🛠️ Tech Stack
- **Language**: Go 1.24+
- **Database**: PostgreSQL 15+
- **Cache**: Redis 7+
- **Infrastructure**: Google Cloud Run, Docker, Terraform
- **CI/CD**: GitHub Actions

## 🏗️ Architecture

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

## 🔌 API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/refresh` - Refresh token
- `POST /api/v1/auth/logout` - Logout (blacklist token)
- `POST /api/v1/auth/revoke` - Revoke specific tokens/sessions

### Stations & Audio
- `GET /api/v1/stations/search` - Search stations
- `GET /api/v1/stations/popular` - Get trending stations
- `GET /api/v1/stream/:id` - **Secure Audio Proxy Endpoint**

### Ads & Monetization
- `GET /api/v1/ads/serve` - Request an ad for insertion
- `POST /api/v1/ads/impression` - Record ad impression
- `POST /api/v1/ads/click` - Record ad click

### Admin & Maintenance (Protected)
- `GET /api/v1/admin/security/metrics` - Security overview
- `GET /api/v1/admin/monitoring/health` - System health status
- `GET /api/v1/admin/maintenance/recommendations` - Database optimization tips

📖 **[View Interactive API Docs (Swagger) →](docs/SWAGGER_SETUP.md)**

## 💻 Development

### Prerequisites
- Go 1.24+
- PostgreSQL
- Redis
- Make

### Common Commands
```bash
# Run locally
make run

# Run tests
make test

# Format code
make fmt

# Run linters
make lint

# Generate Swagger docs
make swagger-generate

# Database migrations
make migrate-up
```

## 🤝 Contributing
See [CONTRIBUTING.md](docs/CONTRIBUTING.md) for detailed guidelines.

## 📄 License
MIT License - see LICENSE file for details.
