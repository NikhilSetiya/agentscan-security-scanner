# Common Issues and Troubleshooting

This guide covers the most common issues encountered when using AgentScan and provides step-by-step solutions.

## Quick Diagnostics

Before diving into specific issues, run these quick diagnostic commands:

```bash
# Check system health
curl http://localhost:8080/health

# Check service status
docker-compose ps

# View recent logs
docker-compose logs --tail=50 api orchestrator

# Check database connectivity
docker-compose exec postgres pg_isready -U agentscan

# Check Redis connectivity
docker-compose exec redis redis-cli ping
```

## Authentication Issues

### Issue: "Authentication required" or 401 Unauthorized

**Symptoms:**
- API requests return 401 status
- Login attempts fail
- Token appears invalid

**Diagnosis:**
```bash
# Test login endpoint
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-password"}'

# Check JWT token validity
echo "YOUR_JWT_TOKEN" | cut -d. -f2 | base64 -d | jq .
```

**Solutions:**

1. **Verify credentials:**
   ```bash
   # Check if user exists in database
   docker-compose exec postgres psql -U agentscan -d agentscan \
     -c "SELECT username, email, role FROM users WHERE username = 'admin';"
   ```

2. **Reset admin password:**
   ```bash
   # Using the CLI tool
   docker-compose exec api ./agentscan user reset-password --username admin

   # Or directly in database
   docker-compose exec postgres psql -U agentscan -d agentscan \
     -c "UPDATE users SET password_hash = crypt('new-password', gen_salt('bf')) WHERE username = 'admin';"
   ```

3. **Check JWT configuration:**
   ```bash
   # Verify JWT secret is set
   docker-compose exec api env | grep JWT_SECRET
   
   # Check token expiration settings
   docker-compose exec api env | grep JWT_EXPIRY
   ```

### Issue: "Session expired" errors

**Symptoms:**
- Frequent re-authentication required
- Token expires quickly
- Session not persisting

**Solutions:**

1. **Adjust token expiration:**
   ```yaml
   # In docker-compose.yml
   environment:
     - JWT_EXPIRY=8h  # Increase from default 1h
   ```

2. **Implement token refresh:**
   ```javascript
   // Frontend token refresh logic
   const refreshToken = async () => {
     try {
       const response = await fetch('/api/v1/auth/refresh', {
         method: 'POST',
         headers: {
           'Authorization': `Bearer ${currentToken}`
         }
       });
       
       if (response.ok) {
         const data = await response.json();
         localStorage.setItem('auth_token', data.token);
         return data.token;
       }
     } catch (error) {
       // Redirect to login
       window.location.href = '/login';
     }
   };
   ```

## Scanning Issues

### Issue: Scans stuck in "queued" or "running" status

**Symptoms:**
- Scans never complete
- Queue appears backed up
- No progress updates

**Diagnosis:**
```bash
# Check orchestrator logs
docker-compose logs orchestrator

# Check queue status
docker-compose exec redis redis-cli LLEN scan_queue

# Check running containers
docker ps | grep agent

# Check system resources
docker stats
```

**Solutions:**

1. **Clear stuck jobs:**
   ```bash
   # Clear Redis queue
   docker-compose exec redis redis-cli FLUSHDB
   
   # Restart orchestrator
   docker-compose restart orchestrator
   ```

2. **Increase worker capacity:**
   ```yaml
   # In docker-compose.yml
   environment:
     - MAX_CONCURRENT_SCANS=10  # Increase from default 5
     - WORKER_POOL_SIZE=20      # Increase worker pool
   ```

3. **Check agent health:**
   ```bash
   # Test individual agents
   curl http://localhost:8080/api/v1/agents/semgrep/health
   curl http://localhost:8080/api/v1/agents/eslint/health
   ```

### Issue: Scans failing with timeout errors

**Symptoms:**
- Scans fail after specific time period
- "Context deadline exceeded" errors
- Large repositories fail consistently

**Solutions:**

1. **Increase scan timeout:**
   ```yaml
   # In docker-compose.yml
   environment:
     - SCAN_TIMEOUT=30m  # Increase from default 10m
     - AGENT_TIMEOUT=15m # Increase agent timeout
   ```

2. **Optimize for large repositories:**
   ```bash
   # Use incremental scanning
   curl -X POST http://localhost:8080/api/v1/scans \
     -H "Authorization: Bearer $TOKEN" \
     -d '{
       "repository_id": "repo-123",
       "scan_type": "incremental",
       "agents": ["semgrep"]
     }'
   ```

3. **Check resource limits:**
   ```yaml
   # In docker-compose.yml
   services:
     orchestrator:
       deploy:
         resources:
           limits:
             memory: 2G
             cpus: '2.0'
   ```

### Issue: Agent execution failures

**Symptoms:**
- Specific agents always fail
- "Agent not available" errors
- Tool-specific error messages

**Diagnosis:**
```bash
# Check agent container logs
docker-compose logs semgrep-agent

# Test agent directly
docker run --rm agentscan/semgrep-agent:latest semgrep --version

# Check agent registration
curl http://localhost:8080/api/v1/agents
```

**Solutions:**

1. **Verify agent images:**
   ```bash
   # Pull latest agent images
   docker-compose pull
   
   # Rebuild if using local images
   docker-compose build --no-cache
   ```

2. **Check agent configuration:**
   ```yaml
   # In config/agents.yaml
   agents:
     semgrep:
       enabled: true
       image: agentscan/semgrep-agent:latest
       timeout: 10m
       resources:
         memory: 1G
         cpu: 1.0
   ```

3. **Test agent manually:**
   ```bash
   # Run agent container manually
   docker run --rm -v $(pwd)/test-repo:/repo \
     agentscan/semgrep-agent:latest \
     semgrep --config=auto /repo
   ```

## Database Issues

### Issue: Database connection failures

**Symptoms:**
- "Connection refused" errors
- "Too many connections" errors
- Slow database queries

**Diagnosis:**
```bash
# Check database status
docker-compose exec postgres pg_isready -U agentscan

# Check connection count
docker-compose exec postgres psql -U agentscan -d agentscan \
  -c "SELECT count(*) FROM pg_stat_activity;"

# Check slow queries
docker-compose exec postgres psql -U agentscan -d agentscan \
  -c "SELECT query, query_start, state FROM pg_stat_activity WHERE state = 'active';"
```

**Solutions:**

1. **Increase connection pool:**
   ```yaml
   # In docker-compose.yml
   environment:
     - DB_MAX_CONNECTIONS=100
     - DB_MAX_IDLE_CONNECTIONS=10
     - DB_CONNECTION_LIFETIME=1h
   ```

2. **Optimize database configuration:**
   ```sql
   -- In PostgreSQL configuration
   ALTER SYSTEM SET max_connections = 200;
   ALTER SYSTEM SET shared_buffers = '256MB';
   ALTER SYSTEM SET effective_cache_size = '1GB';
   SELECT pg_reload_conf();
   ```

3. **Add database monitoring:**
   ```bash
   # Monitor database performance
   docker-compose exec postgres psql -U agentscan -d agentscan \
     -c "SELECT * FROM pg_stat_database WHERE datname = 'agentscan';"
   ```

### Issue: Database migration failures

**Symptoms:**
- Application won't start
- "Migration failed" errors
- Schema version mismatches

**Solutions:**

1. **Check migration status:**
   ```bash
   docker-compose exec api ./agentscan migrate status
   ```

2. **Force migration:**
   ```bash
   # Reset and re-run migrations
   docker-compose exec api ./agentscan migrate reset
   docker-compose exec api ./agentscan migrate up
   ```

3. **Manual migration:**
   ```bash
   # Run specific migration
   docker-compose exec api ./agentscan migrate up --target 20240101000001
   ```

## Performance Issues

### Issue: Slow API responses

**Symptoms:**
- API requests take >2 seconds
- Dashboard loads slowly
- Timeout errors in frontend

**Diagnosis:**
```bash
# Check API response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/api/v1/repositories

# Monitor system resources
docker stats

# Check database query performance
docker-compose exec postgres psql -U agentscan -d agentscan \
  -c "SELECT query, mean_exec_time, calls FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

**Solutions:**

1. **Enable query optimization:**
   ```sql
   -- Add missing indexes
   CREATE INDEX CONCURRENTLY idx_findings_repository_severity 
   ON findings(repository_id, severity);
   
   CREATE INDEX CONCURRENTLY idx_scans_status_created 
   ON scans(status, created_at);
   ```

2. **Implement caching:**
   ```yaml
   # In docker-compose.yml
   environment:
     - REDIS_CACHE_TTL=300  # 5 minutes
     - ENABLE_QUERY_CACHE=true
   ```

3. **Optimize API queries:**
   ```go
   // Use pagination for large datasets
   func (s *ScanService) ListScans(ctx context.Context, opts ListOptions) (*ScanList, error) {
       // Limit default page size
       if opts.Limit == 0 || opts.Limit > 100 {
           opts.Limit = 20
       }
       
       // Use database indexes
       query := `SELECT * FROM scans 
                 WHERE repository_id = $1 
                 ORDER BY created_at DESC 
                 LIMIT $2 OFFSET $3`
       
       // ... execute query
   }
   ```

### Issue: High memory usage

**Symptoms:**
- Out of memory errors
- Container restarts
- System becomes unresponsive

**Solutions:**

1. **Increase memory limits:**
   ```yaml
   # In docker-compose.yml
   services:
     api:
       deploy:
         resources:
           limits:
             memory: 2G
     orchestrator:
       deploy:
         resources:
           limits:
             memory: 4G
   ```

2. **Optimize memory usage:**
   ```go
   // Implement streaming for large results
   func (s *ScanService) StreamResults(ctx context.Context, scanID string, writer io.Writer) error {
       rows, err := s.db.QueryContext(ctx, 
           "SELECT * FROM findings WHERE scan_id = $1 ORDER BY severity DESC", 
           scanID)
       if err != nil {
           return err
       }
       defer rows.Close()
       
       encoder := json.NewEncoder(writer)
       for rows.Next() {
           var finding Finding
           if err := rows.Scan(&finding); err != nil {
               return err
           }
           if err := encoder.Encode(finding); err != nil {
               return err
           }
       }
       
       return rows.Err()
   }
   ```

## Network and Connectivity Issues

### Issue: Cannot connect to external repositories

**Symptoms:**
- "Repository not accessible" errors
- Git clone failures
- Network timeout errors

**Solutions:**

1. **Check network connectivity:**
   ```bash
   # Test from container
   docker-compose exec orchestrator curl -I https://github.com
   
   # Check DNS resolution
   docker-compose exec orchestrator nslookup github.com
   ```

2. **Configure proxy settings:**
   ```yaml
   # In docker-compose.yml
   environment:
     - HTTP_PROXY=http://proxy.company.com:8080
     - HTTPS_PROXY=http://proxy.company.com:8080
     - NO_PROXY=localhost,127.0.0.1
   ```

3. **Add SSH key for private repositories:**
   ```bash
   # Mount SSH keys
   volumes:
     - ~/.ssh:/root/.ssh:ro
   ```

### Issue: WebSocket connection failures

**Symptoms:**
- Real-time updates not working
- WebSocket connection drops
- "Connection refused" errors

**Solutions:**

1. **Check WebSocket endpoint:**
   ```bash
   # Test WebSocket connection
   wscat -c "ws://localhost:8080/api/v1/ws/scans/scan-123?token=YOUR_TOKEN"
   ```

2. **Configure reverse proxy:**
   ```nginx
   # Nginx configuration
   location /api/v1/ws/ {
       proxy_pass http://backend;
       proxy_http_version 1.1;
       proxy_set_header Upgrade $http_upgrade;
       proxy_set_header Connection "upgrade";
       proxy_set_header Host $host;
       proxy_read_timeout 86400;
   }
   ```

## Docker and Container Issues

### Issue: Container startup failures

**Symptoms:**
- Services fail to start
- "Container exited with code 1"
- Port binding errors

**Solutions:**

1. **Check port conflicts:**
   ```bash
   # Check if ports are in use
   lsof -i :8080
   lsof -i :5432
   lsof -i :6379
   
   # Use different ports if needed
   docker-compose -f docker-compose.yml -f docker-compose.override.yml up
   ```

2. **Check resource availability:**
   ```bash
   # Check disk space
   df -h
   
   # Check memory
   free -h
   
   # Check Docker daemon
   docker system info
   ```

3. **Review container logs:**
   ```bash
   # Check specific service logs
   docker-compose logs api
   docker-compose logs orchestrator
   docker-compose logs postgres
   ```

### Issue: Docker build failures

**Symptoms:**
- "Build failed" errors
- Dependency installation failures
- Image size too large

**Solutions:**

1. **Clear Docker cache:**
   ```bash
   # Clear build cache
   docker builder prune -a
   
   # Remove unused images
   docker image prune -a
   ```

2. **Optimize Dockerfile:**
   ```dockerfile
   # Use multi-stage builds
   FROM golang:1.21-alpine AS builder
   WORKDIR /app
   COPY go.mod go.sum ./
   RUN go mod download
   COPY . .
   RUN go build -o agentscan ./cmd/api
   
   FROM alpine:latest
   RUN apk --no-cache add ca-certificates
   COPY --from=builder /app/agentscan /usr/local/bin/
   CMD ["agentscan"]
   ```

## Configuration Issues

### Issue: Environment variables not loading

**Symptoms:**
- Default values used instead of configured values
- "Configuration not found" errors
- Services using wrong settings

**Solutions:**

1. **Verify environment file:**
   ```bash
   # Check .env file exists and is readable
   ls -la .env
   cat .env | grep -v "^#" | grep -v "^$"
   ```

2. **Check Docker Compose configuration:**
   ```yaml
   # Ensure env_file is specified
   services:
     api:
       env_file:
         - .env
       environment:
         - DEBUG=true
   ```

3. **Validate configuration loading:**
   ```bash
   # Check loaded environment in container
   docker-compose exec api env | sort
   ```

### Issue: Agent configuration problems

**Symptoms:**
- Agents not detected
- Wrong tool versions
- Configuration not applied

**Solutions:**

1. **Verify agent configuration:**
   ```bash
   # Check agent config file
   cat config/agents.yaml
   
   # Validate YAML syntax
   python -c "import yaml; yaml.safe_load(open('config/agents.yaml'))"
   ```

2. **Test agent registration:**
   ```bash
   # Check registered agents
   curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/v1/agents
   ```

3. **Update agent images:**
   ```bash
   # Pull latest agent images
   docker pull agentscan/semgrep-agent:latest
   docker pull agentscan/eslint-agent:latest
   
   # Restart services
   docker-compose restart orchestrator
   ```

## Performance Issues

### Issue: Slow scan execution

**Symptoms:**
- Scans take much longer than expected
- High CPU/memory usage
- System becomes unresponsive

**Diagnosis:**
```bash
# Monitor resource usage
docker stats

# Check scan queue length
docker-compose exec redis redis-cli LLEN scan_queue

# Profile application performance
docker-compose exec api go tool pprof http://localhost:6060/debug/pprof/profile
```

**Solutions:**

1. **Optimize scan configuration:**
   ```yaml
   # Reduce concurrent scans
   environment:
     - MAX_CONCURRENT_SCANS=3
     - AGENT_TIMEOUT=5m
   ```

2. **Use incremental scanning:**
   ```bash
   # Submit incremental scan
   curl -X POST http://localhost:8080/api/v1/scans \
     -H "Authorization: Bearer $TOKEN" \
     -d '{
       "repository_id": "repo-123",
       "scan_type": "incremental"
     }'
   ```

3. **Add resource monitoring:**
   ```yaml
   # Add monitoring service
   services:
     prometheus:
       image: prom/prometheus:latest
       ports:
         - "9090:9090"
       volumes:
         - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
   ```

## Data Issues

### Issue: Missing or corrupted scan results

**Symptoms:**
- Scan shows as completed but no results
- Findings count doesn't match displayed results
- Database integrity errors

**Solutions:**

1. **Check database integrity:**
   ```sql
   -- Check for orphaned records
   SELECT s.id, s.status, COUNT(f.id) as findings_count 
   FROM scans s 
   LEFT JOIN findings f ON s.id = f.scan_id 
   WHERE s.status = 'completed' 
   GROUP BY s.id, s.status 
   HAVING COUNT(f.id) = 0;
   
   -- Check for data consistency
   SELECT COUNT(*) FROM findings WHERE scan_id NOT IN (SELECT id FROM scans);
   ```

2. **Repair data inconsistencies:**
   ```bash
   # Run data repair script
   docker-compose exec api ./agentscan repair --check-integrity
   
   # Rebuild finding counts
   docker-compose exec api ./agentscan repair --rebuild-counts
   ```

3. **Restore from backup:**
   ```bash
   # List available backups
   ./scripts/disaster-recovery.sh list-backups --backup-bucket agentscan-backups
   
   # Restore specific backup
   ./scripts/disaster-recovery.sh restore \
     --backup-bucket agentscan-backups \
     --timestamp 20240101-120000
   ```

## Integration Issues

### Issue: GitHub integration not working

**Symptoms:**
- Webhooks not received
- PR comments not posted
- Repository access denied

**Solutions:**

1. **Verify GitHub App configuration:**
   ```bash
   # Check GitHub App settings
   curl -H "Authorization: Bearer $GITHUB_TOKEN" \
     https://api.github.com/app
   
   # Test webhook endpoint
   curl -X POST http://localhost:8080/webhooks/github \
     -H "Content-Type: application/json" \
     -d '{"action": "opened", "pull_request": {...}}'
   ```

2. **Update webhook URL:**
   ```bash
   # Update webhook URL in GitHub App settings
   # URL should be: https://your-domain.com/webhooks/github
   ```

3. **Check permissions:**
   ```bash
   # Verify GitHub App has required permissions:
   # - Repository: Read & Write
   # - Pull requests: Write
   # - Checks: Write
   ```

### Issue: VS Code extension not connecting

**Symptoms:**
- Extension shows "disconnected"
- No real-time updates
- Authentication failures

**Solutions:**

1. **Check extension configuration:**
   ```json
   // In VS Code settings.json
   {
     "agentscan.apiUrl": "http://localhost:8080/api/v1",
     "agentscan.websocketUrl": "ws://localhost:8080/api/v1/ws",
     "agentscan.autoScan": true
   }
   ```

2. **Verify API connectivity:**
   ```bash
   # Test from VS Code terminal
   curl http://localhost:8080/health
   ```

3. **Check extension logs:**
   - Open VS Code Developer Tools (Help > Toggle Developer Tools)
   - Check Console for AgentScan extension errors

## Monitoring and Alerting Issues

### Issue: Alerts not firing

**Symptoms:**
- No notifications for critical findings
- Monitoring dashboards empty
- Prometheus targets down

**Solutions:**

1. **Check Prometheus configuration:**
   ```bash
   # Verify Prometheus targets
   curl http://localhost:9090/api/v1/targets
   
   # Check alert rules
   curl http://localhost:9090/api/v1/rules
   ```

2. **Test notification channels:**
   ```bash
   # Test Slack webhook
   curl -X POST YOUR_SLACK_WEBHOOK_URL \
     -H "Content-Type: application/json" \
     -d '{"text": "Test notification from AgentScan"}'
   
   # Test email notifications
   docker-compose exec api ./agentscan notify test --email your@email.com
   ```

3. **Verify metrics collection:**
   ```bash
   # Check metrics endpoint
   curl http://localhost:8080/metrics
   
   # Verify Grafana data sources
   curl -u admin:admin http://localhost:3000/api/datasources
   ```

## Getting Help

### Log Collection

When reporting issues, collect relevant logs:

```bash
#!/bin/bash
# collect-logs.sh

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
LOG_DIR="agentscan-logs-$TIMESTAMP"

mkdir -p "$LOG_DIR"

# Collect service logs
docker-compose logs --no-color api > "$LOG_DIR/api.log"
docker-compose logs --no-color orchestrator > "$LOG_DIR/orchestrator.log"
docker-compose logs --no-color postgres > "$LOG_DIR/postgres.log"
docker-compose logs --no-color redis > "$LOG_DIR/redis.log"

# Collect system information
docker-compose ps > "$LOG_DIR/services.txt"
docker stats --no-stream > "$LOG_DIR/stats.txt"
docker system df > "$LOG_DIR/disk-usage.txt"

# Collect configuration
cp .env "$LOG_DIR/env.txt" 2>/dev/null || echo "No .env file" > "$LOG_DIR/env.txt"
cp docker-compose.yml "$LOG_DIR/"
cp config/agents.yaml "$LOG_DIR/" 2>/dev/null || echo "No agents config"

# Create archive
tar -czf "agentscan-logs-$TIMESTAMP.tar.gz" "$LOG_DIR"
rm -rf "$LOG_DIR"

echo "Logs collected in agentscan-logs-$TIMESTAMP.tar.gz"
```

### Health Check Script

```bash
#!/bin/bash
# health-check.sh

echo "🏥 AgentScan Health Check"
echo "========================"

# Check services
echo "📋 Service Status:"
docker-compose ps

echo -e "\n🌐 API Health:"
curl -s http://localhost:8080/health | jq .

echo -e "\n💾 Database Health:"
docker-compose exec -T postgres pg_isready -U agentscan

echo -e "\n🔄 Redis Health:"
docker-compose exec -T redis redis-cli ping

echo -e "\n🤖 Agent Health:"
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/api/v1/agents | jq '.[] | {name: .name, status: .status}'

echo -e "\n📊 System Resources:"
docker stats --no-stream

echo -e "\n💿 Disk Usage:"
docker system df

echo -e "\n🔍 Recent Errors:"
docker-compose logs --tail=10 | grep -i error || echo "No recent errors"
```

### Support Channels

- **Documentation**: https://docs.agentscan.dev
- **GitHub Issues**: https://github.com/agentscan/agentscan/issues
- **Community Forum**: https://community.agentscan.dev
- **Email Support**: support@agentscan.dev
- **Emergency**: critical@agentscan.dev (production issues only)

### Information to Include

When reporting issues, please include:

1. **Environment details:**
   - AgentScan version
   - Operating system
   - Docker version
   - Hardware specifications

2. **Configuration:**
   - docker-compose.yml (sanitized)
   - Environment variables (sanitized)
   - Agent configuration

3. **Error details:**
   - Complete error messages
   - Relevant log excerpts
   - Steps to reproduce

4. **System state:**
   - Service status
   - Resource usage
   - Recent changes

This information helps our support team diagnose and resolve issues quickly.