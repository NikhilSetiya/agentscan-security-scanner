# AgentScan Production Deployment Guide

This comprehensive guide covers the complete production deployment process for AgentScan, including setup, configuration, monitoring, and maintenance procedures.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Infrastructure Setup](#infrastructure-setup)
- [Deployment Process](#deployment-process)
- [Configuration Management](#configuration-management)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Security Considerations](#security-considerations)
- [Maintenance Procedures](#maintenance-procedures)
- [Troubleshooting](#troubleshooting)
- [Rollback Procedures](#rollback-procedures)

## Overview

AgentScan production deployment consists of:

- **Backend API**: Deployed on Fly.io with PostgreSQL and Redis
- **Frontend**: Deployed on Vercel with CDN and edge optimization
- **Database**: PostgreSQL on Fly.io with automated backups
- **Cache**: Redis on Fly.io for session and application caching
- **Authentication**: Supabase for user management and authentication
- **Monitoring**: Prometheus metrics with Grafana dashboards
- **Alerting**: Production alerts for critical system events

## Prerequisites

### Required Tools

```bash
# Install Fly.io CLI
curl -L https://fly.io/install.sh | sh

# Install Vercel CLI
npm install -g vercel

# Install Supabase CLI
npm install -g supabase

# Required system tools
brew install jq curl openssl postgresql redis
```

### Required Accounts

- [Fly.io Account](https://fly.io) - For backend and database hosting
- [Vercel Account](https://vercel.com) - For frontend hosting
- [Supabase Account](https://supabase.com) - For authentication services
- [GitHub Account](https://github.com) - For code repository and CI/CD

### Environment Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/your-org/agentscan-security-scanner.git
   cd agentscan-security-scanner
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   cd web/frontend && npm install
   ```

## Infrastructure Setup

### 1. Supabase Project Setup

```bash
# Login to Supabase
supabase login

# Create new project
supabase projects create agentscan-prod --org-id YOUR_ORG_ID --db-password SECURE_PASSWORD

# Get project details
supabase projects list
```

### 2. Fly.io Infrastructure Setup

```bash
# Login to Fly.io
fly auth login

# Create PostgreSQL database
fly postgres create --name agentscan-db --region sjc --vm-size shared-cpu-1x --volume-size 10

# Create Redis instance
fly redis create --name agentscan-redis --region sjc

# Create main application
fly apps create agentscan-prod --org personal
```

### 3. Environment Configuration

Run the production services setup script:

```bash
./deployment/scripts/setup-production-services.sh
```

This script will:
- Set up Supabase project and retrieve API keys
- Configure Fly.io PostgreSQL and Redis instances
- Generate secure production secrets
- Create the `.env.production` file with all required variables

## Deployment Process

### Automated Deployment

Use the production deployment script for a complete deployment:

```bash
./deployment/scripts/deploy-production.sh
```

### Manual Deployment Steps

#### 1. Backend Deployment (Fly.io)

```bash
# Set production secrets
fly secrets set \
  DATABASE_URL="postgres://..." \
  REDIS_URL="redis://..." \
  SUPABASE_URL="https://..." \
  SUPABASE_SERVICE_ROLE_KEY="..." \
  JWT_SECRET="..." \
  --app agentscan-prod

# Deploy application
fly deploy --config deployment/production/fly.toml --dockerfile deployment/production/Dockerfile

# Verify deployment
fly status --app agentscan-prod
```

#### 2. Database Migration

```bash
# Run migrations
fly ssh console --app agentscan-prod
./bin/migrate up
```

#### 3. Frontend Deployment (Vercel)

```bash
cd web/frontend

# Set environment variables
vercel env add VITE_API_URL production
vercel env add VITE_SUPABASE_URL production
vercel env add VITE_SUPABASE_ANON_KEY production

# Deploy to production
vercel --prod
```

### Deployment Verification

Run comprehensive smoke tests:

```bash
./deployment/scripts/run-smoke-tests.sh
```

Or use the verification script:

```bash
./deployment/scripts/verify-production.sh
```

## Configuration Management

### Environment Variables

All production configuration is managed through environment variables. Key variables include:

#### Application Configuration
```bash
APP_NAME=AgentScan
APP_VERSION=1.0.0
GO_ENV=production
PORT=8080
```

#### Database Configuration
```bash
DATABASE_URL=postgres://user:pass@host:port/db?sslmode=require
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=10
```

#### Security Configuration
```bash
JWT_SECRET=your_secure_jwt_secret_32_chars_minimum
HTTPS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://agentscan.vercel.app
RATE_LIMIT_ENABLED=true
```

### Configuration Validation

Validate production configuration:

```bash
# Check configuration
./deployment/scripts/setup-production-services.sh --validate-only

# Or use Go validation
go run cmd/config-check/main.go
```

## Monitoring and Alerting

### Metrics Collection

AgentScan exposes Prometheus metrics at `/metrics`:

- **HTTP Metrics**: Request rates, response times, error rates
- **Database Metrics**: Connection pools, query performance
- **Business Metrics**: Scan statistics, user activity
- **System Metrics**: CPU, memory, disk usage

### Dashboards

Import the production dashboard:

```bash
# Grafana dashboard configuration
curl -X POST \
  -H "Content-Type: application/json" \
  -d @deployment/production/monitoring/dashboard.json \
  http://grafana:3000/api/dashboards/db
```

### Alerting Rules

Production alerts are configured in `deployment/production/monitoring/alerts.yml`:

- **Critical Alerts**: Application down, high error rates, database failures
- **Warning Alerts**: High resource usage, slow responses, security events
- **Business Alerts**: Scan failures, authentication issues

### Alert Channels

Configure alert notifications:

```yaml
# Slack notifications
- name: slack-alerts
  slack_configs:
    - api_url: 'YOUR_SLACK_WEBHOOK_URL'
      channel: '#alerts'
      title: 'AgentScan Production Alert'

# Email notifications  
- name: email-alerts
  email_configs:
    - to: 'ops@yourcompany.com'
      subject: 'AgentScan Production Alert'
```

## Security Considerations

### SSL/TLS Configuration

- **HTTPS Enforcement**: All traffic redirected to HTTPS
- **HSTS Headers**: Strict Transport Security enabled
- **Certificate Management**: Automatic certificate renewal via Fly.io/Vercel

### Security Headers

```yaml
Strict-Transport-Security: max-age=31536000; includeSubDomains; preload
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'; ...
```

### Authentication Security

- **JWT Tokens**: Secure token generation and validation
- **Session Management**: Secure session handling with Supabase
- **Rate Limiting**: Protection against brute force attacks
- **Input Validation**: Comprehensive input sanitization

### Database Security

- **SSL Connections**: All database connections use SSL
- **Connection Pooling**: Secure connection pool management
- **Row Level Security**: Database-level access control
- **Audit Logging**: Comprehensive audit trail

## Maintenance Procedures

### Regular Maintenance Tasks

#### Daily
- Monitor application health and performance
- Check error rates and logs
- Verify backup completion
- Review security alerts

#### Weekly
- Review performance metrics and trends
- Check resource utilization
- Update dependencies (if needed)
- Rotate and archive logs

#### Monthly
- Security vulnerability assessment
- Performance optimization review
- Backup restoration testing
- Disaster recovery drill

### Database Maintenance

```bash
# Connect to database
fly postgres connect --app agentscan-db

# Run maintenance queries
VACUUM ANALYZE;
REINDEX DATABASE agentscan_prod;

# Check database statistics
SELECT * FROM pg_stat_database WHERE datname = 'agentscan_prod';
```

### Log Management

```bash
# View application logs
fly logs --app agentscan-prod

# Export logs for analysis
fly logs --app agentscan-prod --output json > logs.json

# Clean up old logs (automated via retention policy)
```

### Backup Procedures

```bash
# Manual database backup
fly postgres backup --app agentscan-db

# Verify backup integrity
fly postgres backup list --app agentscan-db

# Restore from backup (if needed)
fly postgres restore --app agentscan-db --backup-id BACKUP_ID
```

## Troubleshooting

### Common Issues

#### Application Won't Start

1. **Check environment variables**:
   ```bash
   fly secrets list --app agentscan-prod
   ```

2. **Verify database connectivity**:
   ```bash
   fly ssh console --app agentscan-prod
   ./bin/migrate status
   ```

3. **Check application logs**:
   ```bash
   fly logs --app agentscan-prod
   ```

#### High Response Times

1. **Check database performance**:
   ```sql
   SELECT query, mean_time, calls 
   FROM pg_stat_statements 
   ORDER BY mean_time DESC LIMIT 10;
   ```

2. **Monitor resource usage**:
   ```bash
   fly status --app agentscan-prod
   ```

3. **Review slow query logs**:
   ```bash
   fly logs --app agentscan-prod | grep "slow query"
   ```

#### Database Connection Issues

1. **Check connection pool status**:
   ```bash
   # View metrics endpoint
   curl https://agentscan-prod.fly.dev/metrics | grep database_connections
   ```

2. **Verify database health**:
   ```bash
   fly postgres status --app agentscan-db
   ```

3. **Check SSL configuration**:
   ```bash
   psql "$DATABASE_URL" -c "SHOW ssl;"
   ```

### Performance Optimization

#### Database Optimization

```sql
-- Add missing indexes
CREATE INDEX CONCURRENTLY idx_scans_user_created 
ON scans(user_id, created_at DESC);

-- Update table statistics
ANALYZE scans;

-- Check query performance
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM scans WHERE user_id = $1;
```

#### Application Optimization

```bash
# Enable Go profiling (temporarily)
fly ssh console --app agentscan-prod
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof

# Analyze memory usage
curl http://localhost:6060/debug/pprof/heap > heap.prof
```

## Rollback Procedures

### Automated Rollback

```bash
# Rollback to previous release
./deployment/scripts/deploy-production.sh --rollback
```

### Manual Rollback

#### Backend Rollback (Fly.io)

```bash
# List recent releases
fly releases list --app agentscan-prod

# Rollback to specific version
fly releases rollback v123 --app agentscan-prod
```

#### Frontend Rollback (Vercel)

```bash
cd web/frontend

# List deployments
vercel ls

# Rollback to previous deployment
vercel rollback DEPLOYMENT_URL
```

#### Database Rollback

```bash
# Rollback migrations (if needed)
fly ssh console --app agentscan-prod
./bin/migrate down 1

# Restore from backup (if needed)
fly postgres restore --app agentscan-db --backup-id BACKUP_ID
```

### Rollback Verification

After rollback, run verification tests:

```bash
# Quick health check
curl https://agentscan-prod.fly.dev/health

# Full verification
./deployment/scripts/verify-production.sh

# Smoke tests
./deployment/scripts/run-smoke-tests.sh --infrastructure
```

## Emergency Procedures

### Incident Response

1. **Immediate Response**:
   - Assess impact and severity
   - Check monitoring dashboards
   - Review recent deployments
   - Notify stakeholders

2. **Investigation**:
   - Collect logs and metrics
   - Identify root cause
   - Document findings

3. **Resolution**:
   - Apply fix or rollback
   - Verify resolution
   - Monitor for recurrence

4. **Post-Incident**:
   - Conduct post-mortem
   - Update procedures
   - Implement preventive measures

### Emergency Contacts

- **On-Call Engineer**: [Contact Information]
- **DevOps Team**: [Contact Information]
- **Security Team**: [Contact Information]
- **Management**: [Contact Information]

### Escalation Procedures

1. **Level 1**: On-call engineer responds within 15 minutes
2. **Level 2**: Team lead involved within 30 minutes
3. **Level 3**: Management notified within 1 hour
4. **Level 4**: External support engaged if needed

## Compliance and Auditing

### Audit Logging

All production activities are logged:

- **Authentication Events**: Login attempts, token generation
- **API Access**: All API requests and responses
- **Database Changes**: All data modifications
- **System Events**: Deployments, configuration changes

### Compliance Requirements

- **Data Protection**: GDPR/CCPA compliance measures
- **Security Standards**: SOC 2 Type II compliance
- **Audit Trail**: Complete audit logging and retention
- **Access Control**: Role-based access control (RBAC)

### Regular Audits

- **Security Audits**: Quarterly security assessments
- **Performance Audits**: Monthly performance reviews
- **Compliance Audits**: Annual compliance verification
- **Code Audits**: Continuous code quality monitoring

## Support and Resources

### Documentation

- **API Documentation**: [Link to API docs]
- **User Guide**: [Link to user guide]
- **Architecture Overview**: [Link to architecture docs]
- **Security Guide**: [Link to security docs]

### Monitoring URLs

- **Production API**: https://agentscan-prod.fly.dev
- **Frontend**: https://agentscan.vercel.app
- **Metrics**: https://agentscan-prod.fly.dev/metrics
- **Health Check**: https://agentscan-prod.fly.dev/health

### Support Channels

- **Internal Chat**: #agentscan-support
- **Issue Tracking**: GitHub Issues
- **Documentation**: Internal Wiki
- **Runbooks**: [Link to runbooks]

---

**Last Updated**: [Current Date]
**Version**: 1.0.0
**Maintained By**: DevOps Team