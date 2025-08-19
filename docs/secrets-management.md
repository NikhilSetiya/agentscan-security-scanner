# Secrets Management with Supabase

AgentScan uses Supabase for secure secrets management, moving sensitive configuration away from environment variables and into a secure, encrypted storage system.

## Overview

The secrets management system provides:

- **Secure Storage**: Secrets are stored encrypted in Supabase
- **Access Control**: Only service role can access secrets
- **Caching**: Secrets are cached for performance
- **Migration Tools**: Easy migration from environment variables
- **Audit Trail**: All secret access is logged

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Application   │    │  Supabase Edge  │    │   Supabase DB   │
│                 │    │   Functions     │    │                 │
│  Secrets Mgr    ├────┤  get-secret     ├────┤  secrets table  │
│                 │    │  set-secret     │    │                 │
│                 │    │  list-secrets   │    │                 │
│                 │    │  delete-secret  │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Setup

### 1. Deploy Edge Functions

Deploy the secrets management Edge Functions to your Supabase project:

```bash
# Deploy all functions
supabase functions deploy get-secret
supabase functions deploy set-secret
supabase functions deploy list-secrets
supabase functions deploy delete-secret
```

### 2. Configure Environment Variables

Add Supabase configuration to your environment:

```bash
# .env
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key-here
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key-here
SUPABASE_ENABLE_SECRETS=true
```

### 3. Migrate Existing Secrets

Use the migration tool to move secrets from environment variables:

```bash
# Build the migration tool
./scripts/build-migrate-secrets.sh

# Preview what will be migrated
./bin/migrate-secrets -dry-run

# Perform the migration
./bin/migrate-secrets
```

## Usage

### Backend Integration

The secrets manager is automatically integrated into the application configuration:

```go
// Load configuration with secrets from Supabase
cfg, err := config.Load()
if err != nil {
    log.Fatal(err)
}

// Secrets are automatically loaded if SUPABASE_ENABLE_SECRETS=true
```

### Manual Secret Management

You can also manage secrets programmatically:

```go
import "github.com/NikhilSetiya/agentscan-security-scanner/internal/secrets"

// Create secrets manager
secretsManager := secrets.NewSupabaseSecretsManager(
    supabaseURL, 
    serviceRoleKey, 
    logger,
)

// Get a secret
value, err := secretsManager.GetSecret(ctx, "JWT_SECRET")

// Set a secret
err = secretsManager.SetSecret(ctx, "API_KEY", "secret-value")

// List secrets
names, err := secretsManager.ListSecrets(ctx)

// Delete a secret
err = secretsManager.DeleteSecret(ctx, "OLD_SECRET")
```

## Supported Secrets

The following secrets are automatically managed:

| Secret Name | Description | Required |
|-------------|-------------|----------|
| `JWT_SECRET` | JWT signing secret | Yes |
| `GITHUB_CLIENT_ID` | GitHub OAuth client ID | Yes |
| `GITHUB_SECRET` | GitHub OAuth client secret | Yes |
| `GITLAB_CLIENT_ID` | GitLab OAuth client ID | Optional |
| `GITLAB_SECRET` | GitLab OAuth client secret | Optional |
| `DB_PASSWORD` | Database password | Yes |
| `REDIS_PASSWORD` | Redis password | Optional |

## Security Features

### Access Control

- Only the service role key can access secrets
- All API calls are authenticated and authorized
- Row Level Security (RLS) prevents unauthorized access

### Encryption

- Secrets are encrypted at rest in Supabase
- Transport encryption via HTTPS
- Service role key provides additional security layer

### Caching

- Secrets are cached for 5 minutes to reduce API calls
- Cache is automatically invalidated when secrets are updated
- Cache can be manually cleared if needed

### Audit Logging

- All secret access is logged with timestamps
- Failed access attempts are recorded
- Logs include operation type and secret name (not value)

## Migration Process

### Step 1: Backup Current Configuration

```bash
# Backup your current .env file
cp .env .env.backup
```

### Step 2: Deploy Supabase Functions

```bash
supabase functions deploy get-secret
supabase functions deploy set-secret
supabase functions deploy list-secrets
supabase functions deploy delete-secret
```

### Step 3: Run Migration

```bash
# Preview migration
./bin/migrate-secrets -dry-run

# Perform migration
./bin/migrate-secrets
```

### Step 4: Update Configuration

```bash
# Enable secrets management
echo "SUPABASE_ENABLE_SECRETS=true" >> .env

# Remove sensitive values from .env (keep keys empty)
sed -i 's/JWT_SECRET=.*/JWT_SECRET=/' .env
sed -i 's/GITHUB_SECRET=.*/GITHUB_SECRET=/' .env
# ... etc for other secrets
```

### Step 5: Test Application

```bash
# Test that the application starts correctly
go run cmd/api/main.go
```

## Troubleshooting

### Common Issues

1. **Function deployment fails**
   - Check Supabase CLI is logged in: `supabase auth login`
   - Verify project is linked: `supabase link`

2. **Migration fails**
   - Check service role key is correct
   - Verify Supabase URL is accessible
   - Ensure functions are deployed

3. **Application can't load secrets**
   - Check `SUPABASE_ENABLE_SECRETS=true`
   - Verify service role key has correct permissions
   - Check function logs in Supabase dashboard

### Debug Commands

```bash
# List current secrets
./bin/migrate-secrets -list

# Check function logs
supabase functions logs get-secret

# Test function directly
curl -X POST https://your-project.supabase.co/functions/v1/get-secret \
  -H "Authorization: Bearer YOUR_SERVICE_ROLE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "JWT_SECRET"}'
```

### Performance Considerations

- Secrets are cached for 5 minutes
- Initial application startup may be slower due to secret loading
- Consider warming up secrets cache during deployment

## Best Practices

### Secret Naming

- Use UPPER_CASE with underscores
- Be descriptive but concise
- Group related secrets with prefixes (e.g., `GITHUB_*`)

### Rotation

- Rotate secrets regularly
- Update secrets in Supabase, not environment variables
- Test application after rotation

### Development vs Production

- Use separate Supabase projects for dev/staging/prod
- Never share production secrets with development
- Use different service role keys for each environment

### Monitoring

- Monitor secret access patterns
- Set up alerts for failed secret retrievals
- Regularly audit secret usage

## Security Considerations

### Access Control

- Limit service role key distribution
- Use environment-specific keys
- Rotate service role keys periodically

### Network Security

- Always use HTTPS for Supabase connections
- Consider IP restrictions for production
- Monitor for unusual access patterns

### Backup and Recovery

- Secrets are automatically backed up by Supabase
- Consider additional backup procedures for critical secrets
- Test recovery procedures regularly

## API Reference

### Get Secret

```http
POST /functions/v1/get-secret
Authorization: Bearer SERVICE_ROLE_KEY
Content-Type: application/json

{
  "name": "SECRET_NAME"
}
```

### Set Secret

```http
POST /functions/v1/set-secret
Authorization: Bearer SERVICE_ROLE_KEY
Content-Type: application/json

{
  "name": "SECRET_NAME",
  "value": "SECRET_VALUE"
}
```

### List Secrets

```http
GET /functions/v1/list-secrets
Authorization: Bearer SERVICE_ROLE_KEY
```

### Delete Secret

```http
DELETE /functions/v1/delete-secret
Authorization: Bearer SERVICE_ROLE_KEY
Content-Type: application/json

{
  "name": "SECRET_NAME"
}
```