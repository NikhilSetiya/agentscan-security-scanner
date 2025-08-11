import { chromium, FullConfig } from '@playwright/test';
import { ApiClient } from './utils/api-client';
import { TestDataManager } from './utils/test-data-manager';

async function globalSetup(config: FullConfig) {
  console.log('🚀 Starting global setup for E2E tests...');
  
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:3000';
  const apiClient = new ApiClient(baseURL);
  const testDataManager = new TestDataManager(apiClient);
  
  try {
    // Wait for services to be ready
    console.log('⏳ Waiting for services to be ready...');
    await waitForServices(baseURL);
    
    // Setup test data
    console.log('📊 Setting up test data...');
    await testDataManager.setupTestData();
    
    // Create test users and organizations
    console.log('👥 Creating test users and organizations...');
    await createTestUsers(apiClient);
    
    // Setup test repositories
    console.log('📁 Setting up test repositories...');
    await setupTestRepositories(testDataManager);
    
    console.log('✅ Global setup completed successfully');
    
  } catch (error) {
    console.error('❌ Global setup failed:', error);
    throw error;
  }
}

async function waitForServices(baseURL: string, maxRetries = 30) {
  const apiClient = new ApiClient(baseURL);
  
  for (let i = 0; i < maxRetries; i++) {
    try {
      await apiClient.healthCheck();
      console.log('✅ Services are ready');
      return;
    } catch (error) {
      console.log(`⏳ Waiting for services... (${i + 1}/${maxRetries})`);
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
  }
  
  throw new Error('Services failed to start within timeout');
}

async function createTestUsers(apiClient: ApiClient) {
  const testUsers = [
    {
      email: 'admin@agentscan.test',
      username: 'admin',
      role: 'admin',
      password: 'test-password-123'
    },
    {
      email: 'developer@agentscan.test',
      username: 'developer',
      role: 'developer',
      password: 'test-password-123'
    },
    {
      email: 'viewer@agentscan.test',
      username: 'viewer',
      role: 'viewer',
      password: 'test-password-123'
    }
  ];
  
  for (const user of testUsers) {
    try {
      await apiClient.createUser(user);
      console.log(`✅ Created test user: ${user.username}`);
    } catch (error) {
      console.log(`⚠️  User ${user.username} might already exist`);
    }
  }
}

async function setupTestRepositories(testDataManager: TestDataManager) {
  const testRepos = [
    {
      name: 'vulnerable-js-app',
      language: 'javascript',
      hasVulnerabilities: true,
      vulnerabilityTypes: ['xss', 'sql-injection', 'insecure-dependencies']
    },
    {
      name: 'secure-python-app',
      language: 'python',
      hasVulnerabilities: false,
      vulnerabilityTypes: []
    },
    {
      name: 'mixed-language-app',
      language: 'mixed',
      hasVulnerabilities: true,
      vulnerabilityTypes: ['secrets', 'insecure-crypto', 'path-traversal']
    }
  ];
  
  for (const repo of testRepos) {
    await testDataManager.createTestRepository(repo);
    console.log(`✅ Created test repository: ${repo.name}`);
  }
}

export default globalSetup;