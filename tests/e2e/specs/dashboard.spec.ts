import { test, expect } from '@playwright/test';
import { ApiClient } from '../utils/api-client';
import { TestDataManager } from '../utils/test-data-manager';

test.describe('Dashboard Functionality', () => {
  let apiClient: ApiClient;
  let testDataManager: TestDataManager;

  test.beforeEach(async ({ page }) => {
    apiClient = new ApiClient(page.url());
    testDataManager = new TestDataManager(apiClient);

    // Login as admin user
    await page.goto('/login');
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'test-password-123');
    await page.click('[data-testid="login-button"]');
    await expect(page).toHaveURL('/dashboard');
  });

  test('should display dashboard overview with statistics', async ({ page }) => {
    // Should show main dashboard sections
    await expect(page.locator('[data-testid="dashboard-header"]')).toBeVisible();
    await expect(page.locator('[data-testid="statistics-cards"]')).toBeVisible();
    await expect(page.locator('[data-testid="recent-scans-section"]')).toBeVisible();
    await expect(page.locator('[data-testid="findings-trend-chart"]')).toBeVisible();

    // Should show statistics cards
    await expect(page.locator('[data-testid="total-repositories-card"]')).toBeVisible();
    await expect(page.locator('[data-testid="active-scans-card"]')).toBeVisible();
    await expect(page.locator('[data-testid="total-findings-card"]')).toBeVisible();
    await expect(page.locator('[data-testid="critical-findings-card"]')).toBeVisible();

    // Statistics should have numeric values
    await expect(page.locator('[data-testid="total-repositories-count"]')).toContainText(/\d+/);
    await expect(page.locator('[data-testid="active-scans-count"]')).toContainText(/\d+/);
    await expect(page.locator('[data-testid="total-findings-count"]')).toContainText(/\d+/);
    await expect(page.locator('[data-testid="critical-findings-count"]')).toContainText(/\d+/);
  });

  test('should display recent scans table with correct data', async ({ page }) => {
    // Create test data
    const testRepo = await testDataManager.createTestRepository({
      name: 'dashboard-test-repo',
      language: 'javascript',
      hasVulnerabilities: true,
      vulnerabilityTypes: ['xss']
    });

    const testScan = await testDataManager.createTestScan(testRepo.id);
    await testDataManager.waitForScanCompletion(testScan.id);

    // Refresh dashboard
    await page.reload();

    // Should show recent scans table
    await expect(page.locator('[data-testid="recent-scans-table"]')).toBeVisible();
    
    // Should show table headers
    await expect(page.locator('[data-testid="scans-table-header"]')).toContainText('Repository');
    await expect(page.locator('[data-testid="scans-table-header"]')).toContainText('Status');
    await expect(page.locator('[data-testid="scans-table-header"]')).toContainText('Findings');
    await expect(page.locator('[data-testid="scans-table-header"]')).toContainText('Time');

    // Should show the test scan
    const scanRow = page.locator(`[data-testid="scan-row"]:has-text("${testRepo.name}")`);
    await expect(scanRow).toBeVisible();
    await expect(scanRow.locator('[data-testid="scan-status"]')).toContainText('completed');
    await expect(scanRow.locator('[data-testid="scan-findings-count"]')).toContainText(/\d+/);

    // Cleanup
    await testDataManager.deleteTestRepository(testRepo.name);
  });

  test('should display findings trend chart', async ({ page }) => {
    // Should show chart container
    await expect(page.locator('[data-testid="findings-trend-chart"]')).toBeVisible();
    
    // Should show chart title
    await expect(page.locator('[data-testid="chart-title"]')).toContainText('Findings Trend');
    
    // Should show chart legend
    await expect(page.locator('[data-testid="chart-legend"]')).toBeVisible();
    
    // Should show severity categories in legend
    await expect(page.locator('[data-testid="legend-critical"]')).toBeVisible();
    await expect(page.locator('[data-testid="legend-high"]')).toBeVisible();
    await expect(page.locator('[data-testid="legend-medium"]')).toBeVisible();
    await expect(page.locator('[data-testid="legend-low"]')).toBeVisible();

    // Chart should be rendered (check for SVG or canvas element)
    await expect(page.locator('[data-testid="findings-trend-chart"] svg, [data-testid="findings-trend-chart"] canvas')).toBeVisible();
  });

  test('should navigate to detailed views from dashboard', async ({ page }) => {
    // Click on repositories card
    await page.click('[data-testid="total-repositories-card"]');
    await expect(page).toHaveURL('/repositories');
    
    // Go back to dashboard
    await page.goto('/dashboard');
    
    // Click on scans card
    await page.click('[data-testid="active-scans-card"]');
    await expect(page).toHaveURL('/scans');
    
    // Go back to dashboard
    await page.goto('/dashboard');
    
    // Click on findings card
    await page.click('[data-testid="total-findings-card"]');
    await expect(page).toHaveURL('/findings');
  });

  test('should show quick actions section', async ({ page }) => {
    // Should show quick actions
    await expect(page.locator('[data-testid="quick-actions-section"]')).toBeVisible();
    
    // Should show action buttons
    await expect(page.locator('[data-testid="add-repository-action"]')).toBeVisible();
    await expect(page.locator('[data-testid="start-scan-action"]')).toBeVisible();
    await expect(page.locator('[data-testid="view-reports-action"]')).toBeVisible();
    
    // Test quick action navigation
    await page.click('[data-testid="add-repository-action"]');
    await expect(page).toHaveURL('/repositories/new');
  });

  test('should update statistics in real-time', async ({ page }) => {
    // Get initial statistics
    const initialRepoCount = await page.locator('[data-testid="total-repositories-count"]').textContent();
    
    // Create a new repository
    const testRepo = await testDataManager.createTestRepository({
      name: 'realtime-test-repo',
      language: 'python',
      hasVulnerabilities: false,
      vulnerabilityTypes: []
    });

    // Refresh dashboard
    await page.reload();
    
    // Statistics should be updated
    const updatedRepoCount = await page.locator('[data-testid="total-repositories-count"]').textContent();
    expect(parseInt(updatedRepoCount || '0')).toBeGreaterThan(parseInt(initialRepoCount || '0'));
    
    // Cleanup
    await testDataManager.deleteTestRepository(testRepo.name);
  });

  test('should handle empty state gracefully', async ({ page }) => {
    // For this test, we'd need a clean environment or mock empty responses
    // This would typically be done with API mocking
    
    // Mock empty API responses
    await page.route('**/api/v1/dashboard/statistics', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          total_repositories: 0,
          active_scans: 0,
          total_findings: 0,
          critical_findings: 0
        })
      });
    });

    await page.route('**/api/v1/scans/recent', route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([])
      });
    });

    await page.reload();

    // Should show zero statistics
    await expect(page.locator('[data-testid="total-repositories-count"]')).toContainText('0');
    await expect(page.locator('[data-testid="active-scans-count"]')).toContainText('0');
    
    // Should show empty state for recent scans
    await expect(page.locator('[data-testid="empty-scans-message"]')).toBeVisible();
    await expect(page.locator('[data-testid="empty-scans-message"]')).toContainText('No recent scans');
  });

  test('should be responsive on different screen sizes', async ({ page }) => {
    // Test desktop view (default)
    await expect(page.locator('[data-testid="statistics-cards"]')).toHaveCSS('display', 'grid');
    
    // Test tablet view
    await page.setViewportSize({ width: 768, height: 1024 });
    await expect(page.locator('[data-testid="statistics-cards"]')).toBeVisible();
    
    // Test mobile view
    await page.setViewportSize({ width: 375, height: 667 });
    await expect(page.locator('[data-testid="statistics-cards"]')).toBeVisible();
    
    // On mobile, cards should stack vertically
    const cardsContainer = page.locator('[data-testid="statistics-cards"]');
    await expect(cardsContainer).toHaveCSS('grid-template-columns', /1fr|none/);
  });

  test('should show loading states', async ({ page }) => {
    // Intercept API calls to add delay
    await page.route('**/api/v1/dashboard/statistics', route => {
      setTimeout(() => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            total_repositories: 5,
            active_scans: 2,
            total_findings: 42,
            critical_findings: 3
          })
        });
      }, 2000);
    });

    await page.reload();

    // Should show loading skeletons
    await expect(page.locator('[data-testid="statistics-loading"]')).toBeVisible();
    await expect(page.locator('[data-testid="chart-loading"]')).toBeVisible();
    
    // Loading should disappear after data loads
    await expect(page.locator('[data-testid="statistics-loading"]')).not.toBeVisible({ timeout: 5000 });
    await expect(page.locator('[data-testid="chart-loading"]')).not.toBeVisible({ timeout: 5000 });
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock API error
    await page.route('**/api/v1/dashboard/statistics', route => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal server error' })
      });
    });

    await page.reload();

    // Should show error state
    await expect(page.locator('[data-testid="dashboard-error"]')).toBeVisible();
    await expect(page.locator('[data-testid="dashboard-error"]')).toContainText('Failed to load dashboard data');
    
    // Should show retry button
    await expect(page.locator('[data-testid="retry-button"]')).toBeVisible();
  });

  test('should support keyboard navigation', async ({ page }) => {
    // Focus should start on first interactive element
    await page.keyboard.press('Tab');
    
    // Should be able to navigate through cards
    await expect(page.locator('[data-testid="total-repositories-card"]')).toBeFocused();
    
    await page.keyboard.press('Tab');
    await expect(page.locator('[data-testid="active-scans-card"]')).toBeFocused();
    
    // Should be able to activate cards with Enter
    await page.keyboard.press('Enter');
    await expect(page).toHaveURL('/scans');
  });
});