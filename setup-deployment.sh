#!/bin/bash

# AgentScan Deployment Setup Script
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

echo "🚀 AgentScan Deployment Setup"
echo "=============================="
echo ""

# Step 1: Set environment variables
log_info "Setting up environment variables..."
export JWT_SECRET="CjpUlqe5+QwpJZl2Jz1tz6gOYEUWQCY6wlQfVk982u4="

# Check if GitHub credentials are set
if [[ -z "${GITHUB_CLIENT_ID:-}" ]]; then
    log_warning "GITHUB_CLIENT_ID not set. Please get it from:"
    echo "  https://github.com/settings/applications/3129629"
    echo ""
    read -p "Enter your GitHub Client ID: " GITHUB_CLIENT_ID
    export GITHUB_CLIENT_ID
fi

if [[ -z "${GITHUB_SECRET:-}" ]]; then
    log_warning "GITHUB_SECRET not set. Please get it from:"
    echo "  https://github.com/settings/applications/3129629"
    echo ""
    read -s -p "Enter your GitHub Client Secret: " GITHUB_SECRET
    export GITHUB_SECRET
    echo ""
fi

log_success "Environment variables configured"
echo ""

# Step 2: Deploy to DigitalOcean
log_info "Deploying to DigitalOcean..."
./scripts/deploy.sh deploy

echo ""
log_success "Deployment initiated!"
echo ""

# Step 3: DNS Configuration Instructions
log_info "DNS Configuration Required"
echo "=========================="
echo ""
echo "To complete your deployment, configure these DNS records in Porkbun:"
echo ""
echo "1. Go to https://porkbun.com/account/domainsSpeedy"
echo "2. Click on 'agentscan.dev'"
echo "3. Add these DNS records:"
echo ""
echo "   Type: CNAME"
echo "   Host: @"
echo "   Answer: [YOUR_APP_URL_FROM_DEPLOYMENT]"
echo ""
echo "   Type: CNAME" 
echo "   Host: www"
echo "   Answer: [YOUR_APP_URL_FROM_DEPLOYMENT]"
echo ""
echo "   Type: CNAME"
echo "   Host: api"
echo "   Answer: [YOUR_APP_URL_FROM_DEPLOYMENT]"
echo ""

# Step 4: Get deployment info
log_info "Getting deployment information..."
sleep 5

if doctl apps list --format Name,LiveURL --no-header | grep -q "agentscan-production"; then
    APP_URL=$(doctl apps list --format Name,LiveURL --no-header | grep "agentscan-production" | awk '{print $2}')
    log_success "Deployment URL: $APP_URL"
    echo ""
    echo "Replace [YOUR_APP_URL_FROM_DEPLOYMENT] above with: $APP_URL"
    echo ""
    
    # Update GitHub OAuth URLs
    log_info "Update your GitHub OAuth app with these URLs:"
    echo "  Homepage URL: https://agentscan.dev"
    echo "  Authorization callback URL: https://agentscan.dev/auth/callback"
    echo ""
    echo "Go to: https://github.com/settings/applications/3129629"
else
    log_warning "Could not get deployment URL. Check deployment status with:"
    echo "  doctl apps list"
fi

echo ""
log_success "Setup complete! 🎉"
echo ""
echo "Next steps:"
echo "1. Configure DNS records in Porkbun (see above)"
echo "2. Update GitHub OAuth URLs (see above)"
echo "3. Wait for DNS propagation (5-30 minutes)"
echo "4. Test your deployment at https://agentscan.dev"