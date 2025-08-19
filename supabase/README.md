# AgentScan Supabase Configuration

This directory contains the Supabase configuration and database schema for the AgentScan Security Scanner.

## Setup Instructions

### 1. Install Supabase CLI

```bash
# Using npm
npm install -g supabase

# Using Homebrew (macOS)
brew install supabase/tap/supabase

# Using Scoop (Windows)
scoop bucket add supabase https://github.com/supabase/scoop-bucket.git
scoop install supabase
```

### 2. Local Development Setup

Run the setup script from the project root:

```bash
./scripts/setup-supabase.sh
```

This will:
- Initialize the Supabase project
- Start local Supabase services
- Apply database migrations
- Display configuration details

### 3. Manual Setup (Alternative)

If you prefer manual setup:

```bash
# Initialize Supabase
supabase init

# Start local services
supabase start

# Apply migrations
supabase db reset

# Check status
supabase status
```

### 4. Production Setup

1. Create a new project at [supabase.com](https://supabase.com)
2. Note your project URL and anon key
3. Apply the migrations to your production database:
   ```bash
   supabase db push --linked
   ```
4. Update your production environment variables

## Database Schema

The database schema includes:

- **users**: User profiles linked to Supabase auth
- **repositories**: User repositories for scanning
- **scans**: Security scan records
- **findings**: Security vulnerabilities found in scans
- **finding_suppressions**: User-defined suppressions for findings
- **user_feedback**: User feedback on findings

## Row Level Security (RLS)

All tables have Row Level Security enabled to ensure users can only access their own data:

- Users can only see their own profile and data
- Repository access is restricted to the owner
- Scan results are only visible to the repository owner
- Findings are only accessible through owned scans

## Environment Variables

### Frontend (.env.development, .env.production)
```bash
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key-here
```

### Backend (.env)
```bash
SUPABASE_URL=https://your-project.supabase.co
SUPABASE_ANON_KEY=your-anon-key-here
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key-here
```

## Local Development URLs

When running locally with `supabase start`:

- **Supabase Studio**: http://localhost:54323
- **API**: http://localhost:54321
- **Database**: postgresql://postgres:postgres@localhost:54322/postgres
- **Inbucket (Email testing)**: http://localhost:54324

## Migrations

Database migrations are stored in `supabase/migrations/`. To create a new migration:

```bash
supabase migration new migration_name
```

To apply migrations:

```bash
# Local
supabase db reset

# Production (after linking)
supabase db push
```

## Authentication Configuration

The project is configured to support:

- Email/password authentication
- GitHub OAuth (configured in config.toml)
- GitLab OAuth (configured in config.toml)

OAuth providers require environment variables:
- `SUPABASE_AUTH_EXTERNAL_GITHUB_CLIENT_ID`
- `SUPABASE_AUTH_EXTERNAL_GITHUB_SECRET`
- `SUPABASE_AUTH_EXTERNAL_GITLAB_CLIENT_ID`
- `SUPABASE_AUTH_EXTERNAL_GITLAB_SECRET`

## Functions

The schema includes several PostgreSQL functions:

- `handle_new_user()`: Automatically creates user profile when auth user is created
- `get_dashboard_stats(user_uuid)`: Returns dashboard statistics for a user
- `update_updated_at_column()`: Automatically updates timestamp columns

## Troubleshooting

### Common Issues

1. **Port conflicts**: If ports are in use, stop other services or modify `config.toml`
2. **Migration errors**: Check the migration files for syntax errors
3. **RLS issues**: Ensure you're authenticated when testing queries

### Useful Commands

```bash
# Check status
supabase status

# Stop services
supabase stop

# View logs
supabase logs

# Reset database
supabase db reset

# Generate types (for TypeScript)
supabase gen types typescript --local > types/supabase.ts
```

## Security Notes

- Never commit real API keys to version control
- Use environment variables for all sensitive configuration
- The service role key has admin access - use carefully
- RLS policies are enforced for all user data access
- All user inputs should be validated and sanitized