#!/bin/bash

# AgentScan Production Deployment Script
# This script automates the deployment process to DigitalOcean App Platform

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
APP_NAME="agentscan-production"
REGION="nyc1"
GITHUB_REPO="NikhilSetiya/agentscan-security-scanner"
BRANCH="main"

# Functions
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

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if doctl is installed
    if ! command -v doctl &> /dev/null; then
        log_error "doctl CLI is not installed. Please install it first:"
        echo "  curl -sL https://github.com/digitalocean/doctl/releases/download/v1.94.0/doctl-1.94.0-linux-amd64.tar.gz | tar -xzv"
        echo "  sudo mv doctl /usr/local/bin"
        exit 1
    fi
    
    # Check if authenticated
    if ! doctl account get &> /dev/null; then
        log_error "Not authenticated with DigitalOcean. Please run:"
        echo "  doctl auth init"
        exit 1
    fi
    
    # Check if app spec exists
    if [[ ! -f ".do/app.yaml" ]]; then
        log_error "App specification file not found at .do/app.yaml"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

validate_environment() {
    log_info "Validating environment variables..."
    
    required_vars=(
        "JWT_SECRET"
        "GITHUB_CLIENT_ID"
        "GITHUB_SECRET"
    )
    
    missing_vars=()
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_error "Missing required environment variables:"
        printf '  %s\n' "${missing_vars[@]}"
        echo ""
        echo "Please set these variables and try again:"
        echo "  export JWT_SECRET=\"your-jwt-secret\""
        echo "  export GITHUB_CLIENT_ID=\"your-github-client-id\""
        echo "  export GITHUB_SECRET=\"your-github-secret\""
        exit 1
    fi
    
    log_success "Environment validation passed"
}

create_or_update_app() {
    log_info "Checking if app exists..."
    
    if doctl apps get "$APP_NAME" &> /dev/null; then
        log_info "App exists, updating..."
        
        # Update the app
        doctl apps update "$APP_NAME" --spec .do/app.yaml --wait
        
        log_success "App updated successfully"
    else
        log_info "App does not exist, creating..."
        
        # Create the app
        doctl apps create --spec .do/app.yaml --wait
        
        log_success "App created successfully"
    fi
}

setup_secrets() {
    log_info "Setting up secrets..."
    
    # Get app ID
    APP_ID=$(doctl apps list --format ID,Spec.Name --no-header | grep "$APP_NAME" | awk '{print $1}')
    
    if [[ -z "$APP_ID" ]]; then
        log_error "Could not find app ID for $APP_NAME"
        exit 1
    fi
    
    # Set secrets (Note: doctl doesn't support setting secrets directly via CLI yet)
    log_warning "Please set the following secrets manually in the DigitalOcean dashboard:"
    echo "  - JWT_SECRET"
    echo "  - GITHUB_CLIENT_ID"
    echo "  - GITHUB_SECRET"
    echo ""
    echo "App URL: https://cloud.digitalocean.com/apps/$APP_ID/settings"
}

setup_domains() {
    log_info "Setting up custom domains..."
    
    # Note: Domain setup requires manual DNS configuration
    log_warning "Please configure the following DNS records:"
    echo "  agentscan.dev -> CNAME to your app domain"
    echo "  api.agentscan.dev -> CNAME to your app domain"
    echo "  app.agentscan.dev -> CNAME to your app domain"
    echo "  docs.agentscan.dev -> CNAME to your app domain"
}

run_health_checks() {
    log_info "Running health checks..."
    
    # Get app info
    APP_INFO=$(doctl apps get "$APP_NAME" --format LiveURL --no-header)
    API_URL="$APP_INFO"
    
    if [[ -z "$API_URL" ]]; then
        log_error "Could not get app URL"
        exit 1
    fi
    
    log_info "Waiting for services to be ready..."
    sleep 30
    
    # Check API health
    log_info "Checking API health at $API_URL/health"
    if curl -f -s "$API_URL/health" > /dev/null; then
        log_success "API health check passed"
    else
        log_warning "API health check failed - this might be normal during initial deployment"
    fi
    
    # Check web frontend
    log_info "Checking web frontend at $API_URL"
    if curl -f -s "$API_URL" > /dev/null; then
        log_success "Web frontend health check passed"
    else
        log_warning "Web frontend health check failed - this might be normal during initial deployment"
    fi
}

setup_monitoring() {
    log_info "Setting up monitoring and alerts..."
    
    log_warning "Please configure the following monitoring:"
    echo "  1. Set up Uptime monitoring for your endpoints"
    echo "  2. Configure Slack/email notifications for alerts"
    echo "  3. Set up log aggregation and monitoring"
    echo "  4. Configure backup schedules for databases"
}

print_deployment_info() {
    log_success "Deployment completed!"
    echo ""
    echo "=== Deployment Information ==="
    
    # Get app info
    APP_INFO=$(doctl apps get "$APP_NAME" --format LiveURL,CreatedAt,UpdatedAt --no-header)
    echo "App URL: $APP_INFO"
    
    echo ""
    echo "=== Next Steps ==="
    echo "1. Configure DNS records for custom domains"
    echo "2. Set up environment secrets in the DigitalOcean dashboard"
    echo "3. Configure monitoring and alerting"
    echo "4. Test the deployment thoroughly"
    echo "5. Set up CI/CD pipeline for automated deployments"
    echo ""
    echo "=== Useful Commands ==="
    echo "View app status: doctl apps get $APP_NAME"
    echo "View logs: doctl apps logs $APP_NAME --type=run"
    echo "View deployments: doctl apps list-deployments $APP_NAME"
    echo ""
}

# Main execution
main() {
    log_info "Starting AgentScan deployment to DigitalOcean App Platform..."
    
    check_prerequisites
    validate_environment
    create_or_update_app
    setup_secrets
    setup_domains
    run_health_checks
    setup_monitoring
    print_deployment_info
    
    log_success "Deployment script completed successfully!"
}

# Handle script arguments
case "${1:-}" in
    "check")
        check_prerequisites
        validate_environment
        ;;
    "deploy")
        main
        ;;
    "health")
        run_health_checks
        ;;
    *)
        echo "Usage: $0 {check|deploy|health}"
        echo ""
        echo "Commands:"
        echo "  check   - Check prerequisites and environment"
        echo "  deploy  - Deploy the application"
        echo "  health  - Run health checks"
        exit 1
        ;;
esac