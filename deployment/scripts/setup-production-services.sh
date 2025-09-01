#!/bin/bash

# Production Services Setup Script for AgentScan
# This script sets up real production services and generates actual credentials

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Generate secure random string
generate_secret() {
    local length=${1:-32}
    openssl rand -base64 $length | tr -d "=+/" | cut -c1-$length
}

# Setup Supabase project
setup_supabase() {
    log_info "Setting up Supabase project..."
    
    if ! command -v supabase &> /dev/null; then
        log_error "Supabase CLI not found. Install it first:"
        echo "npm install -g supabase"
        exit 1
    fi
    
    # Login to Supabase
    log_info "Please login to Supabase..."
    supabase login
    
    # Create new project
    log_info "Creating new Supabase project..."
    read -p "Enter project name (agentscan-prod): " project_name
    project_name=${project_name:-agentscan-prod}
    
    read -p "Enter organization ID: " org_id
    read -s -p "Enter database password: " db_password
    echo
    
    # Create project
    supabase projects create "$project_name" --org-id "$org_id" --db-password "$db_password"
    
    # Get project details
    project_ref=$(supabase projects list --output json | jq -r ".[] | select(.name==\"$project_name\") | .id")
    
    if [ -z "$project_ref" ]; then
        log_error "Failed to create Supabase project"
        exit 1
    fi
    
    # Generate URLs and keys
    SUPABASE_URL="https://$project_ref.supabase.co"
    
    # Get API keys
    log_info "Retrieving API keys..."
    api_keys=$(supabase projects api-keys --project-ref "$project_ref" --output json)
    SUPABASE_ANON_KEY=$(echo "$api_keys" | jq -r '.[] | select(.name=="anon") | .api_key')
    SUPABASE_SERVICE_ROLE_KEY=$(echo "$api_keys" | jq -r '.[] | select(.name=="service_role") | .api_key')
    
    log_success "Supabase project created successfully"
    echo "Project URL: $SUPABASE_URL"
    echo "Project Ref: $project_ref"
    
    # Store in environment file
    cat >> .env.production << EOF
SUPABASE_URL=$SUPABASE_URL
SUPABASE_ANON_KEY=$SUPABASE_ANON_KEY
SUPABASE_SERVICE_ROLE_KEY=$SUPABASE_SERVICE_ROLE_KEY
EOF
}

# Setup Fly.io PostgreSQL
setup_fly_postgres() {
    log_info "Setting up Fly.io PostgreSQL..."
    
    if ! command -v fly &> /dev/null; then
        log_error "Fly CLI not found. Install it first:"
        echo "curl -L https://fly.io/install.sh | sh"
        exit 1
    fi
    
    # Login to Fly.io
    log_info "Please login to Fly.io..."
    fly auth login
    
    # Create PostgreSQL app
    log_info "Creating PostgreSQL database..."
    read -p "Enter database app name (agentscan-db): " db_app_name
    db_app_name=${db_app_name:-agentscan-db}
    
    fly postgres create --name "$db_app_name" --region sjc --vm-size shared-cpu-1x --volume-size 10
    
    # Get connection string
    log_info "Retrieving database connection string..."
    DATABASE_URL=$(fly postgres connect --app "$db_app_name" --database postgres --user postgres --password)
    
    log_success "PostgreSQL database created successfully"
    
    # Store in environment file
    cat >> .env.production << EOF
DATABASE_URL=$DATABASE_URL
EOF
}

# Setup Fly.io Redis
setup_fly_redis() {
    log_info "Setting up Fly.io Redis..."
    
    # Create Redis app
    log_info "Creating Redis instance..."
    read -p "Enter Redis app name (agentscan-redis): " redis_app_name
    redis_app_name=${redis_app_name:-agentscan-redis}
    
    fly redis create --name "$redis_app_name" --region sjc
    
    # Get connection string
    log_info "Retrieving Redis connection string..."
    REDIS_URL=$(fly redis status --app "$redis_app_name" --json | jq -r '.redis_url')
    
    log_success "Redis instance created successfully"
    
    # Store in environment file
    cat >> .env.production << EOF
REDIS_URL=$REDIS_URL
EOF
}

# Generate production secrets
generate_production_secrets() {
    log_info "Generating production secrets..."
    
    # Generate JWT secret
    JWT_SECRET=$(generate_secret 64)
    
    # Generate encryption key
    ENCRYPTION_KEY=$(generate_secret 32)
    
    # Generate session secret
    SESSION_SECRET=$(generate_secret 32)
    
    log_success "Production secrets generated"
    
    # Store in environment file
    cat >> .env.production << EOF
JWT_SECRET=$JWT_SECRET
ENCRYPTION_KEY=$ENCRYPTION_KEY
SESSION_SECRET=$SESSION_SECRET
EOF
}

# Setup monitoring services
setup_monitoring() {
    log_info "Setting up monitoring services..."
    
    # Setup Sentry (optional)
    read -p "Do you want to set up Sentry for error tracking? (y/N): " setup_sentry
    if [[ $setup_sentry =~ ^[Yy]$ ]]; then
        read -p "Enter Sentry DSN: " sentry_dsn
        cat >> .env.production << EOF
SENTRY_DSN=$sentry_dsn
EOF
    fi
    
    # Setup external monitoring (optional)
    read -p "Do you want to set up external monitoring? (y/N): " setup_external_monitoring
    if [[ $setup_external_monitoring =~ ^[Yy]$ ]]; then
        read -p "Enter monitoring webhook URL: " monitoring_webhook
        cat >> .env.production << EOF
MONITORING_WEBHOOK_URL=$monitoring_webhook
EOF
    fi
}

# Create production configuration
create_production_config() {
    log_info "Creating production configuration..."
    
    # Create .env.production file
    cat > .env.production << EOF
# AgentScan Production Environment Configuration
# Generated on $(date)

# Application Configuration
APP_NAME=AgentScan
APP_VERSION=1.0.0
GO_ENV=production
PORT=8080
HOST=0.0.0.0

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout

# Server Configuration
READ_TIMEOUT=30s
WRITE_TIMEOUT=30s
IDLE_TIMEOUT=60s
SHUTDOWN_TIMEOUT=10s

# Database Configuration
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=10
DATABASE_CONN_MAX_LIFETIME=5m
DATABASE_CONN_MAX_IDLE_TIME=5m

# Redis Configuration
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10
REDIS_POOL_TIMEOUT=4s

# JWT Configuration
JWT_ALGORITHM=HS256
JWT_ISSUER=agentscan-prod
JWT_AUDIENCE=agentscan-api
JWT_TTL=24h

# Security Configuration
HTTPS_ENABLED=true
HTTPS_REDIRECT_HTTP=true

# CORS Configuration
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://agentscan.vercel.app
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Requested-With
CORS_EXPOSED_HEADERS=X-Total-Count
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=3600

# Rate Limiting Configuration
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=1000
RATE_LIMIT_WINDOW=1h
RATE_LIMIT_BURST=100

# Monitoring Configuration
METRICS_ENABLED=true
METRICS_PORT=9090
METRICS_PATH=/metrics
HEALTH_CHECK_ENABLED=true
PPROF_ENABLED=false

# Security Headers Configuration
SECURITY_HEADERS_ENABLED=true
HSTS_ENABLED=true
HSTS_MAX_AGE=31536000
HSTS_INCLUDE_SUBDOMAINS=true
HSTS_PRELOAD=true
CSP_ENABLED=true
CSP_POLICY=default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self' https:
CSP_REPORT_URI=/csp-report

# Performance Configuration
GC_PERCENT=100
MAX_PROCS=0

# Feature Flags
FEATURE_ADVANCED_SCANNING=true
FEATURE_REAL_TIME_UPDATES=true
FEATURE_EXPORT_REPORTS=true
FEATURE_API_V2=false

# Backup Configuration
BACKUP_ENABLED=true
BACKUP_SCHEDULE=0 2 * * *
BACKUP_RETENTION_DAYS=30

# Development/Debug (disabled in production)
DEBUG=false
ENABLE_PPROF=false
ENABLE_DEBUG_ENDPOINTS=false

EOF
}

# Validate production setup
validate_production_setup() {
    log_info "Validating production setup..."
    
    # Check if .env.production exists and has required variables
    if [ ! -f ".env.production" ]; then
        log_error "Production environment file not found"
        exit 1
    fi
    
    # Source the environment file
    set -a
    source .env.production
    set +a
    
    # Check required variables
    required_vars=(
        "DATABASE_URL"
        "REDIS_URL"
        "SUPABASE_URL"
        "SUPABASE_SERVICE_ROLE_KEY"
        "JWT_SECRET"
    )
    
    missing_vars=()
    for var in "${required_vars[@]}"; do
        if [ -z "${!var:-}" ]; then
            missing_vars+=("$var")
        fi
    done
    
    if [ ${#missing_vars[@]} -ne 0 ]; then
        log_error "Missing required environment variables:"
        for var in "${missing_vars[@]}"; do
            echo "  - $var"
        done
        exit 1
    fi
    
    # Test database connection
    log_info "Testing database connection..."
    if command -v psql &> /dev/null; then
        if psql "$DATABASE_URL" -c "SELECT 1;" &> /dev/null; then
            log_success "Database connection successful"
        else
            log_warning "Database connection test failed"
        fi
    else
        log_warning "psql not found, skipping database connection test"
    fi
    
    # Test Redis connection
    log_info "Testing Redis connection..."
    if command -v redis-cli &> /dev/null; then
        redis_host=$(echo "$REDIS_URL" | sed -n 's/.*@\([^:]*\):.*/\1/p')
        redis_port=$(echo "$REDIS_URL" | sed -n 's/.*:\([0-9]*\).*/\1/p')
        if redis-cli -h "$redis_host" -p "$redis_port" ping &> /dev/null; then
            log_success "Redis connection successful"
        else
            log_warning "Redis connection test failed"
        fi
    else
        log_warning "redis-cli not found, skipping Redis connection test"
    fi
    
    log_success "Production setup validation completed"
}

# Main setup function
main() {
    log_info "Starting AgentScan production services setup..."
    
    # Create production environment file
    create_production_config
    
    # Setup services
    setup_supabase
    setup_fly_postgres
    setup_fly_redis
    generate_production_secrets
    setup_monitoring
    
    # Validate setup
    validate_production_setup
    
    log_success "Production services setup completed!"
    log_info "Environment configuration saved to .env.production"
    log_warning "Please review the configuration before deploying to production"
    
    echo
    echo "Next steps:"
    echo "1. Review .env.production file"
    echo "2. Set up your domain and SSL certificates"
    echo "3. Configure monitoring and alerting"
    echo "4. Run the deployment script: ./deployment/scripts/deploy-production.sh"
}

# Show help
show_help() {
    cat << EOF
AgentScan Production Services Setup Script

This script sets up real production services for AgentScan:
- Supabase project with authentication
- Fly.io PostgreSQL database
- Fly.io Redis instance
- Production secrets generation
- Monitoring configuration

Usage: $0 [OPTIONS]

Options:
    --help          Show this help message
    --validate-only Only validate existing setup

Prerequisites:
- Supabase CLI installed and configured
- Fly.io CLI installed and configured
- OpenSSL for secret generation
- jq for JSON parsing

EOF
}

# Parse arguments
case "${1:-}" in
    --help)
        show_help
        exit 0
        ;;
    --validate-only)
        validate_production_setup
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac