# AgentScan

AgentScan is an intelligent security scanner that orchestrates multiple scanning engines to provide developers with accurate, fast security analysis. The system uses multi-agent consensus to dramatically reduce false positives while maintaining comprehensive coverage across multiple programming languages and vulnerability types.

## Features

- **Multi-Agent Consensus**: Runs multiple security tools in parallel and uses consensus scoring to reduce false positives by 80%
- **Language Support**: JavaScript/TypeScript, Python, Go, Java, C#, Ruby, PHP, Rust
- **Fast Performance**: Full repository scans in under 5 minutes, incremental scans in under 30 seconds
- **Developer Integration**: VS Code extension, GitHub/GitLab integration, CI/CD plugins
- **Clean UI**: Modern dashboard inspired by Linear, Vercel, and Superhuman

## Architecture

AgentScan follows a microservices architecture with:

- **API Server**: REST API for client interactions
- **Orchestrator**: Manages scan jobs and agent execution
- **Agents**: Containerized security tools (Semgrep, ESLint, Bandit, etc.)
- **Database**: PostgreSQL for persistent data
- **Cache**: Redis for job queues and caching
- **Web UI**: React-based dashboard

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for development)
- Node.js 18+ (for frontend development)

### Development Setup

1. Clone the repository:
```bash
git clone https://github.com/agentscan/agentscan.git
cd agentscan
```

2. Set up environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. Start the development environment:
```bash
docker-compose up -d
```

4. Run database migrations:
```bash
go run cmd/migrate/main.go up
```

5. Start the API server:
```bash
go run cmd/api/main.go
```

6. Start the orchestrator:
```bash
go run cmd/orchestrator/main.go
```

The API will be available at `http://localhost:8080` and the web UI at `http://localhost:3000`.

## Configuration

AgentScan is configured via environment variables:

### Database
- `DB_HOST`: PostgreSQL host (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_NAME`: Database name (default: agentscan)
- `DB_USER`: Database user (default: agentscan)
- `DB_PASSWORD`: Database password (required)

### Redis
- `REDIS_HOST`: Redis host (default: localhost)
- `REDIS_PORT`: Redis port (default: 6379)
- `REDIS_PASSWORD`: Redis password (optional)

### Authentication
- `JWT_SECRET`: JWT signing secret (required)
- `GITHUB_CLIENT_ID`: GitHub OAuth client ID (required)
- `GITHUB_SECRET`: GitHub OAuth client secret (required)

### Agents
- `AGENTS_MAX_CONCURRENT`: Maximum concurrent agents (default: 10)
- `AGENTS_DEFAULT_TIMEOUT`: Default agent timeout (default: 10m)
- `AGENTS_MAX_MEMORY_MB`: Maximum memory per agent (default: 1024)

## API Usage

### Start a Scan

```bash
curl -X POST http://localhost:8080/api/v1/scans \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "repo_url": "https://github.com/user/repo",
    "branch": "main",
    "scan_type": "full"
  }'
```

### Check Scan Status

```bash
curl http://localhost:8080/api/v1/scans/{scan_id}/status \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Get Scan Results

```bash
curl http://localhost:8080/api/v1/scans/{scan_id}/results \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

## CLI Usage

```bash
# Start a scan
agentscan scan --repo https://github.com/user/repo

# Check status
agentscan status --job-id abc123

# Get results
agentscan results --job-id abc123 --format json
```

## Development

### Project Structure

```
├── cmd/                    # Application entry points
│   ├── api/               # API server
│   ├── orchestrator/      # Orchestrator service
│   └── cli/               # CLI tool
├── internal/              # Private application code
│   ├── api/               # API handlers and middleware
│   ├── orchestrator/      # Orchestration logic
│   ├── database/          # Database operations
│   └── auth/              # Authentication logic
├── pkg/                   # Public packages
│   ├── agent/             # Agent interface and types
│   ├── config/            # Configuration management
│   ├── types/             # Common data types
│   └── errors/            # Error handling
├── agents/                # Security scanning agents
│   ├── sast/              # Static analysis agents
│   ├── dast/              # Dynamic analysis agents
│   ├── sca/               # Dependency scanning agents
│   └── secrets/           # Secret scanning agents
├── web/                   # Web frontend
│   ├── frontend/          # React application
│   └── backend/           # Backend for frontend
└── docs/                  # Documentation
```

### Adding a New Agent

1. Create agent directory: `agents/category/toolname/`
2. Implement the `SecurityAgent` interface
3. Add Docker configuration
4. Write tests
5. Register with orchestrator

See the [Agent Development Guide](docs/agent-development.md) for details.

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- Documentation: [docs.agentscan.dev](https://docs.agentscan.dev)
- Issues: [GitHub Issues](https://github.com/agentscan/agentscan/issues)
- Discussions: [GitHub Discussions](https://github.com/agentscan/agentscan/discussions)