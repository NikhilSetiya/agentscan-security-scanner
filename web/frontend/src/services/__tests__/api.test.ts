import { describe, it, expect, beforeEach, vi } from 'vitest';
import { apiClient } from '../api';

// Mock fetch globally
global.fetch = vi.fn();

describe('API Client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
  });

  describe('Authentication', () => {
    it('should login successfully and store token', async () => {
      const mockResponse = {
        token: 'test-jwt-token',
        user: {
          id: '1',
          username: 'testuser',
          email: 'test@example.com',
          role: 'developer' as const,
        },
        expires_at: '2024-12-31T23:59:59Z',
      };

      (fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/json',
        },
        json: async () => mockResponse,
      });

      const result = await apiClient.login({
        username: 'testuser',
        password: 'password123',
      });

      expect(result.data).toEqual(mockResponse);
      expect(result.error).toBeUndefined();
      expect(localStorage.getItem('auth_token')).toBe('test-jwt-token');
    });

    it('should handle login failure', async () => {
      const mockError = {
        error: 'Invalid credentials',
        code: 'AUTH_FAILED',
      };

      (fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 401,
        headers: {
          get: () => 'application/json',
        },
        json: async () => mockError,
      });

      const result = await apiClient.login({
        username: 'testuser',
        password: 'wrongpassword',
      });

      expect(result.data).toBeUndefined();
      expect(result.error).toEqual(mockError);
      expect(localStorage.getItem('auth_token')).toBeNull();
    });
  });

  describe('Dashboard Stats', () => {
    it('should fetch dashboard stats successfully', async () => {
      const mockStats = {
        total_scans: 100,
        total_repositories: 25,
        findings_by_severity: {
          critical: 5,
          high: 15,
          medium: 30,
          low: 20,
          info: 10,
        },
        recent_scans: [],
        trend_data: [],
      };

      (fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/json',
        },
        json: async () => mockStats,
      });

      const result = await apiClient.getDashboardStats();

      expect(result.data).toEqual(mockStats);
      expect(result.error).toBeUndefined();
    });
  });

  describe('Error Handling', () => {
    it('should handle network errors', async () => {
      (fetch as any).mockRejectedValueOnce(new Error('Network error'));

      const result = await apiClient.healthCheck();

      expect(result.data).toBeUndefined();
      expect(result.error).toEqual({
        error: 'Network error',
        code: 'NETWORK_ERROR',
      });
    });

    it('should handle timeout errors', async () => {
      // Create an AbortError to simulate timeout
      const abortError = new Error('The operation was aborted.');
      abortError.name = 'AbortError';

      (fetch as any).mockRejectedValueOnce(abortError);

      const result = await apiClient.healthCheck();

      expect(result.error?.code).toBe('TIMEOUT');
      expect(result.error?.error).toBe('Request timeout');
    });

    it('should clear auth token on 401 response', async () => {
      // Set a token first
      localStorage.setItem('auth_token', 'test-token');

      (fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 401,
        headers: {
          get: () => 'application/json',
        },
        json: async () => ({ error: 'Unauthorized', code: 'AUTH_REQUIRED' }),
      });

      await apiClient.getDashboardStats();

      expect(localStorage.getItem('auth_token')).toBeNull();
    });
  });

  describe('Request Headers', () => {
    it('should include auth token in requests when available', async () => {
      // Set token and simulate a successful login to update the client's token
      const mockLoginResponse = {
        token: 'test-token',
        user: {
          id: '1',
          username: 'testuser',
          email: 'test@example.com',
          role: 'developer' as const,
        },
        expires_at: '2024-12-31T23:59:59Z',
      };

      (fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/json',
        },
        json: async () => mockLoginResponse,
      });

      // Login to set the token
      await apiClient.login({ username: 'test', password: 'test' });

      // Clear the mock and set up for the actual test
      vi.clearAllMocks();

      (fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/json',
        },
        json: async () => ({}),
      });

      await apiClient.healthCheck();

      expect(fetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          headers: expect.objectContaining({
            'Authorization': 'Bearer test-token',
          }),
        })
      );
    });

    it('should not include auth header when no token is available', async () => {
      // Ensure no token is set by logging out first
      await apiClient.logout();

      // Clear the mock from logout call
      vi.clearAllMocks();

      (fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/json',
        },
        json: async () => ({ message: 'Logged out' }),
      });

      (fetch as any).mockResolvedValueOnce({
        ok: true,
        status: 200,
        headers: {
          get: () => 'application/json',
        },
        json: async () => ({}),
      });

      await apiClient.healthCheck();

      const callArgs = (fetch as any).mock.calls[0][1];
      expect(callArgs.headers).not.toHaveProperty('Authorization');
    });
  });
});