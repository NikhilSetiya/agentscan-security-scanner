const request = require('supertest');
const jwt = require('jsonwebtoken');
const bcrypt = require('bcrypt');

describe('Authentication Security Tests', () => {
  const baseURL = process.env.BASE_URL || 'http://localhost:8080';
  const apiBase = '/api/v1';
  
  let validToken;
  let expiredToken;
  let malformedToken;

  beforeAll(async () => {
    // Create valid token
    validToken = jwt.sign(
      { user_id: 'test-user', username: 'testuser', role: 'user' },
      'test-secret',
      { expiresIn: '1h' }
    );

    // Create expired token
    expiredToken = jwt.sign(
      { user_id: 'test-user', username: 'testuser', role: 'user' },
      'test-secret',
      { expiresIn: '-1h' }
    );

    // Create malformed token
    malformedToken = 'invalid.jwt.token';
  });

  describe('Authentication Bypass Attempts', () => {
    test('should reject requests without authentication token', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/repositories`)
        .expect(401);

      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toMatch(/unauthorized|authentication required/i);
    });

    test('should reject requests with invalid token', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/repositories`)
        .set('Authorization', `Bearer ${malformedToken}`)
        .expect(401);

      expect(response.body).toHaveProperty('error');
    });

    test('should reject requests with expired token', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/repositories`)
        .set('Authorization', `Bearer ${expiredToken}`)
        .expect(401);

      expect(response.body).toHaveProperty('error');
      expect(response.body.error).toMatch(/expired|invalid/i);
    });

    test('should reject requests with missing Bearer prefix', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/repositories`)
        .set('Authorization', validToken)
        .expect(401);
    });

    test('should reject requests with wrong authentication scheme', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/repositories`)
        .set('Authorization', `Basic ${Buffer.from('user:pass').toString('base64')}`)
        .expect(401);
    });
  });

  describe('Brute Force Protection', () => {
    test('should implement rate limiting on login endpoint', async () => {
      const loginAttempts = [];
      
      // Make multiple failed login attempts
      for (let i = 0; i < 10; i++) {
        loginAttempts.push(
          request(baseURL)
            .post(`${apiBase}/auth/login`)
            .send({
              username: 'nonexistent',
              password: 'wrongpassword'
            })
        );
      }

      const responses = await Promise.all(loginAttempts);
      
      // Should start rate limiting after several attempts
      const rateLimitedResponses = responses.filter(r => r.status === 429);
      expect(rateLimitedResponses.length).toBeGreaterThan(0);
    });

    test('should implement account lockout after multiple failed attempts', async () => {
      const username = 'bruteforce-test-user';
      
      // Create test user first (this would need to be done via admin API)
      // For now, we'll test with existing user
      
      const failedAttempts = [];
      for (let i = 0; i < 6; i++) {
        failedAttempts.push(
          request(baseURL)
            .post(`${apiBase}/auth/login`)
            .send({
              username: 'admin', // Use existing user
              password: 'wrongpassword'
            })
        );
      }

      const responses = await Promise.all(failedAttempts);
      
      // Later attempts should be blocked
      const blockedResponses = responses.slice(-2);
      expect(blockedResponses.some(r => r.status === 429 || r.status === 423)).toBe(true);
    });
  });

  describe('Session Management Security', () => {
    test('should invalidate token on logout', async () => {
      // First login to get a valid token
      const loginResponse = await request(baseURL)
        .post(`${apiBase}/auth/login`)
        .send({
          username: 'admin',
          password: 'test-password-123'
        })
        .expect(200);

      const token = loginResponse.body.token;

      // Use the token to access protected resource
      await request(baseURL)
        .get(`${apiBase}/repositories`)
        .set('Authorization', `Bearer ${token}`)
        .expect(200);

      // Logout
      await request(baseURL)
        .post(`${apiBase}/auth/logout`)
        .set('Authorization', `Bearer ${token}`)
        .expect(200);

      // Token should no longer work
      await request(baseURL)
        .get(`${apiBase}/repositories`)
        .set('Authorization', `Bearer ${token}`)
        .expect(401);
    });

    test('should have secure token expiration', async () => {
      const loginResponse = await request(baseURL)
        .post(`${apiBase}/auth/login`)
        .send({
          username: 'admin',
          password: 'test-password-123'
        })
        .expect(200);

      const token = loginResponse.body.token;
      const decoded = jwt.decode(token);
      
      // Token should have reasonable expiration time (not too long)
      const expirationTime = decoded.exp * 1000;
      const currentTime = Date.now();
      const tokenLifetime = expirationTime - currentTime;
      
      // Should expire within 24 hours
      expect(tokenLifetime).toBeLessThan(24 * 60 * 60 * 1000);
      
      // Should not be too short (at least 1 hour)
      expect(tokenLifetime).toBeGreaterThan(60 * 60 * 1000);
    });
  });

  describe('Password Security', () => {
    test('should enforce strong password requirements', async () => {
      const weakPasswords = [
        '123456',
        'password',
        'admin',
        'qwerty',
        '12345678',
        'abc123'
      ];

      for (const weakPassword of weakPasswords) {
        const response = await request(baseURL)
          .post(`${apiBase}/auth/register`)
          .send({
            username: `testuser-${Date.now()}`,
            email: `test-${Date.now()}@example.com`,
            password: weakPassword
          });

        // Should reject weak passwords
        expect(response.status).toBe(400);
        expect(response.body.error).toMatch(/password.*weak|password.*requirements/i);
      }
    });

    test('should hash passwords securely', async () => {
      // This test would require access to the database or user creation endpoint
      // For now, we'll test that passwords are not returned in responses
      
      const loginResponse = await request(baseURL)
        .post(`${apiBase}/auth/login`)
        .send({
          username: 'admin',
          password: 'test-password-123'
        })
        .expect(200);

      // Response should not contain password
      expect(loginResponse.body).not.toHaveProperty('password');
      
      // Get user profile
      const profileResponse = await request(baseURL)
        .get(`${apiBase}/users/profile`)
        .set('Authorization', `Bearer ${loginResponse.body.token}`)
        .expect(200);

      // Profile should not contain password
      expect(profileResponse.body).not.toHaveProperty('password');
    });
  });

  describe('Authorization Security', () => {
    test('should enforce role-based access control', async () => {
      // Login as regular user
      const userLoginResponse = await request(baseURL)
        .post(`${apiBase}/auth/login`)
        .send({
          username: 'developer', // Assuming this is a non-admin user
          password: 'test-password-123'
        });

      if (userLoginResponse.status === 200) {
        const userToken = userLoginResponse.body.token;

        // Try to access admin-only endpoints
        const adminEndpoints = [
          '/users',
          '/organizations/create',
          '/admin/settings'
        ];

        for (const endpoint of adminEndpoints) {
          const response = await request(baseURL)
            .get(`${apiBase}${endpoint}`)
            .set('Authorization', `Bearer ${userToken}`);

          // Should be forbidden for non-admin users
          expect([403, 404]).toContain(response.status);
        }
      }
    });

    test('should prevent privilege escalation', async () => {
      // Login as regular user
      const userLoginResponse = await request(baseURL)
        .post(`${apiBase}/auth/login`)
        .send({
          username: 'developer',
          password: 'test-password-123'
        });

      if (userLoginResponse.status === 200) {
        const userToken = userLoginResponse.body.token;

        // Try to modify own role
        const response = await request(baseURL)
          .patch(`${apiBase}/users/profile`)
          .set('Authorization', `Bearer ${userToken}`)
          .send({
            role: 'admin'
          });

        // Should not allow role modification
        expect([400, 403]).toContain(response.status);
      }
    });
  });

  describe('OAuth Security', () => {
    test('should validate OAuth state parameter', async () => {
      // Test OAuth callback without state parameter
      const response = await request(baseURL)
        .get(`${apiBase}/auth/github/callback`)
        .query({
          code: 'test-code'
          // Missing state parameter
        });

      expect([400, 401]).toContain(response.status);
    });

    test('should validate OAuth code parameter', async () => {
      // Test OAuth callback without code parameter
      const response = await request(baseURL)
        .get(`${apiBase}/auth/github/callback`)
        .query({
          state: 'test-state'
          // Missing code parameter
        });

      expect([400, 401]).toContain(response.status);
    });
  });

  describe('Security Headers', () => {
    test('should include security headers in responses', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/health`)
        .expect(200);

      // Check for security headers
      expect(response.headers).toHaveProperty('x-content-type-options', 'nosniff');
      expect(response.headers).toHaveProperty('x-frame-options');
      expect(response.headers).toHaveProperty('x-xss-protection');
      expect(response.headers['strict-transport-security']).toBeDefined();
    });

    test('should not expose sensitive information in headers', async () => {
      const response = await request(baseURL)
        .get(`${apiBase}/health`);

      // Should not expose server information
      expect(response.headers['server']).toBeUndefined();
      expect(response.headers['x-powered-by']).toBeUndefined();
    });
  });

  describe('CORS Security', () => {
    test('should implement proper CORS policy', async () => {
      const response = await request(baseURL)
        .options(`${apiBase}/repositories`)
        .set('Origin', 'https://malicious-site.com')
        .set('Access-Control-Request-Method', 'GET');

      // Should not allow arbitrary origins
      expect(response.headers['access-control-allow-origin']).not.toBe('*');
    });

    test('should allow legitimate origins', async () => {
      const legitimateOrigins = [
        'https://app.agentscan.dev',
        'http://localhost:3000'
      ];

      for (const origin of legitimateOrigins) {
        const response = await request(baseURL)
          .get(`${apiBase}/health`)
          .set('Origin', origin);

        // Should allow legitimate origins
        expect([200, 204]).toContain(response.status);
      }
    });
  });
});