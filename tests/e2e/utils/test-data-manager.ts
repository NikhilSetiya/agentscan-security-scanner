import { ApiClient } from './api-client';
import { v4 as uuidv4 } from 'uuid';
import * as fs from 'fs';
import * as path from 'path';

export interface TestRepository {
  name: string;
  language: string;
  hasVulnerabilities: boolean;
  vulnerabilityTypes: string[];
}

export class TestDataManager {
  private apiClient: ApiClient;
  private createdResources: Map<string, string[]> = new Map();

  constructor(apiClient: ApiClient) {
    this.apiClient = apiClient;
  }

  async setupTestData(): Promise<void> {
    // Initialize resource tracking
    this.createdResources.set('users', []);
    this.createdResources.set('organizations', []);
    this.createdResources.set('repositories', []);
    this.createdResources.set('scans', []);
  }

  async createTestRepository(repoConfig: TestRepository): Promise<any> {
    // Create the repository record
    const repository = await this.apiClient.createRepository({
      name: repoConfig.name,
      url: `https://github.com/test/${repoConfig.name}`,
      language: repoConfig.language,
      branch: 'main',
    });

    // Track created resource
    this.createdResources.get('repositories')?.push(repository.id);

    // Create test files for the repository
    await this.createTestFiles(repository.id, repoConfig);

    return repository;
  }

  async createTestFiles(repositoryId: string, repoConfig: TestRepository): Promise<void> {
    const testFilesDir = path.join(__dirname, '../test-files', repoConfig.language);
    
    if (!fs.existsSync(testFilesDir)) {
      fs.mkdirSync(testFilesDir, { recursive: true });
    }

    // Create language-specific test files with vulnerabilities
    switch (repoConfig.language) {
      case 'javascript':
        await this.createJavaScriptTestFiles(testFilesDir, repoConfig.hasVulnerabilities);
        break;
      case 'python':
        await this.createPythonTestFiles(testFilesDir, repoConfig.hasVulnerabilities);
        break;
      case 'mixed':
        await this.createMixedLanguageTestFiles(testFilesDir, repoConfig.hasVulnerabilities);
        break;
    }
  }

  private async createJavaScriptTestFiles(dir: string, hasVulnerabilities: boolean): Promise<void> {
    // Create package.json
    const packageJson = {
      name: 'test-app',
      version: '1.0.0',
      dependencies: hasVulnerabilities ? {
        'lodash': '4.17.15', // Known vulnerable version
        'express': '4.16.0', // Older version with vulnerabilities
      } : {
        'lodash': '^4.17.21',
        'express': '^4.18.0',
      },
    };

    fs.writeFileSync(
      path.join(dir, 'package.json'),
      JSON.stringify(packageJson, null, 2)
    );

    // Create vulnerable JavaScript code
    const vulnerableCode = hasVulnerabilities ? `
// XSS vulnerability
app.get('/search', (req, res) => {
  const query = req.query.q;
  res.send('<h1>Results for: ' + query + '</h1>'); // XSS vulnerability
});

// SQL Injection vulnerability
app.get('/user/:id', (req, res) => {
  const userId = req.params.id;
  const query = 'SELECT * FROM users WHERE id = ' + userId; // SQL injection
  db.query(query, (err, results) => {
    res.json(results);
  });
});

// Insecure crypto
const crypto = require('crypto');
const hash = crypto.createHash('md5').update('password').digest('hex'); // Weak hash

// Hardcoded secret
const API_KEY = 'sk-1234567890abcdef'; // Hardcoded secret
` : `
// Secure code
app.get('/search', (req, res) => {
  const query = req.query.q;
  res.render('search', { query: escapeHtml(query) }); // Properly escaped
});

// Parameterized query
app.get('/user/:id', (req, res) => {
  const userId = parseInt(req.params.id);
  const query = 'SELECT * FROM users WHERE id = ?';
  db.query(query, [userId], (err, results) => {
    res.json(results);
  });
});

// Secure crypto
const crypto = require('crypto');
const hash = crypto.createHash('sha256').update('password').digest('hex');

// Environment variable
const API_KEY = process.env.API_KEY;
`;

    fs.writeFileSync(path.join(dir, 'app.js'), vulnerableCode);
  }

  private async createPythonTestFiles(dir: string, hasVulnerabilities: boolean): Promise<void> {
    // Create requirements.txt
    const requirements = hasVulnerabilities ? `
Django==2.0.0
requests==2.18.0
pyyaml==3.12
` : `
Django>=4.2.0
requests>=2.28.0
pyyaml>=6.0
`;

    fs.writeFileSync(path.join(dir, 'requirements.txt'), requirements.trim());

    // Create vulnerable Python code
    const vulnerableCode = hasVulnerabilities ? `
import os
import subprocess
import pickle
import yaml

# Command injection vulnerability
def run_command(user_input):
    command = f"ls {user_input}"  # Command injection
    subprocess.call(command, shell=True)

# Insecure deserialization
def load_data(data):
    return pickle.loads(data)  # Insecure deserialization

# YAML load vulnerability
def load_config(config_file):
    with open(config_file, 'r') as f:
        return yaml.load(f)  # Unsafe YAML load

# Hardcoded credentials
DATABASE_PASSWORD = "admin123"  # Hardcoded password

# SQL injection (if using raw SQL)
def get_user(user_id):
    query = f"SELECT * FROM users WHERE id = {user_id}"  # SQL injection
    return execute_query(query)
` : `
import os
import subprocess
import pickle
import yaml

# Secure command execution
def run_command(user_input):
    # Validate and sanitize input
    if not user_input.isalnum():
        raise ValueError("Invalid input")
    subprocess.run(["ls", user_input], check=True)

# Secure deserialization
def load_data(data):
    # Use JSON instead of pickle
    import json
    return json.loads(data)

# Safe YAML loading
def load_config(config_file):
    with open(config_file, 'r') as f:
        return yaml.safe_load(f)

# Environment variable
DATABASE_PASSWORD = os.environ.get('DATABASE_PASSWORD')

# Parameterized query
def get_user(user_id):
    query = "SELECT * FROM users WHERE id = %s"
    return execute_query(query, (user_id,))
`;

    fs.writeFileSync(path.join(dir, 'app.py'), vulnerableCode);
  }

  private async createMixedLanguageTestFiles(dir: string, hasVulnerabilities: boolean): Promise<void> {
    // Create both JavaScript and Python files
    await this.createJavaScriptTestFiles(dir, hasVulnerabilities);
    await this.createPythonTestFiles(dir, hasVulnerabilities);

    // Create additional files with secrets
    if (hasVulnerabilities) {
      const envFile = `
# Database configuration
DB_HOST=localhost
DB_USER=admin
DB_PASSWORD=super_secret_password_123
DB_NAME=production

# API Keys
GITHUB_TOKEN=ghp_1234567890abcdefghijklmnopqrstuvwxyz
SLACK_WEBHOOK=https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX
AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# JWT Secret
JWT_SECRET=my-super-secret-jwt-key-that-should-not-be-hardcoded
`;

      fs.writeFileSync(path.join(dir, '.env'), envFile);

      // Create a config file with secrets
      const configFile = `
{
  "database": {
    "host": "localhost",
    "username": "admin",
    "password": "hardcoded_password_123"
  },
  "api_keys": {
    "stripe": "sk_test_1234567890abcdef",
    "sendgrid": "SG.1234567890abcdef.1234567890abcdef"
  }
}
`;

      fs.writeFileSync(path.join(dir, 'config.json'), configFile);
    }
  }

  async createTestScan(repositoryId: string, options: any = {}): Promise<any> {
    const scan = await this.apiClient.submitScan(repositoryId, options);
    this.createdResources.get('scans')?.push(scan.id);
    return scan;
  }

  async waitForScanCompletion(scanId: string, timeoutMs: number = 300000): Promise<any> {
    const startTime = Date.now();
    
    while (Date.now() - startTime < timeoutMs) {
      const scan = await this.apiClient.getScanStatus(scanId);
      
      if (scan.status === 'completed' || scan.status === 'failed') {
        return scan;
      }
      
      // Wait 2 seconds before checking again
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
    
    throw new Error(`Scan ${scanId} did not complete within ${timeoutMs}ms`);
  }

  async createTestOrganization(name: string): Promise<any> {
    const organization = await this.apiClient.createOrganization({
      name,
      description: `Test organization: ${name}`,
    });
    
    this.createdResources.get('organizations')?.push(organization.id);
    return organization;
  }

  async deleteTestRepository(name: string): Promise<void> {
    try {
      const repositories = await this.apiClient.getRepositories();
      const repo = repositories.find(r => r.name === name);
      
      if (repo) {
        await this.apiClient.deleteRepository(repo.id);
        
        // Remove from tracking
        const repoIds = this.createdResources.get('repositories') || [];
        const index = repoIds.indexOf(repo.id);
        if (index > -1) {
          repoIds.splice(index, 1);
        }
      }
    } catch (error) {
      console.warn(`Failed to delete test repository ${name}:`, error);
    }
  }

  async cleanupTestData(): Promise<void> {
    // Clean up in reverse order of creation
    const resourceTypes = ['scans', 'repositories', 'organizations', 'users'];
    
    for (const resourceType of resourceTypes) {
      const resourceIds = this.createdResources.get(resourceType) || [];
      
      for (const resourceId of resourceIds) {
        try {
          switch (resourceType) {
            case 'users':
              await this.apiClient.deleteUser(resourceId);
              break;
            case 'repositories':
              await this.apiClient.deleteRepository(resourceId);
              break;
            // Add other cleanup methods as needed
          }
        } catch (error) {
          console.warn(`Failed to cleanup ${resourceType} ${resourceId}:`, error);
        }
      }
      
      // Clear the tracking
      this.createdResources.set(resourceType, []);
    }
  }

  // Utility methods for test data generation
  generateTestEmail(): string {
    return `test-${uuidv4()}@agentscan.test`;
  }

  generateTestUsername(): string {
    return `testuser-${uuidv4().substring(0, 8)}`;
  }

  generateTestRepositoryName(): string {
    return `test-repo-${uuidv4().substring(0, 8)}`;
  }

  generateTestOrganizationName(): string {
    return `Test Org ${uuidv4().substring(0, 8)}`;
  }
}