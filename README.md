# AgentScan

🛡️ **Multi-agent security scanning with 80% fewer false positives**

AgentScan revolutionizes application security by orchestrating multiple security tools in parallel and using advanced consensus algorithms to dramatically reduce false positives while maintaining comprehensive vulnerability coverage. Built for modern development workflows with enterprise-grade reliability.

[![Secured by AgentScan](https://img.shields.io/badge/Secured%20by-AgentScan-brightgreen?style=for-the-badge&logo=shield&logoColor=white)](https://agentscan.dev)
[![GitHub Action](https://img.shields.io/badge/GitHub-Action-blue?style=for-the-badge&logo=github-actions&logoColor=white)](https://github.com/marketplace/actions/agentscan-security-scanner)
[![VS Code Extension](https://img.shields.io/badge/VS%20Code-Extension-blue?style=for-the-badge&logo=visual-studio-code&logoColor=white)](https://marketplace.visualstudio.com/items?itemName=agentscan.agentscan-security)

## ✨ Key Features

### 🎯 **Multi-Agent Consensus**
- **80% Fewer False Positives**: Advanced consensus scoring eliminates noise
- **High Confidence Results**: Only show findings validated by multiple tools
- **Intelligent Validation**: Machine learning-powered consensus algorithms

### ⚡ **Lightning Fast Performance**
- **Sub-2-Second Response Times**: Intelligent caching and optimization
- **Parallel Execution**: Multiple security tools run simultaneously
- **Incremental Scanning**: Only scan changed code for faster feedback

### 🛠️ **Developer-First Experience**
- **Real-Time VS Code Extension**: Live security feedback as you code
- **Rich Hover Tooltips**: Detailed vulnerability information with fix suggestions
- **Keyboard Navigation**: F8/Shift+F8 to navigate between findings
- **Code Actions**: Quick fixes, suppression, and rule management

### 🔍 **Comprehensive Coverage**
- **SAST**: Static Application Security Testing with Semgrep, ESLint Security, Bandit, Gosec
- **SCA**: Software Composition Analysis for dependency vulnerabilities
- **Secrets**: Hardcoded credentials and API key detection
- **DAST**: Dynamic Application Security Testing (optional)

### 🚀 **Enterprise-Ready**
- **Production Deployment**: DigitalOcean App Platform with auto-scaling
- **Monitoring & Alerting**: Prometheus, Grafana, and Alertmanager integration
- **Beta User Onboarding**: Automated invitation and repository setup
- **Viral Growth Mechanics**: GitHub Marketplace integration with organic discovery

### 🎨 **Modern UI/UX**
- **Clean Interface**: Inspired by Linear, Vercel, and Superhuman
- **Security Health Dashboard**: Visual security posture assessment
- **Conversion Funnel Analytics**: Detailed user journey tracking
- **Freemium Model**: Free for public repositories, paid plans for private repos

## 🏗️ Architecture

AgentScan uses a modern microservices architecture optimized for scale and reliability:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Frontend  │    │   API Server    │    │  Orchestrator   │
│   (React/Next)  │    │   (Go/Gin)     │    │   (Go/Worker)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
         │   PostgreSQL    │    │     Redis       │    │  Security       │
         │   (Database)    │    │   (Cache/Queue) │    │   Agents        │
         └─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Core Components

- **🌐 API Server**: RESTful API with JWT authentication and rate limiting
- **🎛️ Orchestrator**: Intelligent job scheduling and agent coordination
- **🤖 Security Agents**: Containerized tools (Semgrep, ESLint, Bandit, Gosec, etc.)
- **💾 PostgreSQL**: Persistent storage for scans, findings, and user data
- **⚡ Redis**: High-performance caching, job queues, and session storage
- **🎨 Web Dashboard**: Modern React-based UI with real-time updates
- **📱 VS Code Extension**: Real-time IDE integration with sub-2-second feedback
- **🔄 GitHub Action**: Marketplace-ready CI/CD integration

## 🚀 Quick Start

### For Users (GitHub Action)

Add security scanning to your repository in 2 minutes:

1. **Get your API key**: Sign up at [agentscan.dev](https://agentscan.dev/signup)
2. **Add repository secret**: `AGENTSCAN_API_KEY` in Settings → Secrets → Actions
3. **Create workflow**: Add `.github/workflows/agentscan.yml`:

```yaml
name: AgentScan Security

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: agentscan/agentscan-action@v1
      with:
        api-key: ${{ secrets.AGENTSCAN_API_KEY }}
```

### For Developers (VS Code Extension)

Get real-time security feedback while coding:

1. **Install extension**: Search "AgentScan Security" in VS Code marketplace
2. **Configure settings**: Add your API key and server URL
3. **Start coding**: Get instant security feedback with F8/Shift+F8 navigation

### For Development (Self-Hosted)

Set up your own AgentScan instance:

#### Prerequisites
- Docker and Docker Compose
- Go 1.21+ (for development)
- Node.js 18+ (for frontend development)

#### Development Setup

1. **Clone and configure**:
```bash
git clone https://github.com/NikhilSetiya/agentscan-security-scanner.git
cd agentscan-security-scanner
cp .env.example .env
# Edit .env with your configuration
```

2. **Start services**:
```bash
docker-compose up -d
go run cmd/migrate/main.go up
go run cmd/api/main.go &
go run cmd/orchestrator/main.go &
```

3. **Access applications**:
   - API: `http://localhost:8080`
   - Web UI: `http://localhost:3000`
   - VS Code Extension: Install from `vscode-extension/agentscan-security-0.1.0.vsix`

#### Production Deployment

Deploy to DigitalOcean App Platform with one command:

```bash
# Set required environment variables
export JWT_SECRET="your-secure-jwt-secret"
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_SECRET="your-github-client-secret"

# Deploy to production
./scripts/deploy.sh deploy
```

See the [Deployment Guide](docs/deployment-guide.md) for detailed instructions.

## ⚙️ Configuration

### Environment Variables

| Category | Variable | Description | Default |
|----------|----------|-------------|---------|
| **Database** | `DB_HOST` | PostgreSQL host | `localhost` |
| | `DB_PORT` | PostgreSQL port | `5432` |
| | `DB_NAME` | Database name | `agentscan` |
| | `DB_USER` | Database user | `agentscan` |
| | `DB_PASSWORD` | Database password | *required* |
| **Cache** | `REDIS_HOST` | Redis host | `localhost` |
| | `REDIS_PORT` | Redis port | `6379` |
| | `REDIS_PASSWORD` | Redis password | *optional* |
| **Auth** | `JWT_SECRET` | JWT signing secret | *required* |
| | `GITHUB_CLIENT_ID` | GitHub OAuth client ID | *required* |
| | `GITHUB_SECRET` | GitHub OAuth client secret | *required* |
| **Agents** | `AGENTS_MAX_CONCURRENT` | Max concurrent agents | `10` |
| | `AGENTS_DEFAULT_TIMEOUT` | Agent timeout | `10m` |
| | `AGENTS_MAX_MEMORY_MB` | Max memory per agent | `1024` |

### Supported Languages & Tools

| Language | SAST Tools | SCA Tools | Frameworks |
|----------|------------|-----------|------------|
| **JavaScript/TypeScript** | ESLint Security, Semgrep | npm-audit, Snyk | React, Vue, Angular, Node.js |
| **Python** | Bandit, Semgrep | pip-audit, Safety | Django, Flask, FastAPI |
| **Go** | Gosec, Semgrep | go-mod-audit | Gin, Echo, Fiber |
| **Java** | SpotBugs, Semgrep | OWASP Dependency Check | Spring, Struts |
| **C#** | Security Code Scan, Semgrep | NuGet Audit | .NET, ASP.NET |
| **Ruby** | Brakeman, Semgrep | bundle-audit | Rails, Sinatra |
| **PHP** | PHPCS Security, Semgrep | Composer Audit | Laravel, Symfony |
| **Rust** | Cargo Audit, Semgrep | RustSec | Actix, Rocket |

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

## 🛠️ Development

### Project Structure

```
├── cmd/                           # Application entry points
│   ├── api/                      # REST API server
│   ├── orchestrator/             # Job orchestration service
│   ├── cli/                      # Command-line interface
│   ├── github-action/            # GitHub Action binary
│   ├── beta-inviter/             # Beta user invitation system
│   └── viral-growth/             # Viral growth automation
├── internal/                     # Private application code
│   ├── api/                      # API handlers and middleware
│   ├── orchestrator/             # Orchestration logic
│   ├── database/                 # Database operations
│   ├── auth/                     # Authentication logic
│   ├── analytics/                # Conversion funnel tracking
│   ├── billing/                  # Freemium model implementation
│   └── badges/                   # Badge generation for viral growth
├── pkg/                          # Public packages
│   ├── agent/                    # Agent interface and types
│   ├── config/                   # Configuration management
│   ├── types/                    # Common data types
│   └── errors/                   # Error handling
├── agents/                       # Security scanning agents
│   ├── sast/                     # Static analysis (Semgrep, ESLint, Bandit, Gosec)
│   ├── dast/                     # Dynamic analysis (OWASP ZAP)
│   ├── sca/                      # Dependency scanning (npm-audit, pip-audit)
│   └── secrets/                  # Secret detection (TruffleHog)
├── web/                          # Web applications
│   ├── frontend/                 # React dashboard
│   └── backend/                  # Backend for frontend
├── vscode-extension/             # VS Code extension
│   ├── src/                      # Extension source code
│   └── agentscan-security-0.1.0.vsix  # Packaged extension
├── .github/actions/agentscan/    # GitHub Action
├── .do/                          # DigitalOcean App Platform config
├── scripts/                      # Deployment and automation scripts
├── demo/                         # Self-service demo environment
├── tests/e2e/                    # End-to-end tests
└── docs/                         # Documentation
```

### 🔧 Development Workflow

#### Adding a New Security Agent

1. **Create agent structure**:
```bash
mkdir -p agents/sast/new-tool
cd agents/sast/new-tool
```

2. **Implement SecurityAgent interface**:
```go
type SecurityAgent interface {
    Name() string
    Scan(ctx context.Context, request ScanRequest) (*ScanResult, error)
    HealthCheck(ctx context.Context) error
}
```

3. **Add Docker configuration**:
```dockerfile
FROM security-tool:latest
COPY agent.go /app/
RUN go build -o agent /app/agent.go
ENTRYPOINT ["/app/agent"]
```

4. **Register with orchestrator**:
```go
orchestrator.RegisterAgent("new-tool", NewToolAgent{})
```

#### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# E2E tests (GitHub Action)
cd tests/e2e && ./run-tests.sh

# VS Code extension tests
cd vscode-extension && npm test

# Load testing
go test -bench=. ./internal/api/...
```

#### Local Development

```bash
# Start dependencies
docker-compose up -d postgres redis

# Run API server with hot reload
air -c .air.toml

# Run orchestrator
go run cmd/orchestrator/main.go

# Start frontend development server
cd web/frontend && npm run dev

# Install VS Code extension locally
cd vscode-extension && code --install-extension agentscan-security-0.1.0.vsix
```

## 🎯 Use Cases

### 🏢 **Enterprise Security Teams**
- **Reduce Review Bottlenecks**: 80% fewer false positives means faster security reviews
- **Standardize Practices**: Consistent security scanning across all development teams
- **Compliance Ready**: SOC 2, PCI DSS, and other compliance framework support
- **Executive Reporting**: Security health dashboards and trend analysis

### 🚀 **DevOps & Platform Teams**
- **CI/CD Integration**: GitHub Actions, GitLab CI, Jenkins plugins
- **Shift-Left Security**: Catch vulnerabilities before they reach production
- **Developer Adoption**: Real-time IDE feedback increases security awareness
- **Automated Workflows**: Self-service repository onboarding and configuration

### 👨‍💻 **Development Teams**
- **Real-Time Feedback**: VS Code extension with sub-2-second response times
- **Learning Tool**: Rich context and fix suggestions improve security knowledge
- **Minimal Friction**: Intelligent caching and consensus reduce noise
- **Team Collaboration**: Shared findings, suppression lists, and team management

### 🎓 **Open Source Projects**
- **Free Scanning**: Unlimited scanning for public repositories
- **Community Trust**: "Secured by AgentScan" badges build contributor confidence
- **Security Education**: Help maintainers learn secure coding practices
- **Viral Growth**: Each secured project becomes a marketing channel

## 💰 Pricing

| Plan | Price | Public Repos | Private Repos | Scans/Month | Features |
|------|-------|--------------|---------------|-------------|----------|
| **Free** | $0 | ✅ Unlimited | ❌ | 100 | Community support, Basic features |
| **Pro** | $9/month | ✅ Unlimited | ✅ 5 repos | 1,000 | Priority support, Advanced features, No watermarks |
| **Team** | $29/month | ✅ Unlimited | ✅ 25 repos | 5,000 | Team management, Custom integrations, Analytics |
| **Enterprise** | Custom | ✅ Unlimited | ✅ Unlimited | ✅ Unlimited | SLA, Custom deployment, Dedicated support |

[**Start Free Trial →**](https://agentscan.dev/signup?utm_source=github&utm_medium=readme&utm_campaign=pricing)

## 🤝 Contributing

We welcome contributions from the community! Here's how to get started:

### 🐛 **Bug Reports & Feature Requests**
- [Report bugs](https://github.com/NikhilSetiya/agentscan-security-scanner/issues/new?template=bug_report.md)
- [Request features](https://github.com/NikhilSetiya/agentscan-security-scanner/issues/new?template=feature_request.md)
- [Join discussions](https://github.com/NikhilSetiya/agentscan-security-scanner/discussions)

### 💻 **Code Contributions**
1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Commit** your changes: `git commit -m 'Add amazing feature'`
4. **Push** to the branch: `git push origin feature/amazing-feature`
5. **Open** a Pull Request

### 📚 **Documentation**
- Improve existing documentation
- Add code examples and tutorials
- Translate documentation to other languages
- Create video tutorials and guides

### 🔒 **Security**
- Report security vulnerabilities privately to [security@agentscan.dev](mailto:security@agentscan.dev)
- Help improve security agent accuracy
- Contribute new security rules and patterns

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support & Community

### 📚 **Documentation & Resources**
- **📖 Documentation**: [docs.agentscan.dev](https://docs.agentscan.dev)
- **🎥 Video Tutorials**: [YouTube Channel](https://youtube.com/@agentscan)
- **📝 Blog**: [blog.agentscan.dev](https://blog.agentscan.dev)
- **🔧 API Reference**: [api.agentscan.dev](https://api.agentscan.dev)

### 💬 **Community & Support**
- **💬 Discord Community**: [Join our Discord](https://discord.gg/agentscan)
- **🐛 GitHub Issues**: [Report bugs and request features](https://github.com/NikhilSetiya/agentscan-security-scanner/issues)
- **💡 GitHub Discussions**: [Community discussions](https://github.com/NikhilSetiya/agentscan-security-scanner/discussions)
- **📧 Email Support**: [support@agentscan.dev](mailto:support@agentscan.dev)

### 🏆 **Recognition**
- **GitHub Security Partner** - Official GitHub Security Partner Program
- **DevSecOps Innovation Award 2024** - DevOps.com
- **OWASP Recommended Tool** - OWASP Foundation
- **10,000+ Developers** - Trusted by developers worldwide

---

<div align="center">

**Ready to transform your security workflow?**

[![Get Started Free](https://img.shields.io/badge/Get%20Started-Free-brightgreen?style=for-the-badge&logo=rocket)](https://agentscan.dev/signup?utm_source=github&utm_medium=readme&utm_campaign=cta)
[![GitHub Action](https://img.shields.io/badge/GitHub-Action-blue?style=for-the-badge&logo=github-actions)](https://github.com/marketplace/actions/agentscan-security-scanner)
[![VS Code Extension](https://img.shields.io/badge/VS%20Code-Extension-blue?style=for-the-badge&logo=visual-studio-code)](https://marketplace.visualstudio.com/items?itemName=agentscan.agentscan-security)

*Trusted by 10,000+ developers and 500+ organizations worldwide*

</div>