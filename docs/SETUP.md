# Quick Setup Guide

## Prerequisites

- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- OpenSSL (for key generation)

## Setup Steps

### 1. Install Dependencies

```bash
go mod download
```

### 2. Generate JWT Keys

```bash
make generate-keys
```

This will create:
- `keys/jwt-private.pem` - Private key for signing tokens
- `keys/jwt-public.pem` - Public key for verifying tokens

### 3. Set Up Database

**Option A: Using Docker**

```bash
# PostgreSQL
docker run -d --name radio-postgres \
  -e POSTGRES_USER=radio \
  -e POSTGRES_PASSWORD=radio \
  -e POSTGRES_DB=radio_backend \
  -p 5432:5432 \
  postgres:15

# Redis
docker run -d --name radio-redis \
  -p 6379:6379 \
  redis:7
```

**Option B: Local Installation**

Install PostgreSQL and Redis locally, then create the database:

```sql
CREATE DATABASE radio_backend;
CREATE USER radio WITH PASSWORD 'radio';
GRANT ALL PRIVILEGES ON DATABASE radio_backend TO radio;
```

### 4. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and update:

```env
DATABASE_URL=postgres://radio:radio@localhost:5432/radio_backend?sslmode=disable
REDIS_URL=redis://localhost:6379/0
JWT_PRIVATE_KEY_PATH=./keys/jwt-private.pem
JWT_PUBLIC_KEY_PATH=./keys/jwt-public.pem
```

### 5. Run Migrations

```bash
make migrate-up
```

### 6. Start the Server

```bash
make run
```

The server will start on `http://localhost:8080`

## Verify Installation

### Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "service": "radio-backend"
}
```

### Register a User

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "Test1234"
  }'
```

Save the `access_token` from the response.

### Get Popular Stations

```bash
curl http://localhost:8080/api/v1/stations/popular?limit=5
```

### Search Stations

```bash
curl "http://localhost:8080/api/v1/stations/search?q=jazz&limit=5"
```

### Get Current User (Authenticated)

```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

## Development Commands

```bash
make help          # Show all available commands
make run           # Run the server
make dev           # Run with live reload (requires air)
make test          # Run tests with coverage
make lint          # Run linters
make fmt           # Format code
make security      # Run security scan
make build         # Build binary
make docker-build  # Build Docker image
make docker-run    # Run Docker container
```

## Troubleshooting

### Database Connection Error

- Verify PostgreSQL is running: `pg_isready`
- Check DATABASE_URL in `.env`
- Ensure database exists and user has permissions

### Redis Connection Error

- Verify Redis is running: `redis-cli ping`
- Check REDIS_URL in `.env`

### JWT Key Error

- Run `make generate-keys` to generate keys
- Verify keys exist in `keys/` directory
- Check file permissions (private key should be 600)

### Port Already in Use

- Change SERVER_PORT in `.env`
- Or stop the process using port 8080: `lsof -ti:8080 | xargs kill`

## Next Steps

- Read [README.md](../README.md) for full documentation
- Check [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines
- Review [walkthrough.md](walkthrough.md) for architecture details
