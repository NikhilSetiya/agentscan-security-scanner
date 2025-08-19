# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AgentScan is a multi-agent security scanning platform built with Go and React that orchestrates multiple security tools to reduce false positives by 80%. The platform includes:

- **API Server**: RESTful API with JWT authentication (Go/Gin)
- **Orchestrator**: Job scheduling and agent coordination (Go)  
- **Web Dashboard**: Modern React frontend with real-time updates
- **VS Code Extension**: Real-time IDE integration with TypeScript
- **Security Agents**: Containerized tools (Semgrep, ESLint, Bandit, etc.)
- **GitHub Action**: Marketplace-ready CI/CD integration with SARIF support
- **Viral Growth Engine**: Badge generation and automated PR system for organic discovery
- **Freemium Billing**: Usage tracking with 4-tier subscription model (Free, Pro, Team, Enterprise)

## Development Commands

### Core Build & Development
```bash
# Build all components
make build

# Build individual components  
make build-api           # API server
make build-orchestrator  # Orchestrator service
make build-cli          # CLI tool

# Development with hot reload
make dev-api            # Start API in development mode
make dev-orchestrator   # Start orchestrator in development mode
```

### Testing
```bash
# Run all tests
make test

# Run specific test types
make test-cover         # Tests with coverage report
make test-integration   # Integration tests only
./tests/run_all_tests.sh # Comprehensive test suite

# Frontend tests
cd web/frontend && npm test        # React component tests
cd web/frontend && npm run test:watch # Watch mode

# VS Code extension tests  
cd vscode-extension && npm test

# End-to-end tests
cd tests/e2e && npm test
```

### Code Quality & Linting
```bash
# Lint and format Go code
make lint               # Run golangci-lint
make fmt               # Format and organize imports

# Frontend linting
cd web/frontend && npm run lint
cd vscode-extension && npm run lint
```

### Database & Infrastructure
```bash
# Start development environment
make docker-up          # Start all services with docker-compose
make docker-down        # Stop all services

# Database migrations  
make migrate-up         # Run database migrations
make migrate-down       # Rollback migrations
make migrate-version    # Check current migration version
```

### Deployment & Production
```bash
# Docker builds
make docker-build       # Build all Docker images
make docker-build-api   # Build API Docker image  
make docker-build-orchestrator # Build orchestrator image

# Deploy to production (DigitalOcean)
./scripts/deploy.sh deploy
```

### Development Dependencies
```bash
# Install development tools
make install-deps       # Go tools (golangci-lint, goimports)
cd web/frontend && npm install  # Frontend dependencies
cd vscode-extension && npm install # Extension dependencies
```

## Architecture Overview

### Core Services
- **API Server** (`cmd/api`): Handles REST endpoints, authentication, rate limiting
- **Orchestrator** (`cmd/orchestrator`): Manages scan jobs, agent coordination, consensus algorithms
- **CLI** (`cmd/cli`): Command-line interface for scan operations
- **Beta Inviter** (`cmd/beta-inviter`): Automated user onboarding system
- **Viral Growth** (`cmd/viral-growth`): Automated GitHub PR generation and repository discovery for organic marketing
- **GitHub Action** (`cmd/github-action`): Marketplace-ready CI/CD integration with SARIF output
- **Migration Tool** (`cmd/migrate`): Database schema migration management

### Key Internal Packages
- **`internal/api`**: HTTP handlers, middleware, routing logic
- **`internal/orchestrator`**: Job scheduling, agent management, workflow orchestration
- **`internal/database`**: Repository pattern, database operations, migrations
- **`internal/auth`**: JWT authentication, OAuth, RBAC, user management
- **`internal/consensus`**: Multi-agent consensus algorithms, machine learning integration
- **`internal/queue`**: Redis-based job queues, worker management
- **`internal/github`**: GitHub API integration, webhook handling, PR management
- **`internal/findings`**: Vulnerability finding processing, export functionality
- **`internal/badges`**: Dynamic security badge generation system for viral growth
- **`internal/billing`**: Freemium model implementation with usage tracking and plan limits
- **`internal/analytics`**: Conversion funnel tracking and user journey analytics
- **`internal/gitlab`**: GitLab API integration and webhook handling

### Security Agents Structure
All agents follow the `pkg/agent.SecurityAgent` interface:

```go
type SecurityAgent interface {
    Scan(ctx context.Context, config ScanConfig) (*ScanResult, error)
    HealthCheck(ctx context.Context) error
    GetConfig() AgentConfig
    GetVersion() VersionInfo
}
```

Agent categories:
- **SAST** (`agents/sast/`): Static analysis (Semgrep, ESLint, Bandit, Gosec)
- **SCA** (`agents/sca/`): Dependency scanning (npm-audit, pip-audit, govulncheck)
- **Secrets** (`agents/secrets/`): Secret detection (TruffleHog, git-secrets)
- **DAST** (`agents/dast/`): Dynamic analysis (OWASP ZAP)

### Frontend Structure
- **Dashboard** (`web/frontend/`): React 18 with Vite, TypeScript, Tailwind CSS
- **VS Code Extension** (`vscode-extension/`): TypeScript extension with real-time scanning, pre-built VSIX package
- **Documentation Site** (`docs-site/`): Next.js marketing and documentation site

### GitHub Marketplace Integration
- **GitHub Action** (`.github/actions/agentscan/`): Professional marketplace-ready action with comprehensive documentation
- **Marketplace Listing** (`marketplace/listing.md`): Complete marketplace copy with features, pricing, and testimonials
- **End-to-End Testing** (`tests/e2e/github-action-test.yml`): Comprehensive test suite with 6 different scenarios

### Configuration
Environment variables are managed through `.env` files and `pkg/config`. Key variables:
- Database: `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`
- Redis: `REDIS_HOST`, `REDIS_PORT`
- Auth: `JWT_SECRET`, `GITHUB_CLIENT_ID`, `GITHUB_SECRET`
- Agents: `AGENTS_MAX_CONCURRENT`, `AGENTS_DEFAULT_TIMEOUT`
- Viral Growth: `GITHUB_TOKEN`, `AGENTSCAN_API_KEY`, `MAX_PRS_PER_DAY`
- Billing: `STRIPE_KEY`, `STRIPE_WEBHOOK_SECRET` (for payment processing)

## Development Patterns

### Error Handling
Use the custom error package `pkg/errors` for structured error handling:
```go
if err := doSomething(); err != nil {
    return errors.Wrap(err, "failed to do something")
}
```

### Database Operations
Follow the repository pattern in `internal/database/repositories.go`:
```go
func (r *ScanRepository) CreateScan(ctx context.Context, scan *types.Scan) error {
    // Implementation with proper transaction handling
}
```

### Testing Patterns
- Unit tests: `*_test.go` files alongside source code
- Integration tests: `*_integration_test.go` files, use build tag `integration`
- End-to-end tests: Playwright tests in `tests/e2e/specs/`

### Security Considerations
- Never log secrets or sensitive data
- Use proper input validation and sanitization
- Follow secure coding practices for authentication and authorization
- Implement proper rate limiting and CSRF protection

## Common Development Tasks

### Adding a New Security Agent
1. Create agent directory: `agents/[category]/[tool]/`
2. Implement `SecurityAgent` interface in `agent.go`
3. Add scanner implementation in `scanner.go`
4. Write unit and integration tests
5. Add Dockerfile if tool requires containerization
6. Register agent in orchestrator configuration

### Adding API Endpoints
1. Add handlers in `internal/api/[domain].go`
2. Add routes in `internal/api/router.go`
3. Add middleware as needed (auth, rate limiting, etc.)
4. Write unit tests for handlers
5. Update OpenAPI documentation in `docs/api/openapi.yaml`

### Database Schema Changes
1. Create migration files in `migrations/`
2. Update database models in `pkg/types/`
3. Update repository methods in `internal/database/repositories.go`
4. Run migration: `make migrate-up`

### Frontend Development
1. Components go in `web/frontend/src/components/`
2. Pages go in `web/frontend/src/pages/`
3. API client in `web/frontend/src/services/api.ts`
4. Follow existing patterns for styling with Tailwind CSS
5. Write component tests with React Testing Library

### Viral Growth & Badge Generation
1. **Badge System**: Use `internal/badges` to generate "Secured by AgentScan" badges
2. **PR Automation**: `cmd/viral-growth` handles automated repository discovery and PR creation
3. **Repository Targeting**: Focus on popular repositories (1000+ stars) without existing security
4. **Rate Limiting**: Respect GitHub rate limits with `MAX_PRS_PER_DAY` configuration
5. **Templates**: Professional PR templates in `cmd/viral-growth/main.go` for different languages

### Freemium Billing System
1. **Usage Tracking**: `internal/billing/freemium.go` handles plan limits and usage monitoring
2. **Plan Tiers**: Free (100 scans), Pro ($9/month), Team ($29/month), Enterprise (custom)
3. **Limit Enforcement**: Automatic enforcement of scan quotas and private repository limits
4. **Upgrade Prompts**: Contextual upgrade suggestions based on usage patterns
5. **Watermarking**: Free tier results include AgentScan branding for viral growth

## Production Considerations

### Monitoring & Observability
- Prometheus metrics exposed on `/metrics` endpoint
- Structured logging with logrus/zap
- Distributed tracing with OpenTelemetry
- Health checks on `/health` endpoint

### Performance
- Redis caching for scan results and user sessions
- Database query optimization with proper indexing
- Agent execution parallelization and resource limiting
- Frontend code splitting and lazy loading

### Security
- JWT token authentication with refresh tokens
- Rate limiting on API endpoints
- Input validation and sanitization
- Secure headers middleware
- Audit logging for security-sensitive operations

### Deployment
- Containerized deployment with Docker
- DigitalOcean App Platform for production
- Kubernetes manifests in `k8s/` directory
- Infrastructure as Code with Terraform in `terraform/`
- Blue-green deployment scripts in `scripts/`

## VS Code Extension Development

The extension provides real-time security scanning with sub-2-second feedback:
- **Real-time scanning**: Triggered on file save with debouncing
- **Keyboard navigation**: F8/Shift+F8 to navigate findings
- **Rich hover tooltips**: Detailed vulnerability information
- **Code actions**: Quick fixes, suppression, rule management
- **WebSocket connection**: Real-time updates from server

Extension development:
```bash
cd vscode-extension
npm run compile     # Compile TypeScript
npm run watch      # Watch mode for development  
npm run package    # Package for distribution
```

## Integration Testing

The project has comprehensive integration test coverage:
- **System Integration**: Full workflow testing with real database
- **GitHub Integration**: PR comments, webhook handling, Actions integration, marketplace testing
- **Agent Integration**: End-to-end security scanning workflows
- **Performance Testing**: Load testing with K6, stress testing
- **Security Testing**: Penetration testing, auth security validation
- **Chaos Engineering**: Service resilience and failure recovery
- **GitHub Action Testing**: Complete marketplace integration testing with 6 scenarios

Run integration tests with proper environment setup:
```bash
# Full test suite (requires PostgreSQL, Redis running)
./tests/run_all_tests.sh

# Individual test categories
./tests/run_all_tests.sh --integration-only
./tests/run_all_tests.sh --performance-only
./tests/run_all_tests.sh --security-only

# GitHub Action marketplace testing
cd tests/e2e && npm run test:github-action
```

## GitHub Marketplace & Viral Growth

### GitHub Action Development
The project includes a professional GitHub Action at `.github/actions/agentscan/`:
- **SARIF Output**: Integration with GitHub Security tab
- **PR Comments**: Rich, formatted security findings in pull requests
- **Multiple Formats**: JSON, SARIF, and HTML report generation
- **Upgrade CTAs**: Contextual upgrade suggestions for premium features

### Viral Growth Mechanics
- **Badge Generation**: Dynamic "Secured by AgentScan" badges for README integration
- **Repository Discovery**: Automated identification of popular repositories without security scanning
- **Professional PRs**: Respectful, value-driven pull requests with security improvements
- **Network Effects**: Each secured repository becomes a marketing touchpoint

### Usage Analytics
- **Conversion Funnel**: 11-stage user journey tracking from discovery to paid subscription
- **Source Attribution**: Track user origins (GitHub, VS Code, web, API)
- **Cohort Analysis**: User retention and engagement metrics
- **A/B Testing**: Framework for optimizing growth and conversion