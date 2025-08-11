#!/bin/bash

set -euo pipefail

# Blue-Green Deployment Script for AgentScan
# This script implements a safe blue-green deployment strategy with health checks

NAMESPACE="agentscan"
APP_NAME="agentscan-api"
TIMEOUT=300
HEALTH_CHECK_RETRIES=30
HEALTH_CHECK_INTERVAL=10

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

# Function to get current active slot
get_active_slot() {
    kubectl get service ${APP_NAME}-active -n ${NAMESPACE} -o jsonpath='{.metadata.annotations.deployment\.agentscan\.dev/active-slot}' 2>/dev/null || echo "blue"
}

# Function to get inactive slot
get_inactive_slot() {
    local active_slot=$(get_active_slot)
    if [[ "$active_slot" == "blue" ]]; then
        echo "green"
    else
        echo "blue"
    fi
}

# Function to check if deployment exists
deployment_exists() {
    local slot=$1
    kubectl get deployment ${APP_NAME}-${slot} -n ${NAMESPACE} >/dev/null 2>&1
}

# Function to wait for deployment rollout
wait_for_rollout() {
    local slot=$1
    log_info "Waiting for ${slot} deployment to complete..."
    
    if ! kubectl rollout status deployment/${APP_NAME}-${slot} -n ${NAMESPACE} --timeout=${TIMEOUT}s; then
        log_error "Deployment rollout failed for ${slot} slot"
        return 1
    fi
    
    log_success "${slot} deployment completed successfully"
    return 0
}

# Function to perform health checks
health_check() {
    local slot=$1
    local service_name="${APP_NAME}-${slot}"
    
    log_info "Performing health checks for ${slot} slot..."
    
    # Wait for pods to be ready
    local ready_pods=0
    local total_pods=0
    local retries=0
    
    while [[ $retries -lt $HEALTH_CHECK_RETRIES ]]; do
        ready_pods=$(kubectl get deployment ${APP_NAME}-${slot} -n ${NAMESPACE} -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        total_pods=$(kubectl get deployment ${APP_NAME}-${slot} -n ${NAMESPACE} -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
        
        if [[ "$ready_pods" == "$total_pods" ]] && [[ "$ready_pods" -gt 0 ]]; then
            log_success "All pods are ready for ${slot} slot (${ready_pods}/${total_pods})"
            break
        fi
        
        log_info "Waiting for pods to be ready: ${ready_pods}/${total_pods} (attempt $((retries + 1))/${HEALTH_CHECK_RETRIES})"
        sleep $HEALTH_CHECK_INTERVAL
        ((retries++))
    done
    
    if [[ $retries -eq $HEALTH_CHECK_RETRIES ]]; then
        log_error "Health check failed: Not all pods are ready for ${slot} slot"
        return 1
    fi
    
    # Perform HTTP health checks
    log_info "Performing HTTP health checks..."
    retries=0
    
    while [[ $retries -lt $HEALTH_CHECK_RETRIES ]]; do
        if kubectl exec -n ${NAMESPACE} deployment/${APP_NAME}-${slot} -- wget --quiet --tries=1 --timeout=5 --spider http://localhost:8080/health; then
            log_success "HTTP health check passed for ${slot} slot"
            break
        fi
        
        log_info "HTTP health check failed, retrying... (attempt $((retries + 1))/${HEALTH_CHECK_RETRIES})"
        sleep $HEALTH_CHECK_INTERVAL
        ((retries++))
    done
    
    if [[ $retries -eq $HEALTH_CHECK_RETRIES ]]; then
        log_error "HTTP health check failed for ${slot} slot"
        return 1
    fi
    
    # Perform readiness checks
    log_info "Performing readiness checks..."
    retries=0
    
    while [[ $retries -lt $HEALTH_CHECK_RETRIES ]]; do
        if kubectl exec -n ${NAMESPACE} deployment/${APP_NAME}-${slot} -- wget --quiet --tries=1 --timeout=5 --spider http://localhost:8080/ready; then
            log_success "Readiness check passed for ${slot} slot"
            return 0
        fi
        
        log_info "Readiness check failed, retrying... (attempt $((retries + 1))/${HEALTH_CHECK_RETRIES})"
        sleep $HEALTH_CHECK_INTERVAL
        ((retries++))
    done
    
    log_error "Readiness check failed for ${slot} slot"
    return 1
}

# Function to switch traffic
switch_traffic() {
    local new_active_slot=$1
    local old_active_slot=$2
    
    log_info "Switching traffic from ${old_active_slot} to ${new_active_slot}..."
    
    # Update active service selector
    kubectl patch service ${APP_NAME}-active -n ${NAMESPACE} -p "{\"spec\":{\"selector\":{\"version\":\"${new_active_slot}\"}}}"
    kubectl annotate service ${APP_NAME}-active -n ${NAMESPACE} deployment.agentscan.dev/active-slot=${new_active_slot} --overwrite
    
    # Update preview service selector
    kubectl patch service ${APP_NAME}-preview -n ${NAMESPACE} -p "{\"spec\":{\"selector\":{\"version\":\"${old_active_slot}\"}}}"
    kubectl annotate service ${APP_NAME}-preview -n ${NAMESPACE} deployment.agentscan.dev/preview-slot=${old_active_slot} --overwrite
    
    log_success "Traffic switched to ${new_active_slot} slot"
}

# Function to rollback deployment
rollback() {
    local failed_slot=$1
    local active_slot=$2
    
    log_warning "Rolling back deployment..."
    
    # Ensure traffic is on the stable slot
    switch_traffic $active_slot $failed_slot
    
    # Scale down the failed deployment
    kubectl scale deployment ${APP_NAME}-${failed_slot} -n ${NAMESPACE} --replicas=0
    
    log_success "Rollback completed. Traffic is on ${active_slot} slot"
}

# Function to cleanup old deployment
cleanup_old_deployment() {
    local slot=$1
    
    log_info "Scaling down ${slot} deployment..."
    kubectl scale deployment ${APP_NAME}-${slot} -n ${NAMESPACE} --replicas=0
    
    log_success "Cleanup completed for ${slot} slot"
}

# Main deployment function
deploy() {
    local image_tag=$1
    local active_slot=$(get_active_slot)
    local target_slot=$(get_inactive_slot)
    
    log_info "Starting blue-green deployment..."
    log_info "Current active slot: ${active_slot}"
    log_info "Target slot: ${target_slot}"
    log_info "Image tag: ${image_tag}"
    
    # Ensure target deployment exists
    if ! deployment_exists $target_slot; then
        log_error "Target deployment ${APP_NAME}-${target_slot} does not exist"
        exit 1
    fi
    
    # Update the target deployment with new image
    log_info "Updating ${target_slot} deployment with new image..."
    kubectl set image deployment/${APP_NAME}-${target_slot} -n ${NAMESPACE} api=agentscan/api:${image_tag}
    
    # Wait for rollout to complete
    if ! wait_for_rollout $target_slot; then
        log_error "Deployment failed"
        exit 1
    fi
    
    # Perform health checks
    if ! health_check $target_slot; then
        log_error "Health checks failed for ${target_slot} slot"
        rollback $target_slot $active_slot
        exit 1
    fi
    
    # Switch traffic to new deployment
    switch_traffic $target_slot $active_slot
    
    # Wait a bit and perform final health check
    log_info "Waiting 30 seconds before final validation..."
    sleep 30
    
    if ! health_check $target_slot; then
        log_error "Final health check failed"
        rollback $target_slot $active_slot
        exit 1
    fi
    
    # Cleanup old deployment
    cleanup_old_deployment $active_slot
    
    log_success "Blue-green deployment completed successfully!"
    log_success "Active slot is now: ${target_slot}"
}

# Function to show current status
status() {
    local active_slot=$(get_active_slot)
    local inactive_slot=$(get_inactive_slot)
    
    echo "=== AgentScan Deployment Status ==="
    echo "Active slot: ${active_slot}"
    echo "Inactive slot: ${inactive_slot}"
    echo ""
    
    echo "=== Active Deployment Status ==="
    kubectl get deployment ${APP_NAME}-${active_slot} -n ${NAMESPACE} -o wide
    echo ""
    
    echo "=== Inactive Deployment Status ==="
    kubectl get deployment ${APP_NAME}-${inactive_slot} -n ${NAMESPACE} -o wide
    echo ""
    
    echo "=== Service Status ==="
    kubectl get services -n ${NAMESPACE} -l app=${APP_NAME}
}

# Function to show help
show_help() {
    cat << EOF
AgentScan Blue-Green Deployment Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    deploy <image_tag>    Deploy new version using blue-green strategy
    status               Show current deployment status
    rollback             Rollback to previous version
    help                 Show this help message

Examples:
    $0 deploy v1.2.3
    $0 status
    $0 rollback

Environment Variables:
    NAMESPACE            Kubernetes namespace (default: agentscan)
    TIMEOUT              Deployment timeout in seconds (default: 300)
    HEALTH_CHECK_RETRIES Health check retry count (default: 30)
    HEALTH_CHECK_INTERVAL Health check interval in seconds (default: 10)
EOF
}

# Main script logic
case "${1:-help}" in
    deploy)
        if [[ -z "${2:-}" ]]; then
            log_error "Image tag is required for deployment"
            show_help
            exit 1
        fi
        deploy "$2"
        ;;
    status)
        status
        ;;
    rollback)
        active_slot=$(get_active_slot)
        inactive_slot=$(get_inactive_slot)
        
        log_info "Rolling back from ${active_slot} to ${inactive_slot}..."
        
        # Scale up the inactive deployment
        kubectl scale deployment ${APP_NAME}-${inactive_slot} -n ${NAMESPACE} --replicas=3
        
        # Wait for rollout and health checks
        if wait_for_rollout $inactive_slot && health_check $inactive_slot; then
            switch_traffic $inactive_slot $active_slot
            cleanup_old_deployment $active_slot
            log_success "Rollback completed successfully!"
        else
            log_error "Rollback failed"
            exit 1
        fi
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac