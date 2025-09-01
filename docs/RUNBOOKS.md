# AgentScan Production Runbooks

This document contains step-by-step procedures for handling common production scenarios and incidents.

## Table of Contents

- [General Procedures](#general-procedures)
- [Application Issues](#application-issues)
- [Database Issues](#database-issues)
- [Performance Issues](#performance-issues)
- [Security Incidents](#security-incidents)
- [Infrastructure Issues](#infrastructure-issues)
- [Monitoring and Alerting](#monitoring-and-alerting)

## General Procedures

### Initial Response Checklist

When responding to any production incident:

1. **Acknowledge the alert** within 5 minutes
2. **Assess the impact** - check dashboards and metrics
3. **Check recent changes** - deployments, configuration changes
4. **Gather initial information** - logs, error messages, affected users
5. **Communicate status** - update incident channel
6. **Begin investigation** - follow specific runbook procedures

### Communication Protocol

- **Incident Channel**: #agentscan-incidents
- **Status Updates**: Every 15 minutes during active incidents
- **Stakeholder Notification**: Within 30 minutes for customer-impacting issues
- **Post-Incident Report**: Within 24 hours of resolution

## Application Issues

### Runbook: Application Down

**Alert**: `ApplicationDown`
**Severity**: Critical
**SLA**: 5 minutes response, 15 minutes resolution

#### Symptoms
- Health check endpoint returning 500 or timing out
- Users unable to access the application
- Monitoring shows application as down

#### Investigation Steps

1. **Check application status**:
   ```bash
   fly status --app agentscan-prod
   ```

2. **Review recent logs**:
   ```bash
   fly logs --app agentscan-prod | tail -100
   ```

3. **Check for recent deployments**:
   ```bash
   fly releases list --app agentscan-prod
   ```

4. **Verify infrastructure health**:
   ```bash
   # Check database
   fly postgres status --app agentscan-db
   
   # Check Redis
   fly redis status --app agentscan-redis
   ```

#### Resolution Steps

**If application crashed**:
```bash
# Restart the application
fly restart --app agentscan-prod

# Monitor restart process
fly logs --app agentscan-prod -f
```

**If deployment issue**:
```bash
# Rollback to previous version
fly releases rollback v$(previous_version) --app agentscan-prod
```

**If infrastructure issue**:
```bash
# Scale up if needed
fly scale count 2 --app agentscan-prod

# Check resource limits
fly scale show --app agentscan-prod
```

#### Verification
```bash
# Test health endpoint
curl https://agentscan-prod.fly.dev/health

# Run smoke tests
./deployment/scripts/run-smoke-tests.sh --infrastructure
```

### Runbook: High Error Rate

**Alert**: `HighErrorRate`
**Severity**: Critical
**SLA**: 10 minutes response, 30 minutes resolution

#### Symptoms
- Error rate > 5% for 2+ minutes
- Increased 5xx responses
- User reports of application errors

#### Investigation Steps

1. **Check error distribution**:
   ```bash
   # View recent error logs
   fly logs --app agentscan-prod | grep -E "(ERROR|FATAL|5[0-9][0-9])"
   
   # Check metrics
   curl https://agentscan-prod.fly.dev/metrics | grep http_requests_total
   ```

2. **Identify error patterns**:
   ```bash
   # Group errors by endpoint
   fly logs --app agentscan-prod | grep "ERROR" | awk '{print $5}' | sort | uniq -c
   ```

3. **Check dependencies**:
   ```bash
   # Database connectivity
   curl https://agentscan-prod.fly.dev/health | jq '.checks.database'
   
   # Redis connectivity  
   curl https://agentscan-prod.fly.dev/health | jq '.checks.redis'
   ```

#### Resolution Steps

**Database connection errors**:
```bash
# Check connection pool
curl https://agentscan-prod.fly.dev/metrics | grep database_connections

# Restart if connection pool exhausted
fly restart --app agentscan-prod
```

**Memory/resource issues**:
```bash
# Check resource usage
fly status --app agentscan-prod

# Scale up if needed
fly scale memory 1024 --app agentscan-prod
```

**Code-related errors**:
```bash
# Rollback to previous version
fly releases rollback v$(previous_version) --app agentscan-prod
```

### Runbook: High Response Time

**Alert**: `HighResponseTime`
**Severity**: Warning
**SLA**: 15 minutes response, 1 hour resolution

#### Investigation Steps

1. **Check response time metrics**:
   ```bash
   curl https://agentscan-prod.fly.dev/metrics | grep http_request_duration
   ```

2. **Identify slow endpoints**:
   ```bash
   fly logs --app agentscan-prod | grep "slow" | tail -20
   ```

3. **Check database performance**:
   ```bash
   # Connect to database
   fly postgres connect --app agentscan-db
   
   # Check slow queries
   SELECT query, mean_time, calls 
   FROM pg_stat_statements 
   ORDER BY mean_time DESC LIMIT 10;
   ```

#### Resolution Steps

**Database performance issues**:
```sql
-- Kill long-running queries
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE state = 'active' AND query_start < NOW() - INTERVAL '5 minutes';

-- Update statistics
ANALYZE;
```

**Application performance issues**:
```bash
# Scale horizontally
fly scale count 3 --app agentscan-prod

# Scale vertically if needed
fly scale memory 2048 --app agentscan-prod
```

## Database Issues

### Runbook: Database Connection Failure

**Alert**: `DatabaseConnectionFailure`
**Severity**: Critical
**SLA**: 5 minutes response, 15 minutes resolution

#### Investigation Steps

1. **Check database status**:
   ```bash
   fly postgres status --app agentscan-db
   ```

2. **Test connectivity**:
   ```bash
   # From application
   fly ssh console --app agentscan-prod
   psql "$DATABASE_URL" -c "SELECT 1;"
   ```

3. **Check connection limits**:
   ```bash
   fly postgres connect --app agentscan-db
   SELECT * FROM pg_stat_activity;
   SELECT count(*) FROM pg_stat_activity;
   ```

#### Resolution Steps

**Connection limit exceeded**:
```sql
-- Kill idle connections
SELECT pg_terminate_backend(pid) 
FROM pg_stat_activity 
WHERE state = 'idle' AND state_change < NOW() - INTERVAL '1 hour';
```

**Database down**:
```bash
# Restart database
fly restart --app agentscan-db

# Check logs
fly logs --app agentscan-db
```

**Network issues**:
```bash
# Restart application to reset connections
fly restart --app agentscan-prod
```

### Runbook: Database High Connections

**Alert**: `DatabaseHighConnections`
**Severity**: Warning
**SLA**: 30 minutes response, 2 hours resolution

#### Investigation Steps

1. **Check current connections**:
   ```sql
   SELECT count(*), state FROM pg_stat_activity GROUP BY state;
   ```

2. **Identify connection sources**:
   ```sql
   SELECT client_addr, count(*) 
   FROM pg_stat_activity 
   GROUP BY client_addr 
   ORDER BY count DESC;
   ```

#### Resolution Steps

1. **Optimize connection pooling**:
   ```bash
   # Update application configuration
   fly secrets set DATABASE_MAX_OPEN_CONNS=20 --app agentscan-prod
   fly secrets set DATABASE_MAX_IDLE_CONNS=5 --app agentscan-prod
   ```

2. **Kill unnecessary connections**:
   ```sql
   -- Kill long-idle connections
   SELECT pg_terminate_backend(pid) 
   FROM pg_stat_activity 
   WHERE state = 'idle' AND state_change < NOW() - INTERVAL '30 minutes';
   ```

### Runbook: Slow Database Queries

**Alert**: `DatabaseSlowQueries`
**Severity**: Warning
**SLA**: 1 hour response, 4 hours resolution

#### Investigation Steps

1. **Identify slow queries**:
   ```sql
   SELECT query, mean_time, calls, total_time
   FROM pg_stat_statements 
   ORDER BY mean_time DESC LIMIT 10;
   ```

2. **Check for missing indexes**:
   ```sql
   SELECT schemaname, tablename, attname, n_distinct, correlation 
   FROM pg_stats 
   WHERE schemaname = 'public' 
   ORDER BY n_distinct DESC;
   ```

3. **Analyze query plans**:
   ```sql
   EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM scans WHERE user_id = 'example';
   ```

#### Resolution Steps

1. **Add missing indexes**:
   ```sql
   -- Example: Add index for common query pattern
   CREATE INDEX CONCURRENTLY idx_scans_user_status 
   ON scans(user_id, status) WHERE status IN ('running', 'pending');
   ```

2. **Update table statistics**:
   ```sql
   ANALYZE scans;
   ANALYZE findings;
   ```

3. **Optimize queries**:
   - Review application code for N+1 queries
   - Add appropriate WHERE clauses
   - Use LIMIT for large result sets

## Performance Issues

### Runbook: High CPU Usage

**Alert**: `HighCPUUsage`
**Severity**: Warning
**SLA**: 30 minutes response, 2 hours resolution

#### Investigation Steps

1. **Check CPU metrics**:
   ```bash
   fly status --app agentscan-prod
   ```

2. **Identify CPU-intensive processes**:
   ```bash
   fly ssh console --app agentscan-prod
   top -p $(pgrep -f agentscan)
   ```

3. **Check for CPU-intensive queries**:
   ```sql
   SELECT query, calls, total_time, mean_time
   FROM pg_stat_statements 
   ORDER BY total_time DESC LIMIT 10;
   ```

#### Resolution Steps

1. **Scale vertically**:
   ```bash
   fly scale vm shared-cpu-4x --app agentscan-prod
   ```

2. **Scale horizontally**:
   ```bash
   fly scale count 3 --app agentscan-prod
   ```

3. **Optimize application**:
   - Review CPU-intensive code paths
   - Add caching for expensive operations
   - Optimize database queries

### Runbook: High Memory Usage

**Alert**: `HighMemoryUsage`
**Severity**: Warning
**SLA**: 30 minutes response, 2 hours resolution

#### Investigation Steps

1. **Check memory usage**:
   ```bash
   fly ssh console --app agentscan-prod
   free -h
   ps aux --sort=-%mem | head -10
   ```

2. **Check for memory leaks**:
   ```bash
   # Enable pprof temporarily
   curl http://localhost:6060/debug/pprof/heap > heap.prof
   ```

#### Resolution Steps

1. **Increase memory**:
   ```bash
   fly scale memory 2048 --app agentscan-prod
   ```

2. **Restart application** (if memory leak suspected):
   ```bash
   fly restart --app agentscan-prod
   ```

3. **Optimize memory usage**:
   - Review large data structures
   - Implement pagination for large queries
   - Add memory-efficient caching

## Security Incidents

### Runbook: High Authentication Failure Rate

**Alert**: `HighAuthenticationFailureRate`
**Severity**: Warning
**SLA**: 15 minutes response, 1 hour resolution

#### Investigation Steps

1. **Check authentication logs**:
   ```bash
   fly logs --app agentscan-prod | grep "auth" | grep -i "fail"
   ```

2. **Identify source IPs**:
   ```bash
   fly logs --app agentscan-prod | grep "auth.*fail" | awk '{print $8}' | sort | uniq -c | sort -nr
   ```

3. **Check for patterns**:
   ```bash
   # Check for brute force attempts
   fly logs --app agentscan-prod | grep "auth.*fail" | grep -E "([0-9]{1,3}\.){3}[0-9]{1,3}" | head -20
   ```

#### Resolution Steps

1. **Block malicious IPs** (if identified):
   ```bash
   # Add to rate limiting rules
   fly secrets set BLOCKED_IPS="1.2.3.4,5.6.7.8" --app agentscan-prod
   ```

2. **Increase rate limiting**:
   ```bash
   fly secrets set RATE_LIMIT_REQUESTS=100 --app agentscan-prod
   fly secrets set RATE_LIMIT_WINDOW=1h --app agentscan-prod
   ```

3. **Monitor and alert**:
   - Set up additional monitoring for suspicious IPs
   - Consider implementing CAPTCHA for repeated failures

### Runbook: Suspicious Activity

**Alert**: `SuspiciousActivity`
**Severity**: Critical
**SLA**: 5 minutes response, 30 minutes assessment

#### Investigation Steps

1. **Review security events**:
   ```bash
   fly logs --app agentscan-prod | grep -i "security\|suspicious\|attack"
   ```

2. **Check access patterns**:
   ```bash
   # Unusual access patterns
   fly logs --app agentscan-prod | grep -E "(POST|PUT|DELETE)" | awk '{print $8}' | sort | uniq -c | sort -nr
   ```

3. **Review user activity**:
   ```sql
   -- Check recent user activities
   SELECT user_id, action, created_at, ip_address 
   FROM audit_logs 
   WHERE created_at > NOW() - INTERVAL '1 hour'
   ORDER BY created_at DESC;
   ```

#### Resolution Steps

1. **Immediate containment**:
   ```bash
   # Block suspicious IPs
   fly secrets set BLOCKED_IPS="suspicious.ip.list" --app agentscan-prod
   ```

2. **Investigate further**:
   - Review detailed logs
   - Check for data access patterns
   - Verify user account integrity

3. **Notify security team**:
   - Escalate to security team immediately
   - Preserve logs and evidence
   - Follow incident response procedures

## Infrastructure Issues

### Runbook: SSL Certificate Expiring

**Alert**: `SSLCertificateExpiringSoon`
**Severity**: Warning
**SLA**: 24 hours response, 7 days resolution

#### Investigation Steps

1. **Check certificate status**:
   ```bash
   echo | openssl s_client -servername agentscan-prod.fly.dev -connect agentscan-prod.fly.dev:443 2>/dev/null | openssl x509 -noout -dates
   ```

2. **Verify auto-renewal**:
   ```bash
   fly certs list --app agentscan-prod
   ```

#### Resolution Steps

1. **For Fly.io certificates** (auto-managed):
   ```bash
   # Check certificate status
   fly certs show agentscan-prod.fly.dev --app agentscan-prod
   
   # Force renewal if needed
   fly certs add agentscan-prod.fly.dev --app agentscan-prod
   ```

2. **For custom certificates**:
   ```bash
   # Update certificate
   fly certs add your-domain.com --app agentscan-prod
   ```

### Runbook: External Service Down

**Alert**: `ExternalServiceDown`
**Severity**: Warning
**SLA**: 30 minutes response, depends on service

#### Investigation Steps

1. **Identify affected service**:
   ```bash
   # Check which external service is down
   curl -I https://api.supabase.com/health
   ```

2. **Check service status pages**:
   - Supabase: https://status.supabase.com
   - Fly.io: https://status.fly.io
   - Vercel: https://vercel-status.com

#### Resolution Steps

1. **Implement fallback** (if available):
   ```bash
   # Enable fallback mode
   fly secrets set FALLBACK_MODE=true --app agentscan-prod
   ```

2. **Communicate impact**:
   - Update status page
   - Notify affected users
   - Provide ETA if available

3. **Monitor for recovery**:
   ```bash
   # Automated monitoring script
   while ! curl -f https://external-service.com/health; do
     echo "Service still down, checking again in 60s..."
     sleep 60
   done
   echo "Service recovered!"
   ```

## Monitoring and Alerting

### Runbook: Monitoring System Down

**Severity**: High
**SLA**: 15 minutes response, 1 hour resolution

#### Investigation Steps

1. **Check monitoring infrastructure**:
   ```bash
   # Check Prometheus
   curl http://prometheus:9090/-/healthy
   
   # Check Grafana
   curl http://grafana:3000/api/health
   ```

2. **Verify metrics collection**:
   ```bash
   curl https://agentscan-prod.fly.dev/metrics
   ```

#### Resolution Steps

1. **Restart monitoring services**:
   ```bash
   # Restart Prometheus
   docker restart prometheus
   
   # Restart Grafana
   docker restart grafana
   ```

2. **Check configuration**:
   ```bash
   # Validate Prometheus config
   promtool check config prometheus.yml
   ```

### Runbook: Alert Fatigue

**Severity**: Medium
**SLA**: 1 day response, 1 week resolution

#### Investigation Steps

1. **Analyze alert frequency**:
   ```bash
   # Check alert history
   grep "FIRING" /var/log/alertmanager.log | tail -100
   ```

2. **Identify noisy alerts**:
   ```bash
   # Count alerts by type
   grep "FIRING" /var/log/alertmanager.log | awk '{print $5}' | sort | uniq -c | sort -nr
   ```

#### Resolution Steps

1. **Tune alert thresholds**:
   ```yaml
   # Update alert rules
   - alert: HighResponseTime
     expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 5  # Increased from 2
     for: 5m  # Increased from 3m
   ```

2. **Implement alert grouping**:
   ```yaml
   # Group related alerts
   group_by: ['alertname', 'service']
   group_wait: 30s
   group_interval: 5m
   ```

3. **Add alert dependencies**:
   ```yaml
   # Don't alert on dependent services when main service is down
   - alert: DatabaseDown
     expr: up{job="database"} == 0
     for: 1m
   ```

## Escalation Procedures

### Escalation Matrix

| Severity | Initial Response | Escalation Level 1 | Escalation Level 2 | Escalation Level 3 |
|----------|------------------|-------------------|-------------------|-------------------|
| Critical | On-call (5 min) | Team Lead (15 min) | Manager (30 min) | Director (1 hour) |
| High | On-call (15 min) | Team Lead (30 min) | Manager (2 hours) | - |
| Medium | On-call (1 hour) | Team Lead (4 hours) | - | - |
| Low | Business hours | - | - | - |

### Contact Information

- **On-Call Engineer**: [Pager/Phone]
- **Team Lead**: [Contact Info]
- **DevOps Manager**: [Contact Info]
- **Security Team**: [Contact Info]
- **External Support**: [Vendor Contacts]

### Communication Channels

- **Primary**: #agentscan-incidents
- **Escalation**: #agentscan-critical
- **Management**: #leadership-alerts
- **External**: Status page updates

---

**Last Updated**: [Current Date]
**Version**: 1.0.0
**Review Schedule**: Monthly