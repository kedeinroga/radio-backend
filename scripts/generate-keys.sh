#!/bin/bash

# Generate RSA key pair for JWT signing

set -e

echo "Generating RSA key pair for JWT..."

# Create keys directory if it doesn't exist
mkdir -p keys

# Generate private key
openssl genrsa -out keys/jwt-private.pem 2048

# Generate public key from private key
openssl rsa -in keys/jwt-private.pem -pubout -out keys/jwt-public.pem

# Set appropriate permissions
chmod 600 keys/jwt-private.pem
chmod 644 keys/jwt-public.pem

echo "✅ JWT keys generated successfully!"
echo "   Private key: keys/jwt-private.pem"
echo "   Public key:  keys/jwt-public.pem"
echo ""
echo "⚠️  IMPORTANT: Keep the private key secure and never commit it to version control!"
