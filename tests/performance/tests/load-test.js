import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { randomString, randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('error_rate');
const scanDuration = new Trend('scan_duration');
const apiResponseTime = new Trend('api_response_time');
const authFailures = new Counter('auth_failures');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 }, // Ramp up to 10 users
    { duration: '5m', target: 10 }, // Stay at 10 users
    { duration: '2m', target: 20 }, // Ramp up to 20 users
    { duration: '5m', target: 20 }, // Stay at 20 users
    { duration: '2m', target: 0 },  // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<2000'], // 95% of requests must complete below 2s
    http_req_failed: ['rate<0.05'],    // Error rate must be below 5%
    error_rate: ['rate<0.05'],
    scan_duration: ['p(95)<300000'],   // 95% of scans must complete within 5 minutes
    api_response_time: ['p(90)<1000'], // 90% of API calls under 1s
  },
};

// Test data
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_BASE = `${BASE_URL}/api/v1`;

// Authentication tokens (would be generated in setup)
let authTokens = [];

export function setup() {
  console.log('Setting up performance test environment...');
  
  // Create test users and get auth tokens
  const users = [
    { username: 'perf_user_1', password: 'test_password_123' },
    { username: 'perf_user_2', password: 'test_password_123' },
    { username: 'perf_user_3', password: 'test_password_123' },
  ];
  
  const tokens = [];
  
  for (const user of users) {
    const loginResponse = http.post(`${API_BASE}/auth/login`, JSON.stringify(user), {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (loginResponse.status === 200) {
      const token = JSON.parse(loginResponse.body).token;
      tokens.push(token);
    }
  }
  
  console.log(`Created ${tokens.length} auth tokens for testing`);
  return { authTokens: tokens };
}

export default function(data) {
  const authToken = data.authTokens[randomIntBetween(0, data.authTokens.length - 1)];
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${authToken}`,
  };
  
  // Test scenario: Mixed workload simulation
  const scenario = randomIntBetween(1, 100);
  
  if (scenario <= 40) {
    // 40% - Dashboard and navigation
    testDashboardLoad(headers);
  } else if (scenario <= 70) {
    // 30% - Repository management
    testRepositoryOperations(headers);
  } else if (scenario <= 85) {
    // 15% - Scan operations
    testScanOperations(headers);
  } else if (scenario <= 95) {
    // 10% - Results viewing
    testResultsViewing(headers);
  } else {
    // 5% - Heavy operations (exports, reports)
    testHeavyOperations(headers);
  }
  
  sleep(randomIntBetween(1, 3));
}

function testDashboardLoad(headers) {
  const startTime = Date.now();
  
  // Load dashboard statistics
  const statsResponse = http.get(`${API_BASE}/dashboard/statistics`, { headers });
  check(statsResponse, {
    'dashboard stats loaded': (r) => r.status === 200,
    'dashboard stats response time OK': (r) => r.timings.duration < 1000,
  });
  
  if (statsResponse.status !== 200) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }
  
  // Load recent scans
  const scansResponse = http.get(`${API_BASE}/scans/recent?limit=10`, { headers });
  check(scansResponse, {
    'recent scans loaded': (r) => r.status === 200,
  });
  
  // Load findings trend data
  const trendResponse = http.get(`${API_BASE}/dashboard/findings-trend?days=30`, { headers });
  check(trendResponse, {
    'findings trend loaded': (r) => r.status === 200,
  });
  
  const totalTime = Date.now() - startTime;
  apiResponseTime.add(totalTime);
}

function testRepositoryOperations(headers) {
  const startTime = Date.now();
  
  // List repositories
  const listResponse = http.get(`${API_BASE}/repositories`, { headers });
  check(listResponse, {
    'repositories listed': (r) => r.status === 200,
  });
  
  if (listResponse.status === 200) {
    const repositories = JSON.parse(listResponse.body);
    
    if (repositories.length > 0) {
      // Get details for a random repository
      const randomRepo = repositories[randomIntBetween(0, repositories.length - 1)];
      const detailResponse = http.get(`${API_BASE}/repositories/${randomRepo.id}`, { headers });
      
      check(detailResponse, {
        'repository details loaded': (r) => r.status === 200,
      });
      
      // Get repository scan history
      const historyResponse = http.get(`${API_BASE}/repositories/${randomRepo.id}/scans`, { headers });
      check(historyResponse, {
        'repository scan history loaded': (r) => r.status === 200,
      });
    }
  }
  
  const totalTime = Date.now() - startTime;
  apiResponseTime.add(totalTime);
}

function testScanOperations(headers) {
  const startTime = Date.now();
  
  // Get repositories first
  const reposResponse = http.get(`${API_BASE}/repositories`, { headers });
  
  if (reposResponse.status === 200) {
    const repositories = JSON.parse(reposResponse.body);
    
    if (repositories.length > 0) {
      const randomRepo = repositories[randomIntBetween(0, repositories.length - 1)];
      
      // Submit a scan
      const scanPayload = {
        repository_id: randomRepo.id,
        scan_type: 'incremental',
        agents: ['semgrep', 'eslint'],
      };
      
      const scanResponse = http.post(`${API_BASE}/scans`, JSON.stringify(scanPayload), { headers });
      check(scanResponse, {
        'scan submitted': (r) => r.status === 201,
      });
      
      if (scanResponse.status === 201) {
        const scan = JSON.parse(scanResponse.body);
        
        // Poll scan status (simulate monitoring)
        for (let i = 0; i < 5; i++) {
          sleep(2);
          const statusResponse = http.get(`${API_BASE}/scans/${scan.id}`, { headers });
          check(statusResponse, {
            'scan status checked': (r) => r.status === 200,
          });
          
          if (statusResponse.status === 200) {
            const scanStatus = JSON.parse(statusResponse.body);
            if (scanStatus.status === 'completed' || scanStatus.status === 'failed') {
              const scanTime = Date.now() - startTime;
              scanDuration.add(scanTime);
              break;
            }
          }
        }
      }
    }
  }
  
  const totalTime = Date.now() - startTime;
  apiResponseTime.add(totalTime);
}

function testResultsViewing(headers) {
  const startTime = Date.now();
  
  // Get completed scans
  const scansResponse = http.get(`${API_BASE}/scans?status=completed&limit=10`, { headers });
  check(scansResponse, {
    'completed scans loaded': (r) => r.status === 200,
  });
  
  if (scansResponse.status === 200) {
    const scans = JSON.parse(scansResponse.body);
    
    if (scans.length > 0) {
      const randomScan = scans[randomIntBetween(0, scans.length - 1)];
      
      // Get scan results
      const resultsResponse = http.get(`${API_BASE}/scans/${randomScan.id}/results`, { headers });
      check(resultsResponse, {
        'scan results loaded': (r) => r.status === 200,
      });
      
      // Get findings with pagination
      const findingsResponse = http.get(`${API_BASE}/scans/${randomScan.id}/findings?page=1&limit=50`, { headers });
      check(findingsResponse, {
        'findings loaded': (r) => r.status === 200,
      });
      
      if (findingsResponse.status === 200) {
        const findings = JSON.parse(findingsResponse.body);
        
        if (findings.length > 0) {
          // Get details for a random finding
          const randomFinding = findings[randomIntBetween(0, findings.length - 1)];
          const findingResponse = http.get(`${API_BASE}/findings/${randomFinding.id}`, { headers });
          
          check(findingResponse, {
            'finding details loaded': (r) => r.status === 200,
          });
        }
      }
    }
  }
  
  const totalTime = Date.now() - startTime;
  apiResponseTime.add(totalTime);
}

function testHeavyOperations(headers) {
  const startTime = Date.now();
  
  // Get completed scans for export
  const scansResponse = http.get(`${API_BASE}/scans?status=completed&limit=5`, { headers });
  
  if (scansResponse.status === 200) {
    const scans = JSON.parse(scansResponse.body);
    
    if (scans.length > 0) {
      const randomScan = scans[randomIntBetween(0, scans.length - 1)];
      
      // Export scan results (JSON format for performance)
      const exportResponse = http.get(`${API_BASE}/scans/${randomScan.id}/export?format=json`, { headers });
      check(exportResponse, {
        'scan export completed': (r) => r.status === 200,
        'export response time acceptable': (r) => r.timings.duration < 10000, // 10s max for exports
      });
      
      // Generate summary report
      const reportResponse = http.get(`${API_BASE}/scans/${randomScan.id}/report`, { headers });
      check(reportResponse, {
        'scan report generated': (r) => r.status === 200,
      });
    }
  }
  
  const totalTime = Date.now() - startTime;
  apiResponseTime.add(totalTime);
}

export function teardown(data) {
  console.log('Cleaning up performance test environment...');
  
  // Cleanup would go here if needed
  // For now, just log completion
  console.log('Performance test completed');
}