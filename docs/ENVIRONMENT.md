# Environment Variables

This document describes all environment variables used by AgentScan Security Scanner.

## Overview

AgentScan uses environment variables for configuration to follow the [12-factor app methodology](https://12factor.net/config). This allows for easy deployment across different environments without code changes.

## Environment Files

The application loads environment variables from the following files in order of precedence:

1. `.env.local` (highest priority, git-ignored)
2. `.env.{environment}` (e.g., `.env.production`, `.env.development`)
3. `.env` (default environment file)
4. System environment variables (lowest priority)

## Quick Start

1. Copy the template file:
   ```bash
   cp .env.template .env
   ```

2. Update the values in `.env` according to your environment

3. Validate your configuration:
   ```bash
   go run cmd/env-check/main.go -validate
   ```

## Application

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `APP_NAME` | string | No | `AgentScan` | Application name | |
| `APP_VERSION` | string | No | `1.0.0` | Application version | |
| `GO_ENV` | string | No | `development` | Application environment (development, production, etc.) | |
| `APP_DEBUG` | bool | No | `false` | Enable debug mode | |

## Server

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `HOST` | string | No | `0.0.0.0` | Server host address | |
| `PORT` | int | No | `8080` | Server port number | port |
| `READ_TIMEOUT` | duration | No | `30s` | HTTP read timeout | |
| `WRITE_TIMEOUT` | duration | No | `30s` | HTTP write timeout | |
| `IDLE_TIMEOUT` | duration | No | `60s` | HTTP idle timeout | |
| `SHUTDOWN_TIMEOUT` | duration | No | `30s` | Graceful shutdown timeout | |

## Database

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `DATABASE_URL` | string | **Yes** | | Database connection URL | url |
| `DATABASE_MAX_OPEN_CONNS` | int | No | `25` | Maximum number of open database connections | positive |
| `DATABASE_MAX_IDLE_CONNS` | int | No | `5` | Maximum number of idle database connections | positive |
| `DATABASE_CONN_MAX_LIFETIME` | duration | No | `5m` | Maximum connection lifetime | |
| `DATABASE_CONN_MAX_IDLE_TIME` | duration | No | `5m` | Maximum connection idle time | |
| `DATABASE_SSL_MODE` | string | No | `require` | Database SSL mode | |

### Database URL Format

```
postgresql://username:password@host:port/database?sslmode=require
```

Example:
```
DATABASE_URL=postgresql://agentscan:password@localhost:5432/agentscan_production?sslmode=require
```

## Redis

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `REDIS_URL` | string | No | `redis://localhost:6379/0` | Redis connection URL | url |
| `REDIS_PASSWORD` | string | No | | Redis password | |
| `REDIS_MAX_RETRIES` | int | No | `3` | Maximum retry attempts | |
| `REDIS_POOL_SIZE` | int | No | `10` | Connection pool size | |
| `REDIS_MIN_IDLE_CONNS` | int | No | `5` | Minimum idle connections | |
| `REDIS_POOL_TIMEOUT` | duration | No | `4s` | Pool timeout | |
| `REDIS_IDLE_TIMEOUT` | duration | No | `5m` | Idle connection timeout | |

## Security

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `JWT_SECRET` | string | **Yes** | | JWT signing secret | min_length:32 |
| `JWT_ALGORITHM` | string | No | `HS256` | JWT signing algorithm | |
| `JWT_ACCESS_TOKEN_TTL` | duration | No | `15m` | Access token lifetime | |
| `JWT_REFRESH_TOKEN_TTL` | duration | No | `168h` | Refresh token lifetime | |
| `JWT_ISSUER` | string | No | `agentscan.io` | JWT issuer | |
| `JWT_AUDIENCE` | string | No | `agentscan.io` | JWT audience | |

### HTTPS Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `HTTPS_ENABLED` | bool | No | `true` | Enable HTTPS |
| `HTTPS_PORT` | int | No | `443` | HTTPS port |
| `TLS_CERT_FILE` | string | No | `/etc/ssl/certs/server.crt` | TLS certificate file |
| `TLS_KEY_FILE` | string | No | `/etc/ssl/private/server.key` | TLS private key file |
| `HTTPS_REDIRECT_HTTP` | bool | No | `true` | Redirect HTTP to HTTPS |
| `HSTS_MAX_AGE` | int | No | `31536000` | HSTS max age (seconds) |
| `HSTS_INCLUDE_SUBDOMAINS` | bool | No | `true` | Include subdomains in HSTS |
| `HSTS_PRELOAD` | bool | No | `true` | Enable HSTS preload |

### CORS Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `CORS_ENABLED` | bool | No | `true` | Enable CORS |
| `CORS_ALLOWED_ORIGINS` | string | No | | Comma-separated list of allowed origins |
| `CORS_ALLOWED_METHODS` | string | No | `GET,POST,PUT,DELETE,OPTIONS,PATCH` | Allowed HTTP methods |
| `CORS_ALLOWED_HEADERS` | string | No | | Allowed headers |
| `CORS_ALLOW_CREDENTIALS` | bool | No | `true` | Allow credentials |
| `CORS_MAX_AGE` | int | No | `86400` | Preflight cache duration |

## External Services

### Supabase

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `SUPABASE_URL` | string | **Yes** | | Supabase project URL | url |
| `SUPABASE_SERVICE_ROLE_KEY` | string | **Yes** | | Supabase service role key | |
| `SUPABASE_JWT_SECRET` | string | No | | Supabase JWT secret | |
| `SUPABASE_DB_URL` | string | No | | Supabase database URL | |

### GitHub Integration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `GITHUB_CLIENT_ID` | string | No | | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | string | No | | GitHub OAuth client secret |
| `GITHUB_WEBHOOK_SECRET` | string | No | | GitHub webhook secret |

### Email (SMTP)

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `SMTP_HOST` | string | No | | SMTP server host |
| `SMTP_PORT` | int | No | `587` | SMTP server port |
| `SMTP_USERNAME` | string | No | | SMTP username |
| `SMTP_PASSWORD` | string | No | | SMTP password |
| `SMTP_FROM` | string | No | | From email address |
| `SMTP_TLS` | bool | No | `true` | Enable TLS |

## Monitoring

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `METRICS_ENABLED` | bool | No | `true` | Enable Prometheus metrics | |
| `METRICS_PORT` | int | No | `9090` | Metrics server port | port |
| `METRICS_PATH` | string | No | `/metrics` | Metrics endpoint path | |
| `HEALTH_CHECK_ENABLED` | bool | No | `true` | Enable health checks | |
| `HEALTH_CHECK_PATH` | string | No | `/health` | Health check endpoint | |
| `PPROF_ENABLED` | bool | No | `false` | Enable pprof endpoints | |

### Logging

| Variable | Type | Required | Default | Description | Validation |
|----------|------|----------|---------|-------------|------------|
| `LOG_LEVEL` | string | No | `info` | Logging level | log_level |
| `LOG_FORMAT` | string | No | `json` | Log format (json, text) | |
| `LOG_OUTPUT` | string | No | `stdout` | Log output (stdout, file) | |
| `LOG_FILE_PATH` | string | No | `/var/log/agentscan/app.log` | Log file path | |
| `LOG_MAX_SIZE` | int | No | `100` | Max log file size (MB) | |
| `LOG_MAX_BACKUPS` | int | No | `5` | Max log file backups | |
| `LOG_MAX_AGE` | int | No | `30` | Max log file age (days) | |
| `LOG_COMPRESS` | bool | No | `true` | Compress old log files | |

## Rate Limiting

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `RATE_LIMIT_ENABLED` | bool | No | `true` | Enable rate limiting |
| `RATE_LIMIT_GLOBAL_REQUESTS` | int | No | `1000` | Global requests per window |
| `RATE_LIMIT_GLOBAL_WINDOW` | duration | No | `1h` | Global rate limit window |
| `RATE_LIMIT_GLOBAL_BURST` | int | No | `100` | Global burst limit |

### Endpoint-Specific Rate Limits

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `RATE_LIMIT_LOGIN_REQUESTS` | int | `5` | Login requests per window |
| `RATE_LIMIT_LOGIN_WINDOW` | duration | `15m` | Login rate limit window |
| `RATE_LIMIT_REGISTER_REQUESTS` | int | `3` | Registration requests per window |
| `RATE_LIMIT_REGISTER_WINDOW` | duration | `1h` | Registration rate limit window |
| `RATE_LIMIT_SCANS_REQUESTS` | int | `10` | Scan requests per window |
| `RATE_LIMIT_SCANS_WINDOW` | duration | `1h` | Scan rate limit window |

## Performance

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `CACHE_TTL` | duration | No | `1h` | Default cache TTL |
| `CACHE_CLEANUP_INTERVAL` | duration | No | `10m` | Cache cleanup interval |
| `MAX_CONCURRENT_SCANS` | int | No | `5` | Maximum concurrent scans |
| `SCAN_TIMEOUT` | duration | No | `10m` | Scan timeout |
| `WORKER_CONCURRENCY` | int | No | `10` | Background worker concurrency |
| `WORKER_QUEUE_SIZE` | int | No | `1000` | Worker queue size |
| `JOB_TIMEOUT` | duration | No | `5m` | Job timeout |
| `JOB_RETRY_ATTEMPTS` | int | No | `3` | Job retry attempts |
| `JOB_RETRY_DELAY` | duration | No | `30s` | Job retry delay |

## Content Security Policy

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CSP_ENABLED` | bool | `true` | Enable CSP |
| `CSP_DEFAULT_SRC` | string | `'self'` | Default source directive |
| `CSP_SCRIPT_SRC` | string | | Script source directive |
| `CSP_STYLE_SRC` | string | | Style source directive |
| `CSP_IMG_SRC` | string | | Image source directive |
| `CSP_CONNECT_SRC` | string | | Connect source directive |
| `CSP_FONT_SRC` | string | | Font source directive |
| `CSP_OBJECT_SRC` | string | `'none'` | Object source directive |
| `CSP_MEDIA_SRC` | string | | Media source directive |
| `CSP_FRAME_SRC` | string | `'none'` | Frame source directive |
| `CSP_REPORT_URI` | string | `/api/v1/csp-report` | CSP report URI |
| `CSP_REPORT_ONLY` | bool | `false` | CSP report-only mode |

## Environment-Specific Examples

### Development

```bash
# Development environment
GO_ENV=development
APP_DEBUG=true
LOG_LEVEL=debug
HTTPS_ENABLED=false
METRICS_ENABLED=true
PPROF_ENABLED=true

# Local database
DATABASE_URL=postgresql://agentscan:password@localhost:5432/agentscan_dev?sslmode=disable

# Local Redis
REDIS_URL=redis://localhost:6379/0

# Development Supabase project
SUPABASE_URL=https://your-dev-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your_dev_service_role_key

# Weak JWT secret for development (use strong secret in production)
JWT_SECRET=development_jwt_secret_at_least_32_characters_long
```

### Production

```bash
# Production environment
GO_ENV=production
APP_DEBUG=false
LOG_LEVEL=info
HTTPS_ENABLED=true
METRICS_ENABLED=true
PPROF_ENABLED=false

# Production database with SSL
DATABASE_URL=postgresql://agentscan:secure_password@db.example.com:5432/agentscan_prod?sslmode=require

# Production Redis with password
REDIS_URL=redis://redis.example.com:6379/0
REDIS_PASSWORD=secure_redis_password

# Production Supabase project
SUPABASE_URL=https://your-prod-project.supabase.co
SUPABASE_SERVICE_ROLE_KEY=your_production_service_role_key

# Strong JWT secret for production
JWT_SECRET=very_long_and_secure_jwt_secret_for_production_use_at_least_32_characters

# HTTPS configuration
TLS_CERT_FILE=/etc/ssl/certs/agentscan.crt
TLS_KEY_FILE=/etc/ssl/private/agentscan.key

# CORS for production frontend
CORS_ALLOWED_ORIGINS=https://app.agentscan.io,https://agentscan.vercel.app
```

## Security Considerations

### Required in Production

These variables MUST be set in production:

- `DATABASE_URL` - Database connection
- `SUPABASE_URL` - Supabase project URL
- `SUPABASE_SERVICE_ROLE_KEY` - Supabase authentication
- `JWT_SECRET` - Must be at least 32 characters

### Sensitive Variables

Never commit these to version control:

- `JWT_SECRET`
- `SUPABASE_SERVICE_ROLE_KEY`
- `DATABASE_URL` (contains password)
- `REDIS_PASSWORD`
- `GITHUB_CLIENT_SECRET`
- `SMTP_PASSWORD`

### Production Security

In production, ensure:

- `HTTPS_ENABLED=true`
- `APP_DEBUG=false`
- `PPROF_ENABLED=false`
- `LOG_LEVEL=info` (not debug)
- Strong passwords and secrets
- Proper SSL/TLS certificates

## Validation

Use the environment validation tool:

```bash
# Check environment status
go run cmd/env-check/main.go

# Validate configuration
go run cmd/env-check/main.go -validate

# Generate template
go run cmd/env-check/main.go -generate

# Generate documentation
go run cmd/env-check/main.go -docs
```

## Troubleshooting

### Common Issues

1. **Missing required variables**: Use `-validate` to check
2. **Invalid URL format**: Ensure URLs include protocol (http://, https://, etc.)
3. **Invalid port numbers**: Must be between 1-65535
4. **JWT secret too short**: Must be at least 32 characters
5. **Database connection fails**: Check URL format and credentials

### Debug Mode

Enable debug mode for troubleshooting:

```bash
APP_DEBUG=true
LOG_LEVEL=debug
```

This will provide detailed logging and error information.

## Migration Guide

When upgrading between versions, check for new environment variables:

1. Compare your `.env` with the latest `.env.template`
2. Run validation to check for missing variables
3. Update configuration as needed
4. Test in development before deploying to production