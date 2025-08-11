# AgentScan Deployment and Infrastructure

This document provides comprehensive guidance for deploying and managing the AgentScan security scanning platform infrastructure.

## Overview

The AgentScan infrastructure is designed for production-scale deployment with the following key features:

- **Kubernetes-native**: Runs on Amazon EKS with auto-scaling capabilities
- **Blue-Green Deployments**: Zero-downtime deployments with automated rollback
- **Infrastructure as Code**: Complete infrastructure defined in Terraform
- **Comprehensive Monitoring**: Prometheus, Grafana, and Jaeger for observability
- **Automated Backups**: Multi-tier backup strategy with disaster recovery
- **Security Hardened**: Network policies, encryption, and security best practices

## Architecture Components

### Core Infrastructure
- **Amazon EKS**: Kubernetes cluster with managed node groups
- **Amazon RDS**: PostgreSQL database with read replicas
- **Amazon ElastiCache**: Redis cluster for caching and job queues
- **Amazon S3**: Object storage for backups and artifacts
- **AWS Secrets Manager**: Secure credential storage

### Networking
- **VPC**: Dedicated VPC with public/private subnets
- **Application Load Balancer**: HTTPS termination and routing
- **VPC Endpoints**: Secure access to AWS services
- **Network Policies**: Kubernetes network segmentation

### Monitoring & Observability
- **Prometheus**: Metrics collection and alerting
- **Grafana**: Visualization and dashboards
- **Jaeger**: Distributed tracing
- **CloudWatch**: AWS service monitoring
- **AlertManager**: Alert routing and notification

## Prerequisites

### Required Tools
```bash
# Install required tools
brew install terraform kubectl helm awscli jq

# Verify installations
terraform version  # >= 1.0
kubectl version    # >= 1.24
helm version       # >= 3.8
aws --version      # >= 2.0
```

### AWS Configuration
```bash
# Configure AWS credentials
aws configure

# Verify access
aws sts get-caller-identity
```

### Domain and Certificates
- Register domain (e.g., `agentscan.dev`)
- Create ACM certificate for `*.agentscan.dev`
- Update `terraform/variables.tf` with your domain and certificate ARN

## Deployment Guide

### 1. Infrastructure Deployment

#### Quick Start
```bash
# Deploy complete infrastructure
./scripts/deploy-infrastructure.sh --environment production --aws-region us-west-2

# Deploy with dry run first
./scripts/deploy-infrastructure.sh --dry-run --verbose
```

#### Step-by-Step Deployment

1. **Initialize Terraform Backend**
```bash
cd terraform

# Create S3 bucket for state (one-time setup)
aws s3 mb s3://agentscan-terraform-state-$(date +%s)

# Create DynamoDB table for locking
aws dynamodb create-table \
    --table-name agentscan-terraform-locks \
    --attribute-definitions AttributeName=LockID,AttributeType=S \
    --key-schema AttributeName=LockID,KeyType=HASH \
    --provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5

# Update backend configuration in main.tf
```

2. **Deploy Infrastructure**
```bash
# Initialize and plan
terraform init
terraform plan -var="environment=production"

# Apply infrastructure
terraform apply -var="environment=production"
```

3. **Configure Kubernetes Access**
```bash
# Update kubeconfig
aws eks update-kubeconfig --region us-west-2 --name agentscan-production

# Verify connection
kubectl cluster-info
```

4. **Deploy Application**
```bash
# Deploy Kubernetes resources
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secrets.yaml
kubectl apply -f k8s/postgresql.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/api-deployment.yaml
kubectl apply -f k8s/orchestrator-deployment.yaml
kubectl apply -f k8s/ingress.yaml

# Deploy monitoring
kubectl apply -f k8s/monitoring/prometheus-rules.yaml
```

### 2. Blue-Green Deployment Setup

```bash
# Deploy blue-green infrastructure
kubectl apply -f k8s/blue-green/

# Verify deployments
kubectl get deployments -n agentscan

# Test blue-green deployment
./scripts/blue-green-deploy.sh deploy v1.0.0
```

### 3. Monitoring Setup

The monitoring stack is automatically deployed via Terraform Helm releases:

```bash
# Check monitoring components
kubectl get pods -n monitoring

# Access Grafana (get admin password from Secrets Manager)
kubectl port-forward -n monitoring svc/prometheus-stack-grafana 3000:80

# Access Prometheus
kubectl port-forward -n monitoring svc/prometheus-stack-prometheus 9090:9090

# Access Jaeger
kubectl port-forward -n monitoring svc/jaeger-query 16686:16686
```

## Configuration

### Environment Variables

Update the following files with your specific configuration:

#### `k8s/configmap.yaml`
```yaml
data:
  # Update with your actual endpoints
  GITHUB_API_URL: "https://api.github.com"
  GITLAB_API_URL: "https://gitlab.com/api/v4"
```

#### `k8s/secrets.yaml`
```bash
# Generate and encode secrets
echo -n "your-jwt-secret" | base64
echo -n "your-github-client-secret" | base64

# Update secrets.yaml with encoded values
```

### Terraform Variables

Create `terraform/terraform.tfvars`:
```hcl
environment = "production"
aws_region  = "us-west-2"
domain_name = "agentscan.dev"
certificate_arn = "arn:aws:acm:us-west-2:123456789012:certificate/..."

# Node group configuration
node_groups = {
  general = {
    instance_types = ["m5.large"]
    min_size      = 2
    max_size      = 10
    desired_size  = 3
  }
  
  compute = {
    instance_types = ["c5.xlarge"]
    min_size      = 0
    max_size      = 20
    desired_size  = 2
  }
}

# Database configuration
database_config = {
  instance_class    = "db.r6g.large"
  allocated_storage = 100
  multi_az          = true
}
```

## Backup and Disaster Recovery

### Automated Backups

The infrastructure includes comprehensive backup automation:

```bash
# Manual backup
./scripts/disaster-recovery.sh backup --backup-bucket agentscan-backups

# List available backups
./scripts/disaster-recovery.sh list-backups --backup-bucket agentscan-backups

# Validate backup
./scripts/disaster-recovery.sh validate-backup --backup-bucket agentscan-backups --timestamp 20240101-120000
```

### Disaster Recovery

```bash
# Test recovery (dry run)
./scripts/disaster-recovery.sh test-recovery \
    --backup-bucket agentscan-backups \
    --timestamp 20240101-120000

# Full recovery
./scripts/disaster-recovery.sh restore \
    --backup-bucket agentscan-backups \
    --timestamp 20240101-120000
```

### Backup Components

1. **Database Backups**
   - Automated RDS snapshots (daily)
   - Point-in-time recovery enabled
   - Cross-region backup replication

2. **Application Data**
   - Kubernetes resource manifests
   - Persistent volume data
   - Configuration and secrets

3. **Infrastructure State**
   - Terraform state files
   - Helm release configurations
   - Custom resource definitions

## Monitoring and Alerting

### Key Metrics

The platform monitors the following critical metrics:

- **Application Health**: API availability, response times, error rates
- **Infrastructure**: CPU, memory, disk usage, network performance
- **Security**: Authentication failures, suspicious activity, rate limiting
- **Business**: Scan completion rates, queue backlogs, agent health

### Alert Configuration

Alerts are configured in `k8s/monitoring/prometheus-rules.yaml`:

```yaml
# Example: High error rate alert
- alert: HighErrorRate
  expr: |
    (
      rate(http_requests_total{job=~"agentscan-.*",code=~"5.."}[5m]) /
      rate(http_requests_total{job=~"agentscan-.*"}[5m])
    ) > 0.05
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High error rate detected"
    description: "Error rate is {{ $value | humanizePercentage }}"
```

### Notification Channels

Configure notification channels in AlertManager:

- **Email**: Critical alerts to on-call team
- **Slack**: Real-time notifications to development channel
- **PagerDuty**: Critical alerts for immediate response
- **Webhook**: Integration with incident management systems

## Security Considerations

### Network Security
- VPC with private subnets for application components
- Network policies for pod-to-pod communication
- Security groups with least-privilege access
- VPC endpoints for secure AWS service access

### Data Security
- Encryption at rest for all data stores
- Encryption in transit with TLS 1.3
- Secrets managed via AWS Secrets Manager
- Regular security scanning and updates

### Access Control
- IAM roles with least-privilege permissions
- Kubernetes RBAC for service accounts
- OAuth integration for user authentication
- Audit logging for all administrative actions

## Scaling and Performance

### Auto Scaling

The infrastructure includes multiple auto-scaling mechanisms:

1. **Cluster Autoscaler**: Automatically scales EKS nodes
2. **Horizontal Pod Autoscaler**: Scales application pods based on metrics
3. **Vertical Pod Autoscaler**: Adjusts resource requests/limits
4. **Database Scaling**: Read replicas and connection pooling

### Performance Optimization

- **Caching**: Redis for frequently accessed data
- **CDN**: CloudFront for static assets
- **Database**: Connection pooling and query optimization
- **Container**: Resource limits and requests tuning

## Troubleshooting

### Common Issues

1. **Pod Startup Failures**
```bash
# Check pod status and logs
kubectl get pods -n agentscan
kubectl describe pod <pod-name> -n agentscan
kubectl logs <pod-name> -n agentscan
```

2. **Database Connection Issues**
```bash
# Check database connectivity
kubectl exec -it deployment/agentscan-api -n agentscan -- nc -zv postgresql.agentscan.svc.cluster.local 5432

# Check secrets
kubectl get secret agentscan-secrets -n agentscan -o yaml
```

3. **Ingress Issues**
```bash
# Check ingress status
kubectl get ingress -n agentscan
kubectl describe ingress agentscan-ingress -n agentscan

# Check load balancer
aws elbv2 describe-load-balancers
```

### Health Checks

```bash
# API health check
kubectl exec -n agentscan deployment/agentscan-api -- wget -qO- http://localhost:8080/health

# Orchestrator health check
kubectl exec -n agentscan deployment/agentscan-orchestrator -- wget -qO- http://localhost:8081/health

# Database health check
kubectl exec -n agentscan deployment/postgresql -- pg_isready -U agentscan
```

## Maintenance

### Regular Tasks

1. **Security Updates**
   - Update base images monthly
   - Apply Kubernetes security patches
   - Rotate secrets quarterly

2. **Performance Review**
   - Review resource utilization weekly
   - Optimize database queries monthly
   - Update scaling policies as needed

3. **Backup Verification**
   - Test backup restoration monthly
   - Verify cross-region replication
   - Update disaster recovery procedures

### Upgrade Procedures

1. **Kubernetes Upgrades**
```bash
# Upgrade EKS cluster
aws eks update-cluster-version --name agentscan-production --version 1.28

# Upgrade node groups
aws eks update-nodegroup-version --cluster-name agentscan-production --nodegroup-name general
```

2. **Application Upgrades**
```bash
# Use blue-green deployment
./scripts/blue-green-deploy.sh deploy v1.1.0
```

## Cost Optimization

### Resource Right-Sizing
- Regular review of resource utilization
- Adjust instance types based on workload patterns
- Use Spot instances for non-critical workloads

### Storage Optimization
- Implement lifecycle policies for S3 backups
- Use appropriate storage classes
- Regular cleanup of unused resources

### Monitoring Costs
- Set up billing alerts
- Use AWS Cost Explorer for analysis
- Implement resource tagging for cost allocation

## Support and Documentation

### Additional Resources
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [Prometheus Monitoring](https://prometheus.io/docs/)
- [AWS EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)

### Getting Help
- Internal documentation: `docs/`
- Runbooks: `docs/runbooks/`
- Architecture decisions: `docs/adr/`
- API documentation: `docs/api/`