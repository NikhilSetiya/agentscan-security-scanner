#!/bin/bash

# Build the secrets migration tool

set -e

echo "🔧 Building secrets migration tool..."

# Build the migration tool
go build -o bin/migrate-secrets ./scripts/migrate-secrets.go

echo "✅ Migration tool built successfully!"
echo ""
echo "Usage:"
echo "  ./bin/migrate-secrets -supabase-url=<url> -service-role-key=<key>"
echo "  ./bin/migrate-secrets -list  # List existing secrets"
echo "  ./bin/migrate-secrets -dry-run  # Show what would be migrated"
echo ""
echo "Environment variables:"
echo "  SUPABASE_URL - Supabase project URL"
echo "  SUPABASE_SERVICE_ROLE_KEY - Supabase service role key"