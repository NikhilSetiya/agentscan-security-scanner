# AgentScan Production Deployment Guide

This guide covers the complete deployment process for AgentScan to production, including automated deployment, monitoring, and beta user onboarding.

## 🚀 Quick Start

For a rapid deployment to DigitalOcean App Platform:

```bash
# Set required environment variables
export JWT_SECRET="your-secure-jwt-secret"
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_SECRET="your-github-client-secret"

# Deploy to DigitalOcean
./scripts/deploy.sh deploy
```

## 📋 Prerequisites

### Required Tools
- [doctl](https://docs.digitalocean.com/reference/doctl/how-to/install/) - DigitalOcean CLI
- [Docker](https://docs.docker.com/get-docker/) - Container runtime
- [Git](https://git-scm.com/downloads) - Version control
- [curl](https://curl.se/download.html) - HTTP client
- [jq](https://stedolan.github.io/jq/download/) - JSON processor

### Required Accounts
- DigitalOcean account with App Platform access
- GitHub account with repository access
- SendGrid account for email notifications (optional)
- Slack workspace for alerts (optional)

### Environment Variables

```bash
# Required for deployment
export JWT_SECRET="your-secure-jwt-secret-min-32-chars"
export GITHUB_CLIENT_ID="your-github-oauth-client-id"
export GITHUB_SECRET="your-github-oauth-client-secret"

# Optional for enhanced features
export SENDGRID_API_KEY="your-sendgrid-api-key"
export SLACK_WEBHOOK_URL="your-slack-webhook-url"
export ALERT_EMAIL="alerts@yourcompany.com"
```

## 🏗️ Architecture Overview

AgentScan uses a microservices architecture deployed on DigitalOcean App Platform:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend  │    │   API Server    │    │  Orchestrator   │
│   (React/Next)  │    │   (Go/Gin)     │    │   (Go/Worker)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │   PostgreSQL    │    │     Redis       │    │  Scan Workers   │
         │   (Database)    │    │   (Cache/Queue) │    │  (Containers)   │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🚀 Deployment Process

### Step 1: Prepare Environment

1. **Clone the repository:**
   ```bash
   git clone https://github.com/NikhilSetiya/agentscan-security-scanner.git
   cd agentscan-security-scanner
   ```

2. **Install and configure doctl:**
   ```bash
   # Install doctl
   curl -sL https://github.com/digitalocean/doctl/releases/download/v1.94.0/doctl-1.94.0-linux-amd64.tar.gz | tar -xzv
   sudo mv doctl /usr/local/bin
   
   # Authenticate
   doctl auth init
   ```

3. **Set environment variables:**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   source .env
   ```

### Step 2: Deploy to DigitalOcean

1. **Run deployment script:**
   ```bash
   ./scripts/deploy.sh check    # Verify prerequisites
   ./scripts/deploy.sh deploy   # Deploy application
   ```

2. **Configure secrets in DigitalOcean dashboard:**
   - Navigate to your app in the DigitalOcean dashboard
   - Go to Settings → Environment Variables
   - Add the required secrets (JWT_SECRET, GITHUB_CLIENT_ID, GITHUB_SECRET)

3. **Set up custom domains:**
   - Configure DNS records for your domains
   - Add domains in the DigitalOcean dashboard
   - Enable SSL certificates

### Step 3: Set Up Monitoring

1. **Deploy monitoring stack:**
   ```bash
   ./scripts/setup-monitoring.sh \
     --domain monitoring.agentscan.dev \
     --slack-webhook "$SLACK_WEBHOOK_URL" \
     --email "$ALERT_EMAIL"
   ```

2. **Start monitoring services:**
   ```bash
   cd monitoring
   ./start-monitoring.sh
   ```

3. **Access monitoring dashboards:**
   - Grafana: http://localhost:3000 (admin/admin123)
   - Prometheus: http://localhost:9090
   - Alertmanager: http://localhost:9093

## 👥 Beta User Onboarding

### Automated Repository Onboarding

Set up new repositories for beta users:

```bash
./scripts/onboard-repository.sh myorg/myrepo \
  --token "$GITHUB_TOKEN" \
  --api-key "$AGENTSCAN_API_KEY" \
  --auto-scan \
  --pr-comments \
  --status-checks
```

### Beta Invitation System

Send automated beta invitations:

```bash
# Send pending invitations
./cmd/beta-inviter/beta-inviter send-invites

# Show invitation statistics
./cmd/beta-inviter/beta-inviter stats

# Clean up expired invitations
./cmd/beta-inviter/beta-inviter cleanup
```

### Demo Environment Setup

Create self-service demo repositories:

```bash
./demo/setup-demo.sh \
  --github-token "$GITHUB_TOKEN" \
  --api-key "$AGENTSCAN_API_KEY" \
  --org agentscan-demo \
  --public
```

## 🔧 Configuration

### App Platform Configuration

The deployment uses `.do/app.yaml` for configuration:

- **Services:** API server, orchestrator, web frontend
- **Workers:** Scan workers for background processing
- **Databases:** PostgreSQL and Redis
- **Jobs:** Database migrations
- **Static Sites:** Documentation hosting

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `JWT_SECRET` | JWT signing secret (min 32 chars) | Yes |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | Yes |
| `GITHUB_SECRET` | GitHub OAuth client secret | Yes |
| `DB_*` | Database connection settings | Auto |
| `REDIS_*` | Redis connection settings | Auto |
| `AGENTS_MAX_CONCURRENT` | Max concurrent agents | No |
| `LOG_LEVEL` | Logging level | No |

### Scaling Configuration

- **API Server:** 2 instances (basic-xxs)
- **Orchestrator:** 1 instance (basic-xs)
- **Web Frontend:** 2 instances (basic-xxs)
- **Scan Workers:** 3 instances (basic-s)

## 📊 Monitoring and Alerting

### Metrics Collected

- **Application Metrics:** Request rate, response time, error rate
- **System Metrics:** CPU, memory, disk usage
- **Business Metrics:** Scan queue size, agent failures
- **Infrastructure Metrics:** Database connections, Redis memory

### Alert Rules

- Service downtime (5+ minutes)
- High error rate (>10%)
- High response time (>2 seconds)
- Resource usage thresholds
- Scan queue backlog

### Dashboards

- **AgentScan Overview:** Service status, request metrics
- **System Health:** Resource usage, performance
- **Business Metrics:** Scan statistics, user activity

## 🔒 Security Considerations

### Secrets Management

- Use DigitalOcean App Platform environment variables
- Rotate secrets regularly
- Never commit secrets to version control
- Use strong, randomly generated secrets

### Network Security

- All services communicate over HTTPS
- Database and Redis are private
- API authentication required for all endpoints
- Rate limiting enabled

### Data Protection

- Database encryption at rest
- TLS encryption in transit
- Regular automated backups
- GDPR compliance measures

## 🚨 Troubleshooting

### Common Issues

1. **Deployment Fails:**
   ```bash
   # Check app logs
   doctl apps logs agentscan-production --type=run
   
   # Check deployment status
   doctl apps get agentscan-production
   ```

2. **Service Not Responding:**
   ```bash
   # Check health endpoints
   curl https://api.agentscan.dev/health
   
   # Check service status
   doctl apps get agentscan-production --format ID,Spec.Name,LiveURL
   ```

3. **Database Connection Issues:**
   ```bash
   # Check database status
   doctl databases list
   
   # Check connection settings
   doctl apps get agentscan-production --format Spec.Envs
   ```

### Health Checks

The deployment includes comprehensive health checks:

- **API Health:** `/health` endpoint
- **Database Health:** Connection and query tests
- **Redis Health:** Connection and ping tests
- **Agent Health:** Container status checks

### Log Analysis

Access logs through multiple channels:

```bash
# Application logs
doctl apps logs agentscan-production --type=run --follow

# Build logs
doctl apps logs agentscan-production --type=build

# Deploy logs
doctl apps logs agentscan-production --type=deploy
```

## 📈 Performance Optimization

### Caching Strategy

- **Redis:** Session storage, job queues, rate limiting
- **CDN:** Static assets, API responses
- **Database:** Query optimization, connection pooling

### Scaling Guidelines

- **Horizontal Scaling:** Add more instances during peak usage
- **Vertical Scaling:** Increase instance sizes for resource-intensive operations
- **Database Scaling:** Use read replicas for read-heavy workloads

### Performance Monitoring

- Response time percentiles (P50, P95, P99)
- Throughput metrics (requests per second)
- Error rates by endpoint
- Resource utilization trends

## 🔄 Maintenance

### Regular Tasks

1. **Weekly:**
   - Review monitoring dashboards
   - Check error logs
   - Verify backup integrity

2. **Monthly:**
   - Update dependencies
   - Review security alerts
   - Analyze performance trends

3. **Quarterly:**
   - Security audit
   - Disaster recovery testing
   - Capacity planning review

### Backup and Recovery

- **Database:** Automated daily backups with 30-day retention
- **Redis:** Persistence enabled with AOF
- **Application:** Git-based deployment enables quick rollbacks

### Updates and Rollbacks

```bash
# Deploy new version
git push origin main  # Triggers automatic deployment

# Rollback to previous version
doctl apps create-deployment agentscan-production --force-rebuild

# Check deployment history
doctl apps list-deployments agentscan-production
```

## 📞 Support

### Getting Help

- **Documentation:** https://docs.agentscan.dev
- **GitHub Issues:** https://github.com/NikhilSetiya/agentscan-security-scanner/issues
- **Email Support:** support@agentscan.dev
- **Slack Community:** [Join our Slack](https://agentscan.dev/slack)

### Emergency Contacts

- **On-call Engineer:** +1-XXX-XXX-XXXX
- **DevOps Team:** devops@agentscan.dev
- **Security Team:** security@agentscan.dev

---

## 📝 Changelog

### v1.0.0 - Production Release
- Initial production deployment
- Automated monitoring and alerting
- Beta user onboarding system
- Self-service demo environment
- Comprehensive documentation

---

**Need help?** Contact our support team or check the troubleshooting section above.