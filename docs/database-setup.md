# Database Setup Guide

This guide explains how to set up and manage the AgentScan database.

## Prerequisites

- PostgreSQL 15+
- Go 1.21+
- Docker and Docker Compose (for development)

## Development Setup

### Using Docker Compose (Recommended)

1. Start the development environment:
```bash
docker-compose up -d postgres redis
```

2. Wait for PostgreSQL to be ready:
```bash
docker-compose logs postgres
```

3. Run migrations:
```bash
make migrate-up
```

### Manual PostgreSQL Setup

1. Install PostgreSQL 15+
2. Create database and user:
```sql
CREATE DATABASE agentscan;
CREATE USER agentscan WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE agentscan TO agentscan;
```

3. Set environment variables:
```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=agentscan
export DB_USER=agentscan
export DB_PASSWORD=your_password
export JWT_SECRET=your_jwt_secret
export GITHUB_CLIENT_ID=your_github_client_id
export GITHUB_SECRET=your_github_secret
```

4. Run migrations:
```bash
go run cmd/migrate/main.go up
```

## Migration Commands

### Run all migrations
```bash
make migrate-up
# or
go run cmd/migrate/main.go up
```

### Rollback all migrations
```bash
make migrate-down
# or
go run cmd/migrate/main.go down
```

### Check migration version
```bash
make migrate-version
# or
go run cmd/migrate/main.go version
```

### Run specific number of migrations
```bash
go run cmd/migrate/main.go steps 1    # Run 1 migration up
go run cmd/migrate/main.go steps -1   # Rollback 1 migration
```

### Force migration version (use with caution)
```bash
make migrate-force VERSION=1
# or
go run cmd/migrate/main.go force 1
```

## Database Schema

The database schema includes the following main tables:

- `users` - User accounts and authentication
- `organizations` - Organizations/teams
- `organization_members` - User membership in organizations
- `repositories` - Code repositories
- `scan_jobs` - Security scan jobs
- `scan_results` - Results from individual agents
- `findings` - Security vulnerabilities found
- `user_feedback` - User feedback for ML training

## Testing

### Unit Tests
```bash
go test ./internal/database
```

### Integration Tests
```bash
# Set up test database first
export INTEGRATION_TESTS=1
export TEST_DB_NAME=agentscan_test
make test-integration
```

## Production Setup

### Database Configuration

For production, ensure:

1. Use strong passwords
2. Enable SSL connections
3. Configure connection pooling appropriately
4. Set up regular backups
5. Monitor database performance

### Environment Variables

Required environment variables for production:

```bash
DB_HOST=your_db_host
DB_PORT=5432
DB_NAME=agentscan
DB_USER=agentscan
DB_PASSWORD=strong_password_here
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

### Backup Strategy

Set up regular database backups:

```bash
# Daily backup
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME > backup_$(date +%Y%m%d).sql

# Restore from backup
psql -h $DB_HOST -U $DB_USER -d $DB_NAME < backup_20240101.sql
```

## Troubleshooting

### Common Issues

1. **Connection refused**
   - Check if PostgreSQL is running
   - Verify host and port settings
   - Check firewall settings

2. **Authentication failed**
   - Verify username and password
   - Check pg_hba.conf configuration
   - Ensure user has proper permissions

3. **Migration errors**
   - Check database connectivity
   - Verify migration files exist
   - Check for dirty migration state

4. **Performance issues**
   - Monitor connection pool usage
   - Check for missing indexes
   - Analyze slow queries

### Useful Commands

```bash
# Check database connection
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT version();"

# List all tables
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "\dt"

# Check migration status
go run cmd/migrate/main.go version

# View database stats
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT * FROM pg_stat_database WHERE datname = 'agentscan';"
```