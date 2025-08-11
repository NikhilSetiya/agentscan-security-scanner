import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics for stress testing
const errorRate = new Rate('stress_error_rate');
const responseTime = new Trend('stress_response_time');
const concurrentScans = new Counter('concurrent_scans');
const systemRecovery = new Trend('system_recovery_time');

export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Normal load
    { duration: '5m', target: 50 },   // Stress load
    { duration: '2m', target: 100 },  // High stress
    { duration: '5m', target: 100 },  // Sustained high stress
    { duration: '2m', target: 200 },  // Breaking point
    { duration: '5m', target: 200 },  // Sustained breaking point
    { duration: '10m', target: 0 },   // Recovery
  ],
  thresholds: {
    http_req_duration: ['p(99)<5000'], // 99% under 5s during stress
    http_req_failed: ['rate<0.1'],     // Error rate under 10% during stress
    stress_error_rate: ['rate<0.15'],  // Allow higher error rate during stress
    system_recovery_time: ['p(95)<30000'], // System should recover within 30s
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_BASE = `${BASE_URL}/api/v1`;

let authTokens = [];
let testRepositories = [];

export function setup() {
  console.log('Setting up stress test environment...');
  
  // Create multiple test users
  const users = [];
  for (let i = 1; i <= 10; i++) {
    users.push({
      username: `stress_user_${i}`,
      password: 'stress_test_password_123'
    });
  }
  
  const tokens = [];
  for (const user of users) {
    const loginResponse = http.post(`${API_BASE}/auth/login`, JSON.stringify(user), {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (loginResponse.status === 200) {
      tokens.push(JSON.parse(loginResponse.body).token);
    }
  }
  
  // Create test repositories for stress testing
  const repositories = [];
  const authHeader = { 
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${tokens[0]}`
  };
  
  for (let i = 1; i <= 20; i++) {
    const repoPayload = {
      name: `stress-test-repo-${i}`,
      url: `https://github.com/test/stress-repo-${i}`,
      language: i % 2 === 0 ? 'javascript' : 'python',
      branch: 'main'
    };
    
    const repoResponse = http.post(`${API_BASE}/repositories`, JSON.stringify(repoPayload), {
      headers: authHeader
    });
    
    if (repoResponse.status === 201) {
      repositories.push(JSON.parse(repoResponse.body));
    }
  }
  
  console.log(`Created ${tokens.length} users and ${repositories.length} repositories for stress testing`);
  return { authTokens: tokens, repositories };
}

export default function(data) {
  const authToken = data.authTokens[randomIntBetween(0, data.authTokens.length - 1)];
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${authToken}`,
  };
  
  const startTime = Date.now();
  
  // Stress test scenarios
  const scenario = randomIntBetween(1, 100);
  
  if (scenario <= 30) {
    // 30% - Concurrent scan submissions (high load)
    stressConcurrentScans(data, headers);
  } else if (scenario <= 50) {
    // 20% - Heavy API usage
    stressHeavyApiUsage(headers);
  } else if (scenario <= 70) {
    // 20% - Database intensive operations
    stressDatabaseOperations(data, headers);
  } else if (scenario <= 85) {
    // 15% - Memory intensive operations
    stressMemoryOperations(data, headers);
  } else {
    // 15% - Mixed load simulation
    stressMixedLoad(data, headers);
  }
  
  const totalTime = Date.now() - startTime;
  responseTime.add(totalTime);
  
  // Shorter sleep during stress test
  sleep(randomIntBetween(0.1, 1));
}

function stressConcurrentScans(data, headers) {
  if (data.repositories.length === 0) return;
  
  const repository = data.repositories[randomIntBetween(0, data.repositories.length - 1)];
  
  // Submit multiple scans rapidly
  const scanPayloads = [
    {
      repository_id: repository.id,
      scan_type: 'full',
      agents: ['semgrep', 'eslint', 'bandit']
    },
    {
      repository_id: repository.id,
      scan_type: 'incremental',
      agents: ['semgrep']
    }
  ];
  
  for (const payload of scanPayloads) {
    const scanResponse = http.post(`${API_BASE}/scans`, JSON.stringify(payload), { headers });
    
    const success = check(scanResponse, {
      'concurrent scan submitted': (r) => r.status === 201 || r.status === 429, // Accept rate limiting
    });
    
    if (scanResponse.status === 201) {
      concurrentScans.add(1);
    }
    
    errorRate.add(success ? 0 : 1);
  }
}

function stressHeavyApiUsage(headers) {
  // Rapid-fire API requests
  const endpoints = [
    '/dashboard/statistics',
    '/repositories',
    '/scans?limit=100',
    '/findings?limit=100',
    '/organizations',
    '/users/profile'
  ];
  
  for (const endpoint of endpoints) {
    const response = http.get(`${API_BASE}${endpoint}`, { headers });
    
    const success = check(response, {
      [`heavy API ${endpoint} responded`]: (r) => r.status < 500,
    });
    
    errorRate.add(success ? 0 : 1);
  }
}

function stressDatabaseOperations(data, headers) {
  // Operations that stress the database
  
  // 1. Large pagination requests
  const largePageResponse = http.get(`${API_BASE}/findings?page=1&limit=1000`, { headers });
  check(largePageResponse, {
    'large pagination handled': (r) => r.status < 500,
  });
  
  // 2. Complex filtering
  const complexFilterResponse = http.get(
    `${API_BASE}/findings?severity=high,critical&status=open&tool=semgrep&sort=created_at&order=desc&limit=500`,
    { headers }
  );
  check(complexFilterResponse, {
    'complex filtering handled': (r) => r.status < 500,
  });
  
  // 3. Aggregation queries
  const statsResponse = http.get(`${API_BASE}/dashboard/statistics?detailed=true`, { headers });
  check(statsResponse, {
    'aggregation queries handled': (r) => r.status < 500,
  });
  
  errorRate.add(largePageResponse.status >= 500 ? 1 : 0);
  errorRate.add(complexFilterResponse.status >= 500 ? 1 : 0);
  errorRate.add(statsResponse.status >= 500 ? 1 : 0);
}

function stressMemoryOperations(data, headers) {
  if (data.repositories.length === 0) return;
  
  // Operations that consume memory
  
  // 1. Large export operations
  const repository = data.repositories[randomIntBetween(0, data.repositories.length - 1)];
  const scansResponse = http.get(`${API_BASE}/repositories/${repository.id}/scans?limit=1`, { headers });
  
  if (scansResponse.status === 200) {
    const scans = JSON.parse(scansResponse.body);
    if (scans.length > 0) {
      const exportResponse = http.get(`${API_BASE}/scans/${scans[0].id}/export?format=json&include_raw=true`, { headers });
      check(exportResponse, {
        'large export handled': (r) => r.status < 500,
      });
      
      errorRate.add(exportResponse.status >= 500 ? 1 : 0);
    }
  }
  
  // 2. Bulk operations
  const bulkUpdatePayload = {
    finding_ids: Array.from({ length: 100 }, (_, i) => `finding-${i}`),
    status: 'reviewed'
  };
  
  const bulkResponse = http.patch(`${API_BASE}/findings/bulk-update`, JSON.stringify(bulkUpdatePayload), { headers });
  check(bulkResponse, {
    'bulk operation handled': (r) => r.status < 500,
  });
  
  errorRate.add(bulkResponse.status >= 500 ? 1 : 0);
}

function stressMixedLoad(data, headers) {
  // Simulate realistic mixed load under stress
  
  // Quick dashboard check
  const dashResponse = http.get(`${API_BASE}/dashboard/statistics`, { headers });
  
  // Repository operations
  const reposResponse = http.get(`${API_BASE}/repositories?limit=50`, { headers });
  
  // Scan status checks
  const scansResponse = http.get(`${API_BASE}/scans?status=running&limit=20`, { headers });
  
  // Finding operations
  const findingsResponse = http.get(`${API_BASE}/findings?limit=100`, { headers });
  
  const responses = [dashResponse, reposResponse, scansResponse, findingsResponse];
  
  for (const response of responses) {
    const success = check(response, {
      'mixed load response OK': (r) => r.status < 500,
    });
    
    errorRate.add(success ? 0 : 1);
  }
}

export function teardown(data) {
  console.log('Cleaning up stress test environment...');
  
  // Measure system recovery time
  const recoveryStart = Date.now();
  
  // Test if system is responsive after stress
  let recovered = false;
  let attempts = 0;
  const maxAttempts = 30; // 30 seconds max
  
  while (!recovered && attempts < maxAttempts) {
    const healthResponse = http.get(`${BASE_URL}/health`);
    
    if (healthResponse.status === 200) {
      const recoveryTime = Date.now() - recoveryStart;
      systemRecovery.add(recoveryTime);
      recovered = true;
      console.log(`System recovered in ${recoveryTime}ms`);
    } else {
      sleep(1);
      attempts++;
    }
  }
  
  if (!recovered) {
    console.log('System did not recover within timeout period');
    systemRecovery.add(30000); // Max recovery time
  }
  
  // Cleanup test repositories
  if (data.authTokens.length > 0) {
    const authHeader = {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${data.authTokens[0]}`
    };
    
    for (const repo of data.repositories) {
      try {
        http.del(`${API_BASE}/repositories/${repo.id}`, null, { headers: authHeader });
      } catch (error) {
        console.log(`Failed to cleanup repository ${repo.id}: ${error}`);
      }
    }
  }
  
  console.log('Stress test cleanup completed');
}