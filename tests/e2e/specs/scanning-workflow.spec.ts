import { test, expect } from '@playwright/test';
import { ApiClient } from '../utils/api-client';
import { TestDataManager } from '../utils/test-data-manager';

test.describe('Complete Scanning Workflow', () => {
  let apiClient: ApiClient;
  let testDataManager: TestDataManager;
  let testRepository: any;

  test.beforeEach(async ({ page }) => {
    apiClient = new ApiClient(page.url());
    testDataManager = new TestDataManager(apiClient);

    // Login as admin user
    await page.goto('/login');
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'test-password-123');
    await page.click('[data-testid="login-button"]');
    await expect(page).toHaveURL('/dashboard');

    // Create test repository
    testRepository = await testDataManager.createTestRepository({
      name: testDataManager.generateTestRepositoryName(),
      language: 'javascript',
      hasVulnerabilities: true,
      vulnerabilityTypes: ['xss', 'sql-injection']
    });
  });

  test.afterEach(async () => {
    // Cleanup test repository
    if (testRepository) {
      await testDataManager.deleteTestRepository(testRepository.name);
    }
  });

  test('should complete full scanning workflow from repository setup to results', async ({ page }) => {
    // Step 1: Navigate to repositories page
    await page.click('[data-testid="repositories-link"]');
    await expect(page).toHaveURL('/repositories');

    // Step 2: Add new repository
    await page.click('[data-testid="add-repository-button"]');
    
    // Fill repository form
    await page.fill('[data-testid="repo-name-input"]', testRepository.name);
    await page.fill('[data-testid="repo-url-input"]', testRepository.url);
    await page.selectOption('[data-testid="repo-language-select"]', testRepository.language);
    
    // Submit form
    await page.click('[data-testid="save-repository-button"]');
    
    // Should show success message
    await expect(page.locator('[data-testid="success-message"]')).toContainText('Repository added successfully');

    // Step 3: Start scan
    await page.click(`[data-testid="scan-button-${testRepository.id}"]`);
    
    // Configure scan options
    await page.check('[data-testid="sast-scan-checkbox"]');
    await page.check('[data-testid="dependency-scan-checkbox"]');
    await page.check('[data-testid="secret-scan-checkbox"]');
    
    // Start scan
    await page.click('[data-testid="start-scan-button"]');
    
    // Should show scan started message
    await expect(page.locator('[data-testid="success-message"]')).toContainText('Scan started successfully');

    // Step 4: Monitor scan progress
    await page.click('[data-testid="scans-link"]');
    await expect(page).toHaveURL('/scans');
    
    // Should see scan in progress
    const scanRow = page.locator(`[data-testid="scan-row"]:has-text("${testRepository.name}")`).first();
    await expect(scanRow.locator('[data-testid="scan-status"]')).toContainText('running');
    
    // Wait for scan to complete (with timeout)
    await expect(scanRow.locator('[data-testid="scan-status"]')).toContainText('completed', { timeout: 120000 });

    // Step 5: View scan results
    await scanRow.click();
    
    // Should navigate to results page
    await expect(page).toHaveURL(/\/scans\/[^\/]+\/results/);
    
    // Should show scan summary
    await expect(page.locator('[data-testid="scan-summary"]')).toBeVisible();
    await expect(page.locator('[data-testid="findings-count"]')).toContainText(/\d+ findings/);
    
    // Should show findings table
    await expect(page.locator('[data-testid="findings-table"]')).toBeVisible();
    
    // Should have findings (since we created a vulnerable repository)
    const findingsRows = page.locator('[data-testid="finding-row"]');
    await expect(findingsRows).toHaveCountGreaterThan(0);

    // Step 6: Filter and sort findings
    // Filter by severity
    await page.selectOption('[data-testid="severity-filter"]', 'high');
    await expect(page.locator('[data-testid="finding-row"][data-severity="high"]')).toHaveCountGreaterThan(0);
    
    // Sort by file path
    await page.click('[data-testid="sort-by-file"]');
    
    // Step 7: View finding details
    const firstFinding = page.locator('[data-testid="finding-row"]').first();
    await firstFinding.click();
    
    // Should show finding details modal
    await expect(page.locator('[data-testid="finding-details-modal"]')).toBeVisible();
    await expect(page.locator('[data-testid="finding-title"]')).toBeVisible();
    await expect(page.locator('[data-testid="finding-description"]')).toBeVisible();
    await expect(page.locator('[data-testid="finding-code-snippet"]')).toBeVisible();
    
    // Close modal
    await page.click('[data-testid="close-modal-button"]');

    // Step 8: Suppress a false positive
    await firstFinding.locator('[data-testid="suppress-button"]').click();
    
    // Fill suppression form
    await page.fill('[data-testid="suppression-reason"]', 'False positive - this is test code');
    await page.click('[data-testid="confirm-suppression-button"]');
    
    // Should show suppression success
    await expect(page.locator('[data-testid="success-message"]')).toContainText('Finding suppressed');
    
    // Finding should be marked as suppressed
    await expect(firstFinding.locator('[data-testid="finding-status"]')).toContainText('suppressed');

    // Step 9: Export results
    await page.click('[data-testid="export-button"]');
    await page.click('[data-testid="export-json-button"]');
    
    // Should trigger download
    const downloadPromise = page.waitForEvent('download');
    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/scan-results.*\.json$/);

    // Step 10: View scan history
    await page.click('[data-testid="scan-history-tab"]');
    
    // Should show previous scans for this repository
    await expect(page.locator('[data-testid="scan-history-table"]')).toBeVisible();
    await expect(page.locator('[data-testid="scan-history-row"]')).toHaveCountGreaterThanOrEqual(1);
  });

  test('should handle incremental scanning', async ({ page }) => {
    // First, run a full scan
    const firstScan = await testDataManager.createTestScan(testRepository.id, {
      scan_type: 'full'
    });
    
    await testDataManager.waitForScanCompletion(firstScan.id);

    // Navigate to repository page
    await page.goto(`/repositories/${testRepository.id}`);
    
    // Start incremental scan
    await page.click('[data-testid="incremental-scan-button"]');
    
    // Should show incremental scan options
    await expect(page.locator('[data-testid="incremental-scan-modal"]')).toBeVisible();
    await expect(page.locator('text=Only changed files will be scanned')).toBeVisible();
    
    // Confirm incremental scan
    await page.click('[data-testid="confirm-incremental-scan"]');
    
    // Should start incremental scan
    await expect(page.locator('[data-testid="success-message"]')).toContainText('Incremental scan started');
    
    // Navigate to scans page
    await page.click('[data-testid="scans-link"]');
    
    // Should see incremental scan
    const scanRow = page.locator('[data-testid="scan-row"]').first();
    await expect(scanRow.locator('[data-testid="scan-type"]')).toContainText('incremental');
  });

  test('should handle scan failures gracefully', async ({ page }) => {
    // Create a repository that will cause scan failures
    const failingRepo = await testDataManager.createTestRepository({
      name: 'failing-repo',
      language: 'unknown', // This should cause failures
      hasVulnerabilities: false,
      vulnerabilityTypes: []
    });

    try {
      // Navigate to repositories and start scan
      await page.goto('/repositories');
      await page.click(`[data-testid="scan-button-${failingRepo.id}"]`);
      await page.click('[data-testid="start-scan-button"]');
      
      // Wait for scan to fail
      await page.goto('/scans');
      const scanRow = page.locator(`[data-testid="scan-row"]:has-text("${failingRepo.name}")`).first();
      await expect(scanRow.locator('[data-testid="scan-status"]')).toContainText('failed', { timeout: 60000 });
      
      // Click on failed scan
      await scanRow.click();
      
      // Should show error details
      await expect(page.locator('[data-testid="scan-error-details"]')).toBeVisible();
      await expect(page.locator('[data-testid="error-message"]')).toContainText('Scan failed');
      
      // Should show retry button
      await expect(page.locator('[data-testid="retry-scan-button"]')).toBeVisible();
      
    } finally {
      await testDataManager.deleteTestRepository(failingRepo.name);
    }
  });

  test('should support real-time scan progress updates', async ({ page }) => {
    // Navigate to repositories
    await page.goto('/repositories');
    
    // Start scan
    await page.click(`[data-testid="scan-button-${testRepository.id}"]`);
    await page.click('[data-testid="start-scan-button"]');
    
    // Navigate to scan details page
    await page.goto('/scans');
    const scanRow = page.locator(`[data-testid="scan-row"]:has-text("${testRepository.name}")`).first();
    await scanRow.click();
    
    // Should show real-time progress
    await expect(page.locator('[data-testid="scan-progress-bar"]')).toBeVisible();
    await expect(page.locator('[data-testid="current-agent"]')).toBeVisible();
    
    // Progress should update over time
    const initialProgress = await page.locator('[data-testid="progress-percentage"]').textContent();
    
    // Wait a bit and check if progress changed
    await page.waitForTimeout(5000);
    const updatedProgress = await page.locator('[data-testid="progress-percentage"]').textContent();
    
    // Progress should have changed (or scan completed)
    expect(updatedProgress).not.toBe(initialProgress);
  });

  test('should handle concurrent scans properly', async ({ page }) => {
    // Create multiple test repositories
    const repos = await Promise.all([
      testDataManager.createTestRepository({
        name: 'concurrent-repo-1',
        language: 'javascript',
        hasVulnerabilities: true,
        vulnerabilityTypes: ['xss']
      }),
      testDataManager.createTestRepository({
        name: 'concurrent-repo-2',
        language: 'python',
        hasVulnerabilities: true,
        vulnerabilityTypes: ['sql-injection']
      })
    ]);

    try {
      // Start scans for both repositories
      await page.goto('/repositories');
      
      for (const repo of repos) {
        await page.click(`[data-testid="scan-button-${repo.id}"]`);
        await page.click('[data-testid="start-scan-button"]');
        await page.waitForTimeout(1000); // Small delay between scans
      }
      
      // Navigate to scans page
      await page.goto('/scans');
      
      // Should see both scans running
      for (const repo of repos) {
        const scanRow = page.locator(`[data-testid="scan-row"]:has-text("${repo.name}")`);
        await expect(scanRow).toBeVisible();
        await expect(scanRow.locator('[data-testid="scan-status"]')).toContainText(/running|queued/);
      }
      
      // Wait for both scans to complete
      for (const repo of repos) {
        const scanRow = page.locator(`[data-testid="scan-row"]:has-text("${repo.name}")`).first();
        await expect(scanRow.locator('[data-testid="scan-status"]')).toContainText('completed', { timeout: 180000 });
      }
      
    } finally {
      // Cleanup
      for (const repo of repos) {
        await testDataManager.deleteTestRepository(repo.name);
      }
    }
  });
});