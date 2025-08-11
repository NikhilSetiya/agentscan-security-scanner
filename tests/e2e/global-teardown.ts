import { FullConfig } from '@playwright/test';
import { ApiClient } from './utils/api-client';
import { TestDataManager } from './utils/test-data-manager';

async function globalTeardown(config: FullConfig) {
  console.log('🧹 Starting global teardown for E2E tests...');
  
  const baseURL = config.projects[0].use.baseURL || 'http://localhost:3000';
  const apiClient = new ApiClient(baseURL);
  const testDataManager = new TestDataManager(apiClient);
  
  try {
    // Clean up test data
    console.log('🗑️  Cleaning up test data...');
    await testDataManager.cleanupTestData();
    
    // Clean up test repositories
    console.log('📁 Cleaning up test repositories...');
    await cleanupTestRepositories(testDataManager);
    
    // Clean up test users (optional - might want to keep for debugging)
    if (process.env.CLEANUP_TEST_USERS === 'true') {
      console.log('👥 Cleaning up test users...');
      await cleanupTestUsers(apiClient);
    }
    
    console.log('✅ Global teardown completed successfully');
    
  } catch (error) {
    console.error('❌ Global teardown failed:', error);
    // Don't throw error in teardown to avoid masking test failures
  }
}

async function cleanupTestRepositories(testDataManager: TestDataManager) {
  const testRepoNames = [
    'vulnerable-js-app',
    'secure-python-app',
    'mixed-language-app'
  ];
  
  for (const repoName of testRepoNames) {
    try {
      await testDataManager.deleteTestRepository(repoName);
      console.log(`✅ Cleaned up test repository: ${repoName}`);
    } catch (error) {
      console.log(`⚠️  Failed to cleanup repository ${repoName}:`, error);
    }
  }
}

async function cleanupTestUsers(apiClient: ApiClient) {
  const testUsernames = ['admin', 'developer', 'viewer'];
  
  for (const username of testUsernames) {
    try {
      await apiClient.deleteUser(username);
      console.log(`✅ Cleaned up test user: ${username}`);
    } catch (error) {
      console.log(`⚠️  Failed to cleanup user ${username}:`, error);
    }
  }
}

export default globalTeardown;