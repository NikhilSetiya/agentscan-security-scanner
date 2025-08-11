# API Integration Guide

This comprehensive guide covers integrating with the AgentScan API, including authentication, common workflows, SDKs, and best practices.

## Getting Started

### Base URL

The AgentScan API is available at:
- Production: `https://api.agentscan.dev/v1`
- Staging: `https://staging-api.agentscan.dev/v1`
- Local: `http://localhost:8080/api/v1`

### Authentication

AgentScan uses JWT-based authentication. All API requests (except authentication endpoints) require a valid JWT token in the Authorization header.

#### Getting an Access Token

```bash
curl -X POST https://api.agentscan.dev/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-password"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "user-123",
    "username": "your-username",
    "role": "developer"
  },
  "expires_at": "2024-01-01T12:00:00Z"
}
```

#### Using the Token

Include the token in all subsequent requests:

```bash
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  https://api.agentscan.dev/v1/repositories
```

## Core Workflows

### 1. Repository Management

#### Adding a Repository

```bash
curl -X POST https://api.agentscan.dev/v1/repositories \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-secure-app",
    "url": "https://github.com/company/my-secure-app",
    "language": "javascript",
    "branch": "main",
    "description": "Main application repository"
  }'
```

#### Listing Repositories

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.agentscan.dev/v1/repositories?page=1&limit=20&search=my-app"
```

#### Getting Repository Details

```bash
curl -H "Authorization: Bearer $TOKEN" \
  https://api.agentscan.dev/v1/repositories/repo-123
```

### 2. Security Scanning

#### Submitting a Scan

```bash
curl -X POST https://api.agentscan.dev/v1/scans \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "repository_id": "repo-123",
    "scan_type": "full",
    "branch": "main",
    "agents": ["semgrep", "eslint", "bandit", "npm-audit"]
  }'
```

Response:
```json
{
  "id": "scan-456",
  "repository_id": "repo-123",
  "status": "queued",
  "scan_type": "full",
  "agents": ["semgrep", "eslint", "bandit", "npm-audit"],
  "progress": 0,
  "started_at": "2024-01-01T12:00:00Z"
}
```

#### Monitoring Scan Progress

```bash
curl -H "Authorization: Bearer $TOKEN" \
  https://api.agentscan.dev/v1/scans/scan-456
```

#### Getting Scan Results

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.agentscan.dev/v1/scans/scan-456/results?severity=high,critical"
```

### 3. Finding Management

#### Listing Findings

```bash
curl -H "Authorization: Bearer $TOKEN" \
  "https://api.agentscan.dev/v1/findings?repository_id=repo-123&status=open&severity=high"
```

#### Updating Finding Status

```bash
curl -X PATCH https://api.agentscan.dev/v1/findings/finding-789 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "fixed",
    "comment": "Fixed by implementing proper input validation"
  }'
```

#### Suppressing False Positives

```bash
curl -X POST https://api.agentscan.dev/v1/findings/finding-789/suppress \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "False positive - this is test code",
    "expires_at": "2024-06-01T00:00:00Z"
  }'
```

## Real-Time Updates with WebSockets

AgentScan provides WebSocket connections for real-time scan progress updates.

### JavaScript Example

```javascript
const token = 'your-jwt-token';
const scanId = 'scan-456';
const ws = new WebSocket(`wss://api.agentscan.dev/v1/ws/scans/${scanId}?token=${token}`);

ws.onopen = function() {
    console.log('Connected to scan updates');
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    
    switch (message.type) {
        case 'scan_progress':
            console.log(`Scan progress: ${message.data.progress}%`);
            console.log(`Current agent: ${message.data.current_agent}`);
            break;
            
        case 'scan_completed':
            console.log('Scan completed!');
            console.log(`Total findings: ${message.data.findings_count}`);
            break;
            
        case 'scan_failed':
            console.error('Scan failed:', message.data.error);
            break;
            
        case 'finding_detected':
            console.log('New finding detected:', message.data.finding);
            break;
    }
};

ws.onerror = function(error) {
    console.error('WebSocket error:', error);
};

ws.onclose = function() {
    console.log('WebSocket connection closed');
};
```

### Python Example

```python
import asyncio
import websockets
import json

async def monitor_scan(token, scan_id):
    uri = f"wss://api.agentscan.dev/v1/ws/scans/{scan_id}?token={token}"
    
    async with websockets.connect(uri) as websocket:
        print("Connected to scan updates")
        
        async for message in websocket:
            data = json.loads(message)
            
            if data['type'] == 'scan_progress':
                progress = data['data']['progress']
                agent = data['data']['current_agent']
                print(f"Scan progress: {progress}% (agent: {agent})")
                
            elif data['type'] == 'scan_completed':
                findings = data['data']['findings_count']
                print(f"Scan completed! Total findings: {findings}")
                break
                
            elif data['type'] == 'scan_failed':
                error = data['data']['error']
                print(f"Scan failed: {error}")
                break

# Usage
asyncio.run(monitor_scan('your-jwt-token', 'scan-456'))
```

## SDKs and Libraries

### Official Python SDK

```python
from agentscan import AgentScanClient

# Initialize client
client = AgentScanClient(
    base_url='https://api.agentscan.dev/v1',
    username='your-username',
    password='your-password'
)

# Add repository
repository = client.repositories.create({
    'name': 'my-secure-app',
    'url': 'https://github.com/company/my-secure-app',
    'language': 'javascript'
})

# Submit scan
scan = client.scans.create({
    'repository_id': repository['id'],
    'agents': ['semgrep', 'eslint']
})

# Wait for completion
scan = client.scans.wait_for_completion(scan['id'], timeout=300)

# Get results
results = client.scans.get_results(scan['id'])
print(f"Found {len(results['findings'])} security issues")

# Filter critical findings
critical_findings = [
    f for f in results['findings'] 
    if f['severity'] == 'critical'
]
```

### JavaScript/TypeScript SDK

```typescript
import { AgentScanClient } from '@agentscan/sdk';

const client = new AgentScanClient({
  baseUrl: 'https://api.agentscan.dev/v1',
  username: 'your-username',
  password: 'your-password'
});

// Add repository
const repository = await client.repositories.create({
  name: 'my-secure-app',
  url: 'https://github.com/company/my-secure-app',
  language: 'javascript'
});

// Submit scan with progress monitoring
const scan = await client.scans.create({
  repository_id: repository.id,
  agents: ['semgrep', 'eslint']
});

// Monitor progress
client.scans.onProgress(scan.id, (progress) => {
  console.log(`Scan progress: ${progress.progress}%`);
});

// Wait for completion
const completedScan = await client.scans.waitForCompletion(scan.id);

// Get results
const results = await client.scans.getResults(scan.id);
console.log(`Found ${results.findings.length} security issues`);
```

### Go SDK

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/agentscan/go-sdk"
)

func main() {
    client := agentscan.NewClient(&agentscan.Config{
        BaseURL:  "https://api.agentscan.dev/v1",
        Username: "your-username",
        Password: "your-password",
    })
    
    ctx := context.Background()
    
    // Add repository
    repo, err := client.Repositories.Create(ctx, &agentscan.CreateRepositoryRequest{
        Name:     "my-secure-app",
        URL:      "https://github.com/company/my-secure-app",
        Language: "javascript",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Submit scan
    scan, err := client.Scans.Create(ctx, &agentscan.CreateScanRequest{
        RepositoryID: repo.ID,
        Agents:       []string{"semgrep", "eslint"},
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Wait for completion
    scan, err = client.Scans.WaitForCompletion(ctx, scan.ID, 300)
    if err != nil {
        log.Fatal(err)
    }
    
    // Get results
    results, err := client.Scans.GetResults(ctx, scan.ID)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d security issues\n", len(results.Findings))
}
```

## Advanced Integration Patterns

### Batch Operations

```python
# Batch repository creation
repositories = [
    {'name': 'app-1', 'url': 'https://github.com/org/app-1', 'language': 'javascript'},
    {'name': 'app-2', 'url': 'https://github.com/org/app-2', 'language': 'python'},
    {'name': 'app-3', 'url': 'https://github.com/org/app-3', 'language': 'go'},
]

created_repos = []
for repo_data in repositories:
    repo = client.repositories.create(repo_data)
    created_repos.append(repo)
    print(f"Created repository: {repo['name']}")

# Batch scan submission
scan_requests = [
    {'repository_id': repo['id'], 'agents': ['semgrep']}
    for repo in created_repos
]

scans = []
for scan_request in scan_requests:
    scan = client.scans.create(scan_request)
    scans.append(scan)
    print(f"Started scan: {scan['id']}")

# Monitor all scans
completed_scans = client.scans.wait_for_multiple(
    [scan['id'] for scan in scans],
    timeout=600
)
```

### Webhook Integration

```python
from flask import Flask, request, jsonify
import hmac
import hashlib

app = Flask(__name__)
WEBHOOK_SECRET = 'your-webhook-secret'

@app.route('/webhooks/agentscan', methods=['POST'])
def handle_webhook():
    # Verify webhook signature
    signature = request.headers.get('X-AgentScan-Signature')
    if not verify_signature(request.data, signature):
        return jsonify({'error': 'Invalid signature'}), 401
    
    event = request.json
    
    if event['type'] == 'scan.completed':
        handle_scan_completed(event['data'])
    elif event['type'] == 'finding.critical':
        handle_critical_finding(event['data'])
    elif event['type'] == 'scan.failed':
        handle_scan_failed(event['data'])
    
    return jsonify({'status': 'ok'})

def verify_signature(payload, signature):
    expected = hmac.new(
        WEBHOOK_SECRET.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f'sha256={expected}', signature)

def handle_scan_completed(data):
    scan_id = data['scan_id']
    findings_count = data['findings_count']
    
    # Send notification to team
    send_slack_notification(
        f"Scan {scan_id} completed with {findings_count} findings"
    )

def handle_critical_finding(data):
    finding = data['finding']
    
    # Create urgent ticket
    create_jira_ticket(
        title=f"Critical Security Issue: {finding['title']}",
        description=finding['description'],
        priority='Critical'
    )
```

### Custom Reporting

```python
class SecurityReportGenerator:
    def __init__(self, client):
        self.client = client
    
    def generate_executive_summary(self, repository_ids, days=30):
        """Generate executive summary for multiple repositories"""
        
        # Get findings for all repositories
        all_findings = []
        for repo_id in repository_ids:
            findings = self.client.findings.list(
                repository_id=repo_id,
                created_after=datetime.now() - timedelta(days=days)
            )
            all_findings.extend(findings)
        
        # Aggregate statistics
        stats = {
            'total_findings': len(all_findings),
            'by_severity': self._group_by_severity(all_findings),
            'by_category': self._group_by_category(all_findings),
            'trend': self._calculate_trend(all_findings),
            'top_vulnerabilities': self._get_top_vulnerabilities(all_findings)
        }
        
        return self._format_executive_report(stats)
    
    def generate_developer_report(self, repository_id):
        """Generate detailed report for developers"""
        
        findings = self.client.findings.list(
            repository_id=repository_id,
            status='open'
        )
        
        # Group by file and severity
        report = {
            'summary': self._generate_summary(findings),
            'by_file': self._group_by_file(findings),
            'remediation_guide': self._generate_remediation_guide(findings),
            'false_positive_candidates': self._identify_false_positives(findings)
        }
        
        return report
```

### CI/CD Integration

#### GitHub Actions

```yaml
name: Security Scan
on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  security-scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: AgentScan Security Scan
      uses: agentscan/github-action@v1
      with:
        api-token: ${{ secrets.AGENTSCAN_TOKEN }}
        repository-url: ${{ github.repository }}
        branch: ${{ github.ref_name }}
        agents: 'semgrep,eslint,bandit'
        fail-on-critical: true
        
    - name: Upload Results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: security-scan-results
        path: agentscan-results.json
```

#### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    environment {
        AGENTSCAN_TOKEN = credentials('agentscan-token')
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        
        stage('Security Scan') {
            steps {
                script {
                    def scanResult = sh(
                        script: """
                            curl -X POST https://api.agentscan.dev/v1/scans \\
                              -H "Authorization: Bearer ${AGENTSCAN_TOKEN}" \\
                              -H "Content-Type: application/json" \\
                              -d '{
                                "repository_id": "${env.REPOSITORY_ID}",
                                "branch": "${env.BRANCH_NAME}",
                                "commit": "${env.GIT_COMMIT}",
                                "agents": ["semgrep", "eslint"]
                              }'
                        """,
                        returnStdout: true
                    ).trim()
                    
                    def scan = readJSON text: scanResult
                    
                    // Wait for completion
                    timeout(time: 10, unit: 'MINUTES') {
                        waitUntil {
                            def status = sh(
                                script: """
                                    curl -H "Authorization: Bearer ${AGENTSCAN_TOKEN}" \\
                                      https://api.agentscan.dev/v1/scans/${scan.id}
                                """,
                                returnStdout: true
                            ).trim()
                            
                            def scanStatus = readJSON text: status
                            return scanStatus.status == 'completed' || scanStatus.status == 'failed'
                        }
                    }
                    
                    // Get results
                    def results = sh(
                        script: """
                            curl -H "Authorization: Bearer ${AGENTSCAN_TOKEN}" \\
                              https://api.agentscan.dev/v1/scans/${scan.id}/results
                        """,
                        returnStdout: true
                    ).trim()
                    
                    writeFile file: 'security-results.json', text: results
                    archiveArtifacts artifacts: 'security-results.json'
                    
                    def resultsData = readJSON text: results
                    def criticalFindings = resultsData.findings.findAll { it.severity == 'critical' }
                    
                    if (criticalFindings.size() > 0) {
                        error("Found ${criticalFindings.size()} critical security issues")
                    }
                }
            }
        }
    }
    
    post {
        always {
            publishHTML([
                allowMissing: false,
                alwaysLinkToLastBuild: true,
                keepAll: true,
                reportDir: '.',
                reportFiles: 'security-results.json',
                reportName: 'Security Scan Results'
            ])
        }
    }
}
```

## Error Handling

### Common HTTP Status Codes

- `200 OK` - Request successful
- `201 Created` - Resource created successfully
- `400 Bad Request` - Invalid request parameters
- `401 Unauthorized` - Authentication required or invalid
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource already exists
- `422 Unprocessable Entity` - Validation errors
- `429 Too Many Requests` - Rate limit exceeded
- `500 Internal Server Error` - Server error

### Error Response Format

```json
{
  "error": "Invalid request parameters",
  "code": "INVALID_REQUEST",
  "details": [
    {
      "field": "repository_url",
      "message": "Invalid URL format"
    }
  ]
}
```

### Retry Logic

```python
import time
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

class AgentScanClient:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.session = requests.Session()
        
        # Configure retry strategy
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["HEAD", "GET", "OPTIONS", "POST", "PUT", "PATCH"]
        )
        
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)
        
        # Set default headers
        self.session.headers.update({
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        })
    
    def make_request(self, method, endpoint, **kwargs):
        url = f"{self.base_url}{endpoint}"
        
        try:
            response = self.session.request(method, url, **kwargs)
            response.raise_for_status()
            return response.json()
            
        except requests.exceptions.HTTPError as e:
            if e.response.status_code == 401:
                raise AuthenticationError("Invalid or expired token")
            elif e.response.status_code == 403:
                raise PermissionError("Insufficient permissions")
            elif e.response.status_code == 429:
                raise RateLimitError("Rate limit exceeded")
            else:
                raise APIError(f"HTTP {e.response.status_code}: {e.response.text}")
                
        except requests.exceptions.RequestException as e:
            raise ConnectionError(f"Failed to connect to API: {e}")
```

## Rate Limiting

AgentScan implements rate limiting to ensure fair usage:

- **Authentication**: 10 requests per minute
- **API Operations**: 1000 requests per hour per user
- **Scan Submissions**: 100 scans per hour per organization

### Handling Rate Limits

```python
def handle_rate_limit(response):
    if response.status_code == 429:
        retry_after = int(response.headers.get('Retry-After', 60))
        print(f"Rate limited. Waiting {retry_after} seconds...")
        time.sleep(retry_after)
        return True
    return False

def make_request_with_backoff(client, method, endpoint, **kwargs):
    max_retries = 3
    
    for attempt in range(max_retries):
        response = client.session.request(method, endpoint, **kwargs)
        
        if not handle_rate_limit(response):
            return response
        
        if attempt == max_retries - 1:
            raise Exception("Max retries exceeded due to rate limiting")
    
    return response
```

## Best Practices

### 1. Authentication Management

```python
class TokenManager:
    def __init__(self, username, password):
        self.username = username
        self.password = password
        self.token = None
        self.expires_at = None
    
    def get_valid_token(self):
        if self.token and self.expires_at > datetime.now():
            return self.token
        
        # Refresh token
        response = requests.post('/auth/login', json={
            'username': self.username,
            'password': self.password
        })
        
        data = response.json()
        self.token = data['token']
        self.expires_at = datetime.fromisoformat(data['expires_at'])
        
        return self.token
```

### 2. Efficient Polling

```python
def wait_for_scan_completion(client, scan_id, timeout=300):
    start_time = time.time()
    backoff = 1
    
    while time.time() - start_time < timeout:
        scan = client.scans.get(scan_id)
        
        if scan['status'] in ['completed', 'failed']:
            return scan
        
        # Exponential backoff with jitter
        sleep_time = min(backoff + random.uniform(0, 1), 30)
        time.sleep(sleep_time)
        backoff = min(backoff * 1.5, 30)
    
    raise TimeoutError(f"Scan {scan_id} did not complete within {timeout} seconds")
```

### 3. Resource Cleanup

```python
class ScanManager:
    def __init__(self, client):
        self.client = client
        self.active_scans = set()
    
    def submit_scan(self, **kwargs):
        scan = self.client.scans.create(kwargs)
        self.active_scans.add(scan['id'])
        return scan
    
    def cleanup_completed_scans(self):
        for scan_id in list(self.active_scans):
            scan = self.client.scans.get(scan_id)
            if scan['status'] in ['completed', 'failed']:
                self.active_scans.remove(scan_id)
    
    def cancel_all_scans(self):
        for scan_id in self.active_scans:
            try:
                self.client.scans.cancel(scan_id)
            except Exception as e:
                print(f"Failed to cancel scan {scan_id}: {e}")
        
        self.active_scans.clear()
```

### 4. Structured Logging

```python
import structlog

logger = structlog.get_logger()

def submit_scan_with_logging(client, repository_id, agents):
    log = logger.bind(
        repository_id=repository_id,
        agents=agents,
        operation="submit_scan"
    )
    
    log.info("Submitting security scan")
    
    try:
        scan = client.scans.create({
            'repository_id': repository_id,
            'agents': agents
        })
        
        log.info("Scan submitted successfully", scan_id=scan['id'])
        return scan
        
    except Exception as e:
        log.error("Failed to submit scan", error=str(e))
        raise
```

## Testing

### Unit Tests

```python
import unittest
from unittest.mock import Mock, patch
from agentscan import AgentScanClient

class TestAgentScanClient(unittest.TestCase):
    def setUp(self):
        self.client = AgentScanClient(
            base_url='https://api.example.com',
            token='test-token'
        )
    
    @patch('requests.Session.request')
    def test_create_repository(self, mock_request):
        mock_response = Mock()
        mock_response.json.return_value = {
            'id': 'repo-123',
            'name': 'test-repo'
        }
        mock_response.status_code = 201
        mock_request.return_value = mock_response
        
        result = self.client.repositories.create({
            'name': 'test-repo',
            'url': 'https://github.com/test/repo'
        })
        
        self.assertEqual(result['id'], 'repo-123')
        mock_request.assert_called_once()
    
    @patch('requests.Session.request')
    def test_submit_scan(self, mock_request):
        mock_response = Mock()
        mock_response.json.return_value = {
            'id': 'scan-456',
            'status': 'queued'
        }
        mock_response.status_code = 201
        mock_request.return_value = mock_response
        
        result = self.client.scans.create({
            'repository_id': 'repo-123',
            'agents': ['semgrep']
        })
        
        self.assertEqual(result['id'], 'scan-456')
        self.assertEqual(result['status'], 'queued')
```

### Integration Tests

```python
class TestAgentScanIntegration(unittest.TestCase):
    def setUp(self):
        self.client = AgentScanClient(
            base_url=os.getenv('AGENTSCAN_TEST_URL'),
            username=os.getenv('AGENTSCAN_TEST_USERNAME'),
            password=os.getenv('AGENTSCAN_TEST_PASSWORD')
        )
    
    def test_full_scan_workflow(self):
        # Create test repository
        repo = self.client.repositories.create({
            'name': f'test-repo-{int(time.time())}',
            'url': 'https://github.com/test/vulnerable-app',
            'language': 'javascript'
        })
        
        try:
            # Submit scan
            scan = self.client.scans.create({
                'repository_id': repo['id'],
                'agents': ['semgrep']
            })
            
            # Wait for completion
            completed_scan = self.client.scans.wait_for_completion(
                scan['id'], timeout=300
            )
            
            self.assertEqual(completed_scan['status'], 'completed')
            
            # Get results
            results = self.client.scans.get_results(scan['id'])
            self.assertIsInstance(results['findings'], list)
            
        finally:
            # Cleanup
            self.client.repositories.delete(repo['id'])
```

## Troubleshooting

### Common Issues

1. **Authentication Errors**
   - Verify token is not expired
   - Check token format and encoding
   - Ensure proper Authorization header format

2. **Rate Limiting**
   - Implement exponential backoff
   - Monitor rate limit headers
   - Consider request batching

3. **Timeout Issues**
   - Increase timeout values for large repositories
   - Use WebSocket connections for real-time updates
   - Implement proper retry logic

4. **Network Connectivity**
   - Check firewall settings
   - Verify DNS resolution
   - Test with curl or similar tools

### Debug Mode

```python
import logging

# Enable debug logging
logging.basicConfig(level=logging.DEBUG)

# Enable HTTP request logging
import http.client as http_client
http_client.HTTPConnection.debuglevel = 1

# Your API calls will now show detailed HTTP traffic
client = AgentScanClient(base_url='...', token='...')
result = client.repositories.list()
```

For more detailed troubleshooting, see the [Troubleshooting Guide](../troubleshooting.md).