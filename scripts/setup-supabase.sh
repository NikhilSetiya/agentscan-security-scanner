#!/bin/bash

# AgentScan Supabase Setup Script
# This script helps set up Supabase for the AgentScan project

set -e

echo "🛡️  AgentScan Supabase Setup"
echo "================================"

# Check if Supabase CLI is installed
if ! command -v supabase &> /dev/null; then
    echo "❌ Supabase CLI is not installed."
    echo "Please install it first:"
    echo "npm install -g supabase"
    echo "or"
    echo "brew install supabase/tap/supabase"
    exit 1
fi

echo "✅ Supabase CLI found"

# Check if we're in the project root
if [ ! -f "PROJECT_OVERVIEW.md" ]; then
    echo "❌ Please run this script from the project root directory"
    exit 1
fi

# Initialize Supabase if not already done
if [ ! -f "supabase/config.toml" ]; then
    echo "📦 Initializing Supabase project..."
    supabase init
else
    echo "✅ Supabase project already initialized"
fi

# Start local Supabase (for development)
echo "🚀 Starting local Supabase services..."
supabase start

# Get the local project details
echo ""
echo "📋 Local Supabase Details:"
echo "================================"
supabase status

# Extract URLs and keys for environment setup
API_URL=$(supabase status | grep "API URL" | awk '{print $3}')
ANON_KEY=$(supabase status | grep "anon key" | awk '{print $3}')
SERVICE_ROLE_KEY=$(supabase status | grep "service_role key" | awk '{print $3}')

echo ""
echo "🔧 Environment Configuration:"
echo "================================"
echo "Add these to your .env files:"
echo ""
echo "Frontend (.env.development):"
echo "VITE_SUPABASE_URL=$API_URL"
echo "VITE_SUPABASE_ANON_KEY=$ANON_KEY"
echo ""
echo "Backend (.env):"
echo "SUPABASE_URL=$API_URL"
echo "SUPABASE_ANON_KEY=$ANON_KEY"
echo "SUPABASE_SERVICE_ROLE_KEY=$SERVICE_ROLE_KEY"

# Apply migrations
echo ""
echo "📊 Applying database migrations..."
if [ -f "supabase/migrations/001_initial_schema.sql" ]; then
    supabase db reset
    echo "✅ Database schema applied"
else
    echo "⚠️  No migrations found. Please create the initial schema."
fi

echo ""
echo "🎉 Supabase setup complete!"
echo ""
echo "Next steps:"
echo "1. Update your .env files with the configuration above"
echo "2. For production, create a Supabase project at https://supabase.com"
echo "3. Update production environment variables with your production Supabase details"
echo "4. Run the migrations on your production database"
echo ""
echo "Local Supabase Studio: http://localhost:54323"
echo "Local Database: postgresql://postgres:postgres@localhost:54322/postgres"