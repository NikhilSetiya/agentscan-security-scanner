#!/bin/bash

set -euo pipefail

# Disaster Recovery Script for AgentScan
# This script provides disaster recovery capabilities for the AgentScan platform

NAMESPACE="agentscan"
BACKUP_BUCKET=""
RESTORE_TIMESTAMP=""
DRY_RUN=false
VERBOSE=false

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
AgentScan Disaster Recovery Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
    backup                Create a full system backup
    restore               Restore from backup
    list-backups         List available backups
    validate-backup      Validate a specific backup
    test-recovery        Test recovery procedures (dry run)

Options:
    --backup-bucket BUCKET    S3 bucket for backups
    --timestamp TIMESTAMP     Backup timestamp for restore operations
    --namespace NAMESPACE     Kubernetes namespace (default: agentscan)
    --dry-run                Perform dry run without making changes
    --verbose                Enable verbose logging
    --help                   Show this help message

Examples:
    $0 backup --backup-bucket agentscan-backups
    $0 restore --backup-bucket agentscan-backups --timestamp 20240101-120000
    $0 list-backups --backup-bucket agentscan-backups
    $0 test-recovery --backup-bucket agentscan-backups --timestamp 20240101-120000

Environment Variables:
    AWS_REGION              AWS region (default: us-west-2)
    KUBECONFIG             Path to kubeconfig file
    BACKUP_ENCRYPTION_KEY   KMS key for backup encryption
EOF
}

# Function to validate prerequisites
validate_prerequisites() {
    log_info "Validating prerequisites..."
    
    # Check required tools
    local required_tools=("kubectl" "aws" "helm" "jq")
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
    
    # Check Kubernetes connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    # Check S3 bucket access
    if [[ -n "$BACKUP_BUCKET" ]]; then
        if ! aws s3 ls "s3://$BACKUP_BUCKET" &> /dev/null; then
            log_error "Cannot access S3 bucket: $BACKUP_BUCKET"
            exit 1
        fi
    fi
    
    log_success "Prerequisites validation completed"
}

# Function to create full system backup
create_backup() {
    local timestamp=$(date +%Y%m%d-%H%M%S)
    local backup_dir="/tmp/agentscan-backup-$timestamp"
    
    log_info "Creating full system backup with timestamp: $timestamp"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual backup will be created"
    fi
    
    mkdir -p "$backup_dir"
    
    # 1. Backup Kubernetes resources
    backup_kubernetes_resources "$backup_dir" "$timestamp"
    
    # 2. Backup database
    backup_database "$backup_dir" "$timestamp"
    
    # 3. Backup Redis data
    backup_redis "$backup_dir" "$timestamp"
    
    # 4. Backup application data
    backup_application_data "$backup_dir" "$timestamp"
    
    # 5. Create backup manifest
    create_backup_manifest "$backup_dir" "$timestamp"
    
    # 6. Upload to S3
    if [[ "$DRY_RUN" == "false" ]]; then
        upload_backup_to_s3 "$backup_dir" "$timestamp"
    fi
    
    # 7. Cleanup local backup
    rm -rf "$backup_dir"
    
    log_success "Backup completed successfully: $timestamp"
}

# Function to backup Kubernetes resources
backup_kubernetes_resources() {
    local backup_dir=$1
    local timestamp=$2
    
    log_info "Backing up Kubernetes resources..."
    
    local k8s_backup_dir="$backup_dir/kubernetes"
    mkdir -p "$k8s_backup_dir"
    
    # Backup all resources in the agentscan namespace
    local resources=("deployments" "services" "configmaps" "secrets" "persistentvolumeclaims" "ingresses" "serviceaccounts" "roles" "rolebindings")
    
    for resource in "${resources[@]}"; do
        log_verbose "Backing up $resource..."
        if [[ "$DRY_RUN" == "false" ]]; then
            kubectl get "$resource" -n "$NAMESPACE" -o yaml > "$k8s_backup_dir/$resource.yaml" 2>/dev/null || true
        fi
    done
    
    # Backup cluster-wide resources
    local cluster_resources=("clusterroles" "clusterrolebindings" "storageclasses" "persistentvolumes")
    
    for resource in "${cluster_resources[@]}"; do
        log_verbose "Backing up cluster resource: $resource..."
        if [[ "$DRY_RUN" == "false" ]]; then
            kubectl get "$resource" -o yaml > "$k8s_backup_dir/cluster-$resource.yaml" 2>/dev/null || true
        fi
    done
    
    # Backup Helm releases
    log_verbose "Backing up Helm releases..."
    if [[ "$DRY_RUN" == "false" ]]; then
        helm list -n "$NAMESPACE" -o json > "$k8s_backup_dir/helm-releases.json" 2>/dev/null || true
    fi
    
    log_success "Kubernetes resources backup completed"
}

# Function to backup database
backup_database() {
    local backup_dir=$1
    local timestamp=$2
    
    log_info "Backing up database..."
    
    local db_backup_dir="$backup_dir/database"
    mkdir -p "$db_backup_dir"
    
    # Get database connection details from Kubernetes secrets
    local db_host=$(kubectl get secret agentscan-secrets -n "$NAMESPACE" -o jsonpath='{.data.DB_HOST}' | base64 -d 2>/dev/null || echo "localhost")
    local db_user=$(kubectl get secret agentscan-secrets -n "$NAMESPACE" -o jsonpath='{.data.DB_USER}' | base64 -d 2>/dev/null || echo "agentscan")
    local db_name=$(kubectl get secret agentscan-secrets -n "$NAMESPACE" -o jsonpath='{.data.DB_NAME}' | base64 -d 2>/dev/null || echo "agentscan")
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Create database dump
        log_verbose "Creating database dump..."
        kubectl exec -n "$NAMESPACE" deployment/postgresql -- pg_dump -U "$db_user" -d "$db_name" --clean --if-exists > "$db_backup_dir/database-dump.sql" 2>/dev/null || {
            log_warning "Direct database dump failed, trying alternative method..."
            # Alternative: use AWS RDS snapshot if available
            create_rds_snapshot "$timestamp"
        }
        
        # Backup database schema
        kubectl exec -n "$NAMESPACE" deployment/postgresql -- pg_dump -U "$db_user" -d "$db_name" --schema-only > "$db_backup_dir/schema-only.sql" 2>/dev/null || true
    fi
    
    log_success "Database backup completed"
}

# Function to create RDS snapshot
create_rds_snapshot() {
    local timestamp=$1
    
    log_info "Creating RDS snapshot..."
    
    # Find RDS instance
    local db_instance=$(aws rds describe-db-instances --query "DBInstances[?contains(DBInstanceIdentifier, 'agentscan')].DBInstanceIdentifier" --output text)
    
    if [[ -n "$db_instance" ]]; then
        local snapshot_id="agentscan-dr-snapshot-$timestamp"
        aws rds create-db-snapshot --db-instance-identifier "$db_instance" --db-snapshot-identifier "$snapshot_id"
        log_success "RDS snapshot created: $snapshot_id"
    else
        log_warning "No RDS instance found for snapshot creation"
    fi
}

# Function to backup Redis data
backup_redis() {
    local backup_dir=$1
    local timestamp=$2
    
    log_info "Backing up Redis data..."
    
    local redis_backup_dir="$backup_dir/redis"
    mkdir -p "$redis_backup_dir"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Create Redis dump
        kubectl exec -n "$NAMESPACE" deployment/redis -- redis-cli BGSAVE
        sleep 5  # Wait for background save to complete
        
        # Copy dump file
        kubectl cp "$NAMESPACE/$(kubectl get pods -n "$NAMESPACE" -l app=redis -o jsonpath='{.items[0].metadata.name}'):/data/dump.rdb" "$redis_backup_dir/dump.rdb" 2>/dev/null || {
            log_warning "Redis dump backup failed"
        }
    fi
    
    log_success "Redis backup completed"
}

# Function to backup application data
backup_application_data() {
    local backup_dir=$1
    local timestamp=$2
    
    log_info "Backing up application data..."
    
    local app_backup_dir="$backup_dir/application"
    mkdir -p "$app_backup_dir"
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Backup persistent volume data
        local pvcs=$(kubectl get pvc -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')
        
        for pvc in $pvcs; do
            log_verbose "Backing up PVC: $pvc"
            # Create a temporary pod to access PVC data
            create_backup_pod "$pvc" "$app_backup_dir"
        done
        
        # Backup application logs
        kubectl logs -n "$NAMESPACE" --all-containers=true --prefix=true > "$app_backup_dir/application-logs.txt" 2>/dev/null || true
    fi
    
    log_success "Application data backup completed"
}

# Function to create backup pod for PVC data
create_backup_pod() {
    local pvc_name=$1
    local backup_dir=$2
    
    local pod_name="backup-pod-$(date +%s)"
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: $pod_name
  namespace: $NAMESPACE
spec:
  containers:
  - name: backup
    image: alpine:latest
    command: ["/bin/sh", "-c", "sleep 3600"]
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: $pvc_name
  restartPolicy: Never
EOF

    # Wait for pod to be ready
    kubectl wait --for=condition=Ready pod/$pod_name -n "$NAMESPACE" --timeout=60s
    
    # Create tar archive of PVC data
    kubectl exec -n "$NAMESPACE" "$pod_name" -- tar -czf /tmp/pvc-backup.tar.gz -C /data .
    kubectl cp "$NAMESPACE/$pod_name:/tmp/pvc-backup.tar.gz" "$backup_dir/$pvc_name.tar.gz"
    
    # Cleanup
    kubectl delete pod "$pod_name" -n "$NAMESPACE"
}

# Function to create backup manifest
create_backup_manifest() {
    local backup_dir=$1
    local timestamp=$2
    
    log_info "Creating backup manifest..."
    
    local manifest_file="$backup_dir/manifest.json"
    
    cat > "$manifest_file" << EOF
{
  "backup_timestamp": "$timestamp",
  "backup_type": "full",
  "environment": "$(kubectl config current-context)",
  "namespace": "$NAMESPACE",
  "created_by": "$(whoami)",
  "aws_region": "${AWS_REGION:-us-west-2}",
  "components": {
    "kubernetes": {
      "included": true,
      "resources": ["deployments", "services", "configmaps", "secrets", "pvc", "ingresses"]
    },
    "database": {
      "included": true,
      "type": "postgresql",
      "backup_method": "pg_dump"
    },
    "redis": {
      "included": true,
      "backup_method": "dump.rdb"
    },
    "application_data": {
      "included": true,
      "persistent_volumes": true,
      "logs": true
    }
  },
  "backup_size_mb": $(du -sm "$backup_dir" | cut -f1),
  "files": $(find "$backup_dir" -type f -exec basename {} \; | jq -R . | jq -s .)
}
EOF

    log_success "Backup manifest created"
}

# Function to upload backup to S3
upload_backup_to_s3() {
    local backup_dir=$1
    local timestamp=$2
    
    log_info "Uploading backup to S3..."
    
    local s3_prefix="disaster-recovery-backups/$(kubectl config current-context)/$timestamp"
    
    # Create compressed archive
    local archive_name="agentscan-backup-$timestamp.tar.gz"
    tar -czf "/tmp/$archive_name" -C "$backup_dir" .
    
    # Upload to S3
    aws s3 cp "/tmp/$archive_name" "s3://$BACKUP_BUCKET/$s3_prefix/$archive_name"
    
    # Upload individual files for easier access
    aws s3 sync "$backup_dir" "s3://$BACKUP_BUCKET/$s3_prefix/" --exclude "*.tar.gz"
    
    # Cleanup local archive
    rm -f "/tmp/$archive_name"
    
    log_success "Backup uploaded to S3: s3://$BACKUP_BUCKET/$s3_prefix/"
}

# Function to list available backups
list_backups() {
    log_info "Listing available backups..."
    
    if [[ -z "$BACKUP_BUCKET" ]]; then
        log_error "Backup bucket not specified"
        exit 1
    fi
    
    local context=$(kubectl config current-context)
    local s3_prefix="disaster-recovery-backups/$context"
    
    echo "Available backups in s3://$BACKUP_BUCKET/$s3_prefix/:"
    echo "================================================================"
    
    aws s3 ls "s3://$BACKUP_BUCKET/$s3_prefix/" --recursive | grep manifest.json | while read -r line; do
        local backup_path=$(echo "$line" | awk '{print $4}')
        local timestamp=$(echo "$backup_path" | cut -d'/' -f3)
        local size=$(echo "$line" | awk '{print $3}')
        local date=$(echo "$line" | awk '{print $1, $2}')
        
        echo "Timestamp: $timestamp"
        echo "Date: $date"
        echo "Size: $size bytes"
        echo "Path: s3://$BACKUP_BUCKET/$backup_path"
        echo "----------------------------------------------------------------"
    done
}

# Function to validate backup
validate_backup() {
    local timestamp=$1
    
    log_info "Validating backup: $timestamp"
    
    if [[ -z "$BACKUP_BUCKET" || -z "$timestamp" ]]; then
        log_error "Backup bucket and timestamp are required for validation"
        exit 1
    fi
    
    local context=$(kubectl config current-context)
    local s3_prefix="disaster-recovery-backups/$context/$timestamp"
    
    # Check if manifest exists
    if ! aws s3 ls "s3://$BACKUP_BUCKET/$s3_prefix/manifest.json" &> /dev/null; then
        log_error "Backup manifest not found: $timestamp"
        exit 1
    fi
    
    # Download and parse manifest
    local temp_manifest="/tmp/manifest-$timestamp.json"
    aws s3 cp "s3://$BACKUP_BUCKET/$s3_prefix/manifest.json" "$temp_manifest"
    
    local backup_type=$(jq -r '.backup_type' "$temp_manifest")
    local components=$(jq -r '.components | keys[]' "$temp_manifest")
    
    echo "Backup Validation Report"
    echo "========================"
    echo "Timestamp: $timestamp"
    echo "Type: $backup_type"
    echo "Components:"
    
    for component in $components; do
        local included=$(jq -r ".components.$component.included" "$temp_manifest")
        echo "  - $component: $included"
    done
    
    # Validate file integrity
    log_info "Checking file integrity..."
    local expected_files=$(jq -r '.files[]' "$temp_manifest")
    local validation_passed=true
    
    for file in $expected_files; do
        if ! aws s3 ls "s3://$BACKUP_BUCKET/$s3_prefix/$file" &> /dev/null; then
            log_warning "Missing file: $file"
            validation_passed=false
        fi
    done
    
    if [[ "$validation_passed" == "true" ]]; then
        log_success "Backup validation passed"
    else
        log_error "Backup validation failed - some files are missing"
        exit 1
    fi
    
    rm -f "$temp_manifest"
}

# Function to restore from backup
restore_from_backup() {
    local timestamp=$1
    
    log_info "Starting restore from backup: $timestamp"
    
    if [[ -z "$BACKUP_BUCKET" || -z "$timestamp" ]]; then
        log_error "Backup bucket and timestamp are required for restore"
        exit 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warning "DRY RUN MODE - No actual restore will be performed"
    fi
    
    # Validate backup first
    validate_backup "$timestamp"
    
    local context=$(kubectl config current-context)
    local s3_prefix="disaster-recovery-backups/$context/$timestamp"
    local restore_dir="/tmp/agentscan-restore-$timestamp"
    
    mkdir -p "$restore_dir"
    
    # Download backup
    log_info "Downloading backup from S3..."
    aws s3 sync "s3://$BACKUP_BUCKET/$s3_prefix/" "$restore_dir/"
    
    # Restore components
    restore_kubernetes_resources "$restore_dir"
    restore_database "$restore_dir"
    restore_redis "$restore_dir"
    restore_application_data "$restore_dir"
    
    # Cleanup
    rm -rf "$restore_dir"
    
    log_success "Restore completed successfully"
}

# Function to restore Kubernetes resources
restore_kubernetes_resources() {
    local restore_dir=$1
    
    log_info "Restoring Kubernetes resources..."
    
    local k8s_restore_dir="$restore_dir/kubernetes"
    
    if [[ ! -d "$k8s_restore_dir" ]]; then
        log_warning "Kubernetes backup directory not found, skipping..."
        return
    fi
    
    # Create namespace if it doesn't exist
    if [[ "$DRY_RUN" == "false" ]]; then
        kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    # Restore resources in order
    local restore_order=("configmaps" "secrets" "serviceaccounts" "roles" "rolebindings" "persistentvolumeclaims" "services" "deployments" "ingresses")
    
    for resource in "${restore_order[@]}"; do
        local resource_file="$k8s_restore_dir/$resource.yaml"
        if [[ -f "$resource_file" ]]; then
            log_verbose "Restoring $resource..."
            if [[ "$DRY_RUN" == "false" ]]; then
                kubectl apply -f "$resource_file" -n "$NAMESPACE" || log_warning "Failed to restore $resource"
            fi
        fi
    done
    
    log_success "Kubernetes resources restore completed"
}

# Function to restore database
restore_database() {
    local restore_dir=$1
    
    log_info "Restoring database..."
    
    local db_restore_dir="$restore_dir/database"
    local dump_file="$db_restore_dir/database-dump.sql"
    
    if [[ ! -f "$dump_file" ]]; then
        log_warning "Database dump file not found, skipping database restore..."
        return
    fi
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Wait for database pod to be ready
        kubectl wait --for=condition=Ready pod -l app=postgresql -n "$NAMESPACE" --timeout=300s
        
        # Restore database
        kubectl exec -i -n "$NAMESPACE" deployment/postgresql -- psql -U agentscan -d agentscan < "$dump_file"
    fi
    
    log_success "Database restore completed"
}

# Function to restore Redis
restore_redis() {
    local restore_dir=$1
    
    log_info "Restoring Redis data..."
    
    local redis_restore_dir="$restore_dir/redis"
    local dump_file="$redis_restore_dir/dump.rdb"
    
    if [[ ! -f "$dump_file" ]]; then
        log_warning "Redis dump file not found, skipping Redis restore..."
        return
    fi
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Wait for Redis pod to be ready
        kubectl wait --for=condition=Ready pod -l app=redis -n "$NAMESPACE" --timeout=300s
        
        # Stop Redis, copy dump file, and restart
        kubectl exec -n "$NAMESPACE" deployment/redis -- redis-cli SHUTDOWN NOSAVE || true
        kubectl cp "$dump_file" "$NAMESPACE/$(kubectl get pods -n "$NAMESPACE" -l app=redis -o jsonpath='{.items[0].metadata.name}'):/data/dump.rdb"
        kubectl rollout restart deployment/redis -n "$NAMESPACE"
        kubectl rollout status deployment/redis -n "$NAMESPACE"
    fi
    
    log_success "Redis restore completed"
}

# Function to restore application data
restore_application_data() {
    local restore_dir=$1
    
    log_info "Restoring application data..."
    
    local app_restore_dir="$restore_dir/application"
    
    if [[ ! -d "$app_restore_dir" ]]; then
        log_warning "Application data directory not found, skipping..."
        return
    fi
    
    if [[ "$DRY_RUN" == "false" ]]; then
        # Restore PVC data
        for pvc_archive in "$app_restore_dir"/*.tar.gz; do
            if [[ -f "$pvc_archive" ]]; then
                local pvc_name=$(basename "$pvc_archive" .tar.gz)
                log_verbose "Restoring PVC data: $pvc_name"
                restore_pvc_data "$pvc_name" "$pvc_archive"
            fi
        done
    fi
    
    log_success "Application data restore completed"
}

# Function to restore PVC data
restore_pvc_data() {
    local pvc_name=$1
    local archive_path=$2
    
    local pod_name="restore-pod-$(date +%s)"
    
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: $pod_name
  namespace: $NAMESPACE
spec:
  containers:
  - name: restore
    image: alpine:latest
    command: ["/bin/sh", "-c", "sleep 3600"]
    volumeMounts:
    - name: data
      mountPath: /data
  volumes:
  - name: data
    persistentVolumeClaim:
      claimName: $pvc_name
  restartPolicy: Never
EOF

    # Wait for pod to be ready
    kubectl wait --for=condition=Ready pod/$pod_name -n "$NAMESPACE" --timeout=60s
    
    # Copy and extract archive
    kubectl cp "$archive_path" "$NAMESPACE/$pod_name:/tmp/restore.tar.gz"
    kubectl exec -n "$NAMESPACE" "$pod_name" -- sh -c "cd /data && tar -xzf /tmp/restore.tar.gz"
    
    # Cleanup
    kubectl delete pod "$pod_name" -n "$NAMESPACE"
}

# Function to test recovery procedures
test_recovery() {
    log_info "Testing recovery procedures (dry run)..."
    
    DRY_RUN=true
    
    if [[ -n "$RESTORE_TIMESTAMP" ]]; then
        restore_from_backup "$RESTORE_TIMESTAMP"
    else
        log_error "Timestamp required for recovery test"
        exit 1
    fi
    
    log_success "Recovery test completed"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --backup-bucket)
            BACKUP_BUCKET="$2"
            shift 2
            ;;
        --timestamp)
            RESTORE_TIMESTAMP="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
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
        --help|-h)
            show_help
            exit 0
            ;;
        backup|restore|list-backups|validate-backup|test-recovery)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Validate prerequisites
validate_prerequisites

# Execute command
case "${COMMAND:-help}" in
    backup)
        if [[ -z "$BACKUP_BUCKET" ]]; then
            log_error "Backup bucket is required for backup operation"
            exit 1
        fi
        create_backup
        ;;
    restore)
        if [[ -z "$RESTORE_TIMESTAMP" ]]; then
            log_error "Timestamp is required for restore operation"
            exit 1
        fi
        restore_from_backup "$RESTORE_TIMESTAMP"
        ;;
    list-backups)
        list_backups
        ;;
    validate-backup)
        if [[ -z "$RESTORE_TIMESTAMP" ]]; then
            log_error "Timestamp is required for backup validation"
            exit 1
        fi
        validate_backup "$RESTORE_TIMESTAMP"
        ;;
    test-recovery)
        test_recovery
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: ${COMMAND:-}"
        show_help
        exit 1
        ;;
esac