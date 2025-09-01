#!/bin/bash

# AgentScan Production Deployment Script
# This script deploys the AgentScan application to production environments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployment"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    local missing_tools=()
    
    # Check required tools
    if ! command_exists "fly"; then
        missing_tools+=("fly (Fly.io CLI)")
    fi
    
    if ! command_exists "vercel"; then
        missing_tools+=("vercel (Vercel CLI)")
    fi
    
    if ! command_exists "docker"; then
        missing_tools+=("docker")
    fi
    
    if ! command_exists "git"; then
        missing_tools+=("git")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools:"
        for tool in "${missing_tools[@]}"; do
            echo "  - $tool"
        done
        exit 1
    fi
    
    # Check if we're in the right directory
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        log_error "Not in AgentScan project root directory"
        exit 1
    fi
    
    # Check if we're on the main branch
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ]; then
        log_warning "Not on main branch (current: $current_branch)"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    # Check for uncommitted changes
    if ! git diff-index --quiet HEAD --; then
        log_warning "Uncommitted changes detected"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    log_success "Prerequisites validated"
}

# Validate environment variables
validate_environment() {
    log_info "Validating environment variables..."
    
    local required_vars=(
        "DATABASE_URL"
        "REDIS_URL"
        "SUPABASE_URL"
        "SUPABASE_SERVICE_ROLE_KEY"
        "JWT_SECRET"
    )
    
    local missing_vars=()
    
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
        echo
        echo "Please set these variables or use the --set-secrets flag to set them interactively."
        exit 1
    fi
    
    # Validate JWT secret length
    if [ ${#JWT_SECRET} -lt 32 ]; then
        log_error "JWT_SECRET must be at least 32 characters long"
        exit 1
    fi
    
    log_success "Environment variables validated"
}

# Set Fly.io secrets
set_fly_secrets() {
    log_info "Setting Fly.io secrets..."
    
    fly secrets set \
        DATABASE_URL="$DATABASE_URL" \
        REDIS_URL="$REDIS_URL" \
        SUPABASE_URL="$SUPABASE_URL" \
        SUPABASE_SERVICE_ROLE_KEY="$SUPABASE_SERVICE_ROLE_KEY" \
        JWT_SECRET="$JWT_SECRET" \
        --app agentscan-prod
    
    log_success "Fly.io secrets set"
}

# Deploy backend to Fly.io
deploy_backend() {
    log_info "Deploying backend to Fly.io..."
    
    cd "$PROJECT_ROOT"
    
    # Check if app exists
    if ! fly apps list | grep -q "agentscan-prod"; then
        log_info "Creating Fly.io app..."
        fly apps create agentscan-prod --org personal
    fi
    
    # Deploy the application
    fly deploy --config "$DEPLOYMENT_DIR/production/fly.toml" --dockerfile "$DEPLOYMENT_DIR/production/Dockerfile"
    
    # Wait for deployment to be ready
    log_info "Waiting for deployment to be ready..."
    fly status --app agentscan-prod
    
    # Run health check
    log_info "Running health check..."
    local health_url="https://agentscan-prod.fly.dev/health"
    local max_attempts=30
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$health_url" > /dev/null; then
            log_success "Backend health check passed"
            break
        fi
        
        log_info "Health check attempt $attempt/$max_attempts failed, retrying in 10s..."
        sleep 10
        ((attempt++))
    done
    
    if [ $attempt -gt $max_attempts ]; then
        log_error "Backend health check failed after $max_attempts attempts"
        exit 1
    fi
    
    log_success "Backend deployed successfully to Fly.io"
}

# Deploy frontend to Vercel
deploy_frontend() {
    log_info "Deploying frontend to Vercel..."
    
    cd "$PROJECT_ROOT/web/frontend"
    
    # Install dependencies
    log_info "Installing frontend dependencies..."
    npm ci
    
    # Build the application
    log_info "Building frontend application..."
    npm run build
    
    # Deploy to Vercel
    log_info "Deploying to Vercel..."
    vercel --prod --yes
    
    log_success "Frontend deployed successfully to Vercel"
}

# Setup monitoring and alerting
setup_monitoring() {
    log_info "Setting up monitoring and alerting..."
    
    # Create monitoring configuration
    cat > "$DEPLOYMENT_DIR/production/monitoring.yml" << EOF
# Monitoring configuration for AgentScan production
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
      - '--web.enable-lifecycle'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-storage:/var/lib/grafana

volumes:
  grafana-storage:
EOF
    
    # Create Prometheus configuration
    cat > "$DEPLOYMENT_DIR/production/prometheus.yml" << EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'agentscan-api'
    static_configs:
      - targets: ['agentscan-prod.fly.dev:9090']
    metrics_path: '/metrics'
    scrape_interval: 30s

  - job_name: 'fly-metrics'
    static_configs:
      - targets: ['fly.io:443']
    scheme: https
    metrics_path: '/v1/apps/agentscan-prod/metrics'
EOF
    
    log_success "Monitoring configuration created"
}

# Run smoke tests
run_smoke_tests() {
    log_info "Running smoke tests..."
    
    local api_url="https://agentscan-prod.fly.dev"
    local frontend_url="https://agentscan.vercel.app"
    
    # Test API endpoints
    log_info "Testing API endpoints..."
    
    # Health check
    if ! curl -f -s "$api_url/health" > /dev/null; then
        log_error "API health check failed"
        return 1
    fi
    
    # Readiness check
    if ! curl -f -s "$api_url/ready" > /dev/null; then
        log_error "API readiness check failed"
        return 1
    fi
    
    # Metrics endpoint
    if ! curl -f -s "$api_url/metrics" > /dev/null; then
        log_error "API metrics endpoint failed"
        return 1
    fi
    
    # Test frontend
    log_info "Testing frontend..."
    
    if ! curl -f -s "$frontend_url" > /dev/null; then
        log_error "Frontend health check failed"
        return 1
    fi
    
    log_success "Smoke tests passed"
}

# Rollback deployment
rollback_deployment() {
    log_warning "Rolling back deployment..."
    
    # Rollback Fly.io deployment
    fly releases list --app agentscan-prod
    read -p "Enter release version to rollback to: " rollback_version
    
    if [ -n "$rollback_version" ]; then
        fly releases rollback "$rollback_version" --app agentscan-prod
        log_success "Backend rolled back to version $rollback_version"
    fi
    
    # Rollback Vercel deployment
    cd "$PROJECT_ROOT/web/frontend"
    vercel rollback
    
    log_success "Deployment rolled back"
}

# Main deployment function
deploy() {
    local skip_validation=false
    local set_secrets=false
    local skip_frontend=false
    local skip_backend=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-validation)
                skip_validation=true
                shift
                ;;
            --set-secrets)
                set_secrets=true
                shift
                ;;
            --skip-frontend)
                skip_frontend=true
                shift
                ;;
            --skip-backend)
                skip_backend=true
                shift
                ;;
            --rollback)
                rollback_deployment
                exit 0
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    log_info "Starting AgentScan production deployment..."
    
    # Validation
    if [ "$skip_validation" = false ]; then
        validate_prerequisites
        validate_environment
    fi
    
    # Set secrets if requested
    if [ "$set_secrets" = true ]; then
        set_fly_secrets
    fi
    
    # Deploy backend
    if [ "$skip_backend" = false ]; then
        deploy_backend
    fi
    
    # Deploy frontend
    if [ "$skip_frontend" = false ]; then
        deploy_frontend
    fi
    
    # Setup monitoring
    setup_monitoring
    
    # Run smoke tests
    run_smoke_tests
    
    log_success "Production deployment completed successfully!"
    log_info "Backend URL: https://agentscan-prod.fly.dev"
    log_info "Frontend URL: https://agentscan.vercel.app"
    log_info "Metrics URL: https://agentscan-prod.fly.dev/metrics"
}

# Show help
show_help() {
    cat << EOF
AgentScan Production Deployment Script

Usage: $0 [OPTIONS]

Options:
    --skip-validation    Skip prerequisite and environment validation
    --set-secrets       Set Fly.io secrets interactively
    --skip-frontend     Skip frontend deployment
    --skip-backend      Skip backend deployment
    --rollback          Rollback to previous deployment
    -h, --help          Show this help message

Environment Variables:
    DATABASE_URL                PostgreSQL connection string
    REDIS_URL                  Redis connection string
    SUPABASE_URL               Supabase project URL
    SUPABASE_SERVICE_ROLE_KEY  Supabase service role key
    JWT_SECRET                 JWT signing secret (32+ characters)

Examples:
    $0                         # Full deployment with validation
    $0 --skip-validation       # Deploy without validation
    $0 --set-secrets          # Set secrets and deploy
    $0 --skip-frontend        # Deploy backend only
    $0 --rollback             # Rollback deployment

EOF
}

# Script entry point
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    deploy "$@"
fi