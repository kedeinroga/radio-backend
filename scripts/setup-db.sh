#!/bin/bash

# Database setup script for development

set -e

echo "🔧 Setting up development database..."

# Check if PostgreSQL is running
if ! command -v psql &> /dev/null; then
    echo "❌ PostgreSQL is not installed. Please install it first:"
    echo "   brew install postgresql@15"
    exit 1
fi

# Check if Docker is available
if command -v docker &> /dev/null; then
    echo "🐳 Docker detected. Starting PostgreSQL and Redis containers..."
    
    # Start PostgreSQL
    docker run -d --name radio-postgres \
        -e POSTGRES_USER=radio \
        -e POSTGRES_PASSWORD=radio \
        -e POSTGRES_DB=radio_backend \
        -p 5432:5432 \
        postgres:15 2>/dev/null || echo "PostgreSQL container already exists"
    
    # Start Redis
    docker run -d --name radio-redis \
        -p 6379:6379 \
        redis:7 2>/dev/null || echo "Redis container already exists"
    
    echo "✅ Containers started"
    echo ""
    echo "📝 Update your .env file with:"
    echo "DATABASE_URL=postgres://radio:radio@localhost:5432/radio_backend?sslmode=disable"
    echo "REDIS_URL=redis://localhost:6379/0"
else
    echo "⚠️  Docker not found. Please set up PostgreSQL and Redis manually."
    echo ""
    echo "PostgreSQL setup:"
    echo "  createdb radio_backend"
    echo "  createuser radio -P  # Set password: radio"
    echo ""
    echo "Redis setup:"
    echo "  brew install redis"
    echo "  brew services start redis"
fi

echo ""
echo "✅ Setup complete!"
