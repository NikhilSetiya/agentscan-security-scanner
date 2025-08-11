#!/bin/bash

set -euo pipefail

# Infrastructure Deployment Script for AgentScan
# This script deploys the complete AgentScan infrastructure using Terraform and Kubernetes

ENVIRONMENT="production"
AWS_REGION="us-west-2"
TERRAFORM_DIR="terraform"
K8S_DIR="k8s"
DRY_RUN=false
VERBOSE=false
SKIP_TERRAFORM=false
SKIP_K8S=false

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

log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Function to show help
show_help() {
    cat << EOF
AgentScan Infrastructure Deployment Script

Usage: $0 [OPTIONS]

Options:
    --environment ENV        Environment to deploy (default: production)
    --aws-region REGION      AWS region (default: us-west-2)
    --terraform-dir DIR      Terraform directory (default: terraform)
    --k8s-dir DIR           Kubernetes manifests directory (default: k8s)
    --dry-run               Perform dry run without making changes
    --verbose               Enable verbose logging
    --skip-terraform        Skip Terraform deployment
    --skip-k8s             Skip Kubernetes deployment
    --help                 Show this help message

Examples:
    $0 --environment staging --aws-region us-east-1
    $0 --dry-run --verbose
    $0 --skip-terraform

Environment Variables:
    AWS_PROFILE            AWS profile to use
    TF_VAR_*              Terraform variables
EOF
}

# Function to validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    # Check required tools
    local required_tools=("terraform" "kubectl" "helm" "aws" "jq")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Check AWS credentials
    if ! aws sts get-caller-identity &> /dev/null; then
        log_error "AWS credentials not configured or invalid"
        exit 1
    fi
    
    # Check Terraform version
    local tf_version=$(terraform version -json | jq -r '.terraform_version')
    log_verbose "Terraform version: $tf_version"
    
    # Check if Terraform directory exists
    if [[ ! -d "$TERRAFORM_DIR" ]]; then
        log_error "Terraform directory not found: $TERRAFORM_DIR"
        exit 1
    fi
    
    # Check if Kubernetes directory exists
    if [[ ! -d "$K8S_DIR" ]]; then
        log_error "Kubernetes directory not found: $K8S_DIR"
        exit 1
    fi
    
    log_success "Prerequisites validation completed"
}

# Function to deploy Terraform infrastructure
deploy_terraform() {
    log_info "Deploying Terraform infrastructure..."
    
    cd "$TERRAFORM_DIR"
    
    # Initialize Terraform
    log_verbose "Initializing Terraform..."
    if [[ "$DRY_RUN" == "false" ]]; then
        terraform init -upgrade
    else
        log_warning "DRY RUN: Would initialize Terraform"
    fi
    
    # Validate Terraform configuration
    log_verbose "Validating Terraform configuration..."
    terraform validate
    
    # Plan Terraform deployment
    log_info "Planning Terraform deployment..."
    local plan_file="tfplan-$(date +%Y%m%d-%H%M%S)"
    
    terraform plan \
        -var="environment=$ENVIRONMENT" \
        -var="aws_region=$AWS_REGION" \
        -out="$plan_file"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Apply Terraform plan
        log_info "Applying Terraform plan..."
        terraform apply "$plan_file"
        
        # Clean up plan file
        rm -f "$plan_file"
        
        # Output important values
        log_info "Terraform deployment completed. Important outputs:"
        terraform output -json | jq -r 'to_entries[] | "\(.key): \(.value.value)"' | head -10
    else
        log_warning "DRY RUN: Would apply Terraform plan"
        rm -f "$plan_file"
    fi
    
    cd - > /dev/null
    log_success "Terraform infrastructure deployment completed"
}

# Function to configure kubectl
configure_kubectl() {
    log_info "Configuring kubectl..."
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Get cluster name from Terraform output
        cd "$TERRAFORM_DIR"
        local cluster_name=$(terraform output -raw cluster_name 2>/dev/null || echo "agentscan-$ENVIRONMENT")
        cd - > /dev/null
        
        # Update kubeconfig
        aws eks update-kubeconfig --region "$AWS_REGION" --name "$cluster_name"
        
        # Verify connection
        kubectl cluster-info
        
        log_success "kubectl configured successfully"
    else
        log_warning "DRY RUN: Would configure kubectl"
    fi
}

# Function to deploy Kubernetes resources
deploy_kubernetes() {
    log_info "Deploying Kubernetes resources..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN: Would deploy Kubernetes resources"
        return
    fi
    
    # Deploy in order
    local deployment_order=(
        "namespace.yaml"
        "configmap.yaml"
        "secrets.yaml"
        "postgresql.yaml"
        "redis.yaml"
        "api-deployment.yaml"
        "orchestrator-deployment.yaml"
        "ingress.yaml"
        "monitoring/prometheus-rules.yaml"
    )
    
    for manifest in "${deployment_order[@]}"; do
        local manifest_path="$K8S_DIR/$manifest"
        
        if [[ -f "$manifest_path" ]]; then
            log_verbose "Deploying $manifest..."
            kubectl apply -f "$manifest_path"
        else
            log_warning "Manifest not found: $manifest_path"
        fi
    done
    
    # Wait for deployments to be ready
    log_info "Waiting for deployments to be ready..."
    kubectl wait --for=condition=available --timeout=600s deployment --all -n agentscan
    
    log_success "Kubernetes resources deployment completed"
}

# Function to deploy blue-green infrastructure
deploy_blue_green() {
    log_info "Deploying blue-green infrastructure..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN: Would deploy blue-green infrastructure"
        return
    fi
    
    # Deploy blue-green manifests
    local bg_manifests=(
        "blue-green/blue-deployment.yaml"
        "blue-green/green-deployment.yaml"
        "blue-green/services.yaml"
    )
    
    for manifest in "${bg_manifests[@]}"; do
        local manifest_path="$K8S_DIR/$manifest"
        
        if [[ -f "$manifest_path" ]]; then
            log_verbose "Deploying $manifest..."
            kubectl apply -f "$manifest_path"
        fi
    done
    
    log_success "Blue-green infrastructure deployment completed"
}

# Function to verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN: Would verify deployment"
        return
    fi
    
    # Check pod status
    log_verbose "Checking pod status..."
    kubectl get pods -n agentscan
    
    # Check service status
    log_verbose "Checking service status..."
    kubectl get services -n agentscan
    
    # Check ingress status
    log_verbose "Checking ingress status..."
    kubectl get ingress -n agentscan
    
    # Perform health checks
    log_info "Performing health checks..."
    
    # Wait for API to be ready
    local api_ready=false
    local retries=0
    local max_retries=30
    
    while [[ $retries -lt $max_retries ]]; do
        if kubectl exec -n agentscan deployment/agentscan-api -- wget --quiet --tries=1 --timeout=5 --spider http://localhost:8080/health; then
            api_ready=true
            break
        fi
        
        log_verbose "API health check failed, retrying... ($((retries + 1))/$max_retries)"
        sleep 10
        ((retries++))
    done
    
    if [[ "$api_ready" == "true" ]]; then
        log_success "API health check passed"
    else
        log_error "API health check failed after $max_retries attempts"
        return 1
    fi
    
    # Check orchestrator health
    if kubectl exec -n agentscan deployment/agentscan-orchestrator -- wget --quiet --tries=1 --timeout=5 --spider http://localhost:8081/health; then
        log_success "Orchestrator health check passed"
    else
        log_warning "Orchestrator health check failed"
    fi
    
    log_success "Deployment verification completed"
}

# Function to setup monitoring
setup_monitoring() {
    log_info "Setting up monitoring and alerting..."
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN: Would setup monitoring"
        return
    fi
    
    # The monitoring stack is deployed via Terraform Helm releases
    # Just verify it's working
    
    # Check if monitoring namespace exists
    if kubectl get namespace monitoring &> /dev/null; then
        log_verbose "Monitoring namespace exists"
        
        # Check Prometheus
        if kubectl get pods -n monitoring -l app.kubernetes.io/name=prometheus &> /dev/null; then
            log_success "Prometheus is deployed"
        else
            log_warning "Prometheus not found"
        fi
        
        # Check Grafana
        if kubectl get pods -n monitoring -l app.kubernetes.io/name=grafana &> /dev/null; then
            log_success "Grafana is deployed"
        else
            log_warning "Grafana not found"
        fi
        
        # Check Alertmanager
        if kubectl get pods -n monitoring -l app.kubernetes.io/name=alertmanager &> /dev/null; then
            log_success "Alertmanager is deployed"
        else
            log_warning "Alertmanager not found"
        fi
    else
        log_warning "Monitoring namespace not found"
    fi
    
    log_success "Monitoring setup completed"
}

# Function to display deployment summary
display_summary() {
    log_info "Deployment Summary"
    echo "=================="
    echo "Environment: $ENVIRONMENT"
    echo "AWS Region: $AWS_REGION"
    echo "Dry Run: $DRY_RUN"
    echo ""
    
    if [[ "$DRY_RUN" == "false" ]]; then
        echo "Deployed Resources:"
        echo "- EKS Cluster: $(cd "$TERRAFORM_DIR" && terraform output -raw cluster_name 2>/dev/null || echo 'N/A')"
        echo "- RDS Instance: $(cd "$TERRAFORM_DIR" && terraform output -raw db_instance_id 2>/dev/null || echo 'N/A')"
        echo "- Redis Cluster: $(cd "$TERRAFORM_DIR" && terraform output -raw redis_replication_group_id 2>/dev/null || echo 'N/A')"
        echo ""
        
        echo "Application Endpoints:"
        local api_endpoint=$(kubectl get ingress agentscan-ingress -n agentscan -o jsonpath='{.spec.rules[0].host}' 2>/dev/null || echo 'N/A')
        echo "- API: https://$api_endpoint"
        echo "- Grafana: https://grafana.agentscan.dev"
        echo "- Jaeger: https://jaeger.agentscan.dev"
        echo ""
        
        echo "Next Steps:"
        echo "1. Configure DNS records for ingress hosts"
        echo "2. Update secrets with actual values"
        echo "3. Configure monitoring alerts"
        echo "4. Run integration tests"
        echo "5. Setup backup schedules"
    else
        echo "This was a dry run. No resources were actually deployed."
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --aws-region)
            AWS_REGION="$2"
            shift 2
            ;;
        --terraform-dir)
            TERRAFORM_DIR="$2"
            shift 2
            ;;
        --k8s-dir)
            K8S_DIR="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --skip-terraform)
            SKIP_TERRAFORM=true
            shift
            ;;
        --skip-k8s)
            SKIP_K8S=true
            shift
            ;;
        --help|-h)
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

# Main deployment flow
main() {
    log_info "Starting AgentScan infrastructure deployment..."
    log_info "Environment: $ENVIRONMENT"
    log_info "AWS Region: $AWS_REGION"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "Running in DRY RUN mode - no changes will be made"
    fi
    
    # Validate prerequisites
    validate_prerequisites
    
    # Deploy Terraform infrastructure
    if [[ "$SKIP_TERRAFORM" == "false" ]]; then
        deploy_terraform
        configure_kubectl
    else
        log_warning "Skipping Terraform deployment"
    fi
    
    # Deploy Kubernetes resources
    if [[ "$SKIP_K8S" == "false" ]]; then
        deploy_kubernetes
        deploy_blue_green
        verify_deployment
        setup_monitoring
    else
        log_warning "Skipping Kubernetes deployment"
    fi
    
    # Display summary
    display_summary
    
    log_success "AgentScan infrastructure deployment completed successfully!"
}

# Run main function
main "$@"