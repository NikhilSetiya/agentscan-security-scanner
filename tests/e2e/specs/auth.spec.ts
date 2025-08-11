import { test, expect } from '@playwright/test';
import { ApiClient } from '../utils/api-client';

test.describe('Authentication Flow', () => {
  let apiClient: ApiClient;

  test.beforeEach(async ({ page }) => {
    apiClient = new ApiClient(page.url());
  });

  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/login');

    // Fill login form
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'test-password-123');
    
    // Submit form
    await page.click('[data-testid="login-button"]');

    // Should redirect to dashboard
    await expect(page).toHaveURL('/dashboard');
    
    // Should show user menu
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
    
    // Should display welcome message
    await expect(page.locator('text=Welcome back')).toBeVisible();
  });

  test('should show error for invalid credentials', async ({ page }) => {
    await page.goto('/login');

    // Fill with invalid credentials
    await page.fill('[data-testid="username-input"]', 'invalid');
    await page.fill('[data-testid="password-input"]', 'wrong-password');
    
    // Submit form
    await page.click('[data-testid="login-button"]');

    // Should show error message
    await expect(page.locator('[data-testid="error-message"]')).toContainText('Invalid credentials');
    
    // Should remain on login page
    await expect(page).toHaveURL('/login');
  });

  test('should logout successfully', async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'test-password-123');
    await page.click('[data-testid="login-button"]');
    
    await expect(page).toHaveURL('/dashboard');

    // Logout
    await page.click('[data-testid="user-menu"]');
    await page.click('[data-testid="logout-button"]');

    // Should redirect to login page
    await expect(page).toHaveURL('/login');
    
    // Should show logout success message
    await expect(page.locator('text=Successfully logged out')).toBeVisible();
  });

  test('should redirect to login when accessing protected route', async ({ page }) => {
    // Try to access dashboard without authentication
    await page.goto('/dashboard');

    // Should redirect to login
    await expect(page).toHaveURL('/login');
    
    // Should show message about authentication required
    await expect(page.locator('text=Please log in to continue')).toBeVisible();
  });

  test('should handle session expiration', async ({ page }) => {
    // Login first
    await page.goto('/login');
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'test-password-123');
    await page.click('[data-testid="login-button"]');
    
    await expect(page).toHaveURL('/dashboard');

    // Simulate expired token by clearing localStorage
    await page.evaluate(() => {
      localStorage.removeItem('auth_token');
    });

    // Try to make an authenticated request
    await page.click('[data-testid="repositories-link"]');

    // Should redirect to login due to expired session
    await expect(page).toHaveURL('/login');
    await expect(page.locator('text=Session expired')).toBeVisible();
  });

  test('should remember login state after page refresh', async ({ page }) => {
    // Login
    await page.goto('/login');
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.fill('[data-testid="password-input"]', 'test-password-123');
    await page.click('[data-testid="login-button"]');
    
    await expect(page).toHaveURL('/dashboard');

    // Refresh page
    await page.reload();

    // Should still be logged in
    await expect(page).toHaveURL('/dashboard');
    await expect(page.locator('[data-testid="user-menu"]')).toBeVisible();
  });

  test('should handle OAuth login flow', async ({ page }) => {
    await page.goto('/login');

    // Click GitHub OAuth button
    await page.click('[data-testid="github-login-button"]');

    // Should redirect to GitHub OAuth (in test environment, this might be mocked)
    // For E2E tests, you might want to mock the OAuth flow
    await expect(page).toHaveURL(/github\.com\/login\/oauth/);
  });

  test('should validate form inputs', async ({ page }) => {
    await page.goto('/login');

    // Try to submit empty form
    await page.click('[data-testid="login-button"]');

    // Should show validation errors
    await expect(page.locator('[data-testid="username-error"]')).toContainText('Username is required');
    await expect(page.locator('[data-testid="password-error"]')).toContainText('Password is required');

    // Fill username only
    await page.fill('[data-testid="username-input"]', 'admin');
    await page.click('[data-testid="login-button"]');

    // Should still show password error
    await expect(page.locator('[data-testid="password-error"]')).toContainText('Password is required');
    
    // Username error should be gone
    await expect(page.locator('[data-testid="username-error"]')).not.toBeVisible();
  });

  test('should handle rate limiting', async ({ page }) => {
    await page.goto('/login');

    // Make multiple failed login attempts
    for (let i = 0; i < 6; i++) {
      await page.fill('[data-testid="username-input"]', 'admin');
      await page.fill('[data-testid="password-input"]', 'wrong-password');
      await page.click('[data-testid="login-button"]');
      
      // Wait a bit between attempts
      await page.waitForTimeout(500);
    }

    // Should show rate limiting message
    await expect(page.locator('[data-testid="error-message"]')).toContainText('Too many failed attempts');
    
    // Login button should be disabled
    await expect(page.locator('[data-testid="login-button"]')).toBeDisabled();
  });

  test('should support password visibility toggle', async ({ page }) => {
    await page.goto('/login');

    const passwordInput = page.locator('[data-testid="password-input"]');
    const toggleButton = page.locator('[data-testid="password-toggle"]');

    // Password should be hidden by default
    await expect(passwordInput).toHaveAttribute('type', 'password');

    // Click toggle to show password
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');

    // Click toggle to hide password again
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });
});