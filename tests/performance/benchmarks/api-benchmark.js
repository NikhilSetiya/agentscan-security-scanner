const autocannon = require('autocannon');
const axios = require('axios');
const { v4: uuidv4 } = require('uuid');

class ApiBenchmark {
  constructor(baseUrl = 'http://localhost:8080') {
    this.baseUrl = baseUrl;
    this.apiBase = `${baseUrl}/api/v1`;
    this.authToken = null;
  }

  async setup() {
    console.log('Setting up API benchmark...');
    
    // Login to get auth token
    try {
      const loginResponse = await axios.post(`${this.apiBase}/auth/login`, {
        username: 'admin',
        password: 'test-password-123'
      });
      
      this.authToken = loginResponse.data.token;
      console.log('✅ Authentication successful');
    } catch (error) {
      console.error('❌ Authentication failed:', error.message);
      throw error;
    }
  }

  getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${this.authToken}`
    };
  }

  async benchmarkEndpoint(name, url, options = {}) {
    console.log(`\n🚀 Benchmarking ${name}...`);
    
    const defaultOptions = {
      url,
      connections: 10,
      pipelining: 1,
      duration: 30,
      headers: this.getHeaders(),
      ...options
    };

    return new Promise((resolve, reject) => {
      const instance = autocannon(defaultOptions, (err, result) => {
        if (err) {
          reject(err);
        } else {
          this.printResults(name, result);
          resolve(result);
        }
      });

      // Handle the instance if needed
      autocannon.track(instance);
    });
  }

  printResults(name, result) {
    console.log(`\n📊 Results for ${name}:`);
    console.log(`   Requests/sec: ${result.requests.average.toFixed(2)}`);
    console.log(`   Latency (avg): ${result.latency.average.toFixed(2)}ms`);
    console.log(`   Latency (p99): ${result.latency.p99.toFixed(2)}ms`);
    console.log(`   Throughput: ${(result.throughput.average / 1024 / 1024).toFixed(2)} MB/sec`);
    console.log(`   Total requests: ${result.requests.total}`);
    console.log(`   Total errors: ${result.errors}`);
    console.log(`   Error rate: ${((result.errors / result.requests.total) * 100).toFixed(2)}%`);
    
    // Performance thresholds
    const thresholds = {
      'Dashboard Statistics': { maxLatency: 500, minRps: 100 },
      'Repository List': { maxLatency: 800, minRps: 80 },
      'Scan Status': { maxLatency: 300, minRps: 150 },
      'Finding Details': { maxLatency: 600, minRps: 90 },
      'Health Check': { maxLatency: 100, minRps: 500 }
    };
    
    const threshold = thresholds[name];
    if (threshold) {
      const passed = result.latency.average <= threshold.maxLatency && 
                    result.requests.average >= threshold.minRps;
      
      console.log(`   Threshold: ${passed ? '✅ PASSED' : '❌ FAILED'}`);
      if (!passed) {
        console.log(`   Expected: <${threshold.maxLatency}ms latency, >${threshold.minRps} RPS`);
      }
    }
  }

  async runAllBenchmarks() {
    const results = {};
    
    try {
      // 1. Health Check (baseline)
      results.healthCheck = await this.benchmarkEndpoint(
        'Health Check',
        `${this.baseUrl}/health`,
        { connections: 50, duration: 10 }
      );

      // 2. Dashboard Statistics
      results.dashboardStats = await this.benchmarkEndpoint(
        'Dashboard Statistics',
        `${this.apiBase}/dashboard/statistics`
      );

      // 3. Repository List
      results.repositoryList = await this.benchmarkEndpoint(
        'Repository List',
        `${this.apiBase}/repositories`
      );

      // 4. Scan Status (simulated)
      results.scanStatus = await this.benchmarkEndpoint(
        'Scan Status',
        `${this.apiBase}/scans?limit=20`
      );

      // 5. Finding Details
      results.findingDetails = await this.benchmarkEndpoint(
        'Finding Details',
        `${this.apiBase}/findings?limit=50`
      );

      // 6. Authentication endpoint
      results.authentication = await this.benchmarkEndpoint(
        'Authentication',
        `${this.apiBase}/auth/login`,
        {
          method: 'POST',
          body: JSON.stringify({
            username: 'admin',
            password: 'test-password-123'
          }),
          headers: { 'Content-Type': 'application/json' },
          connections: 5, // Lower connections for auth
          duration: 15
        }
      );

      // 7. Heavy operation (export simulation)
      results.heavyOperation = await this.benchmarkEndpoint(
        'Heavy Operation',
        `${this.apiBase}/dashboard/statistics?detailed=true`,
        { connections: 5, duration: 20 }
      );

      return results;
      
    } catch (error) {
      console.error('❌ Benchmark failed:', error.message);
      throw error;
    }
  }

  async benchmarkConcurrentScans() {
    console.log('\n🔄 Benchmarking concurrent scan submissions...');
    
    // First, create a test repository
    const repoPayload = {
      name: `benchmark-repo-${uuidv4().substring(0, 8)}`,
      url: 'https://github.com/test/benchmark-repo',
      language: 'javascript',
      branch: 'main'
    };
    
    let repositoryId;
    try {
      const repoResponse = await axios.post(`${this.apiBase}/repositories`, repoPayload, {
        headers: this.getHeaders()
      });
      repositoryId = repoResponse.data.id;
      console.log(`✅ Created test repository: ${repositoryId}`);
    } catch (error) {
      console.error('❌ Failed to create test repository:', error.message);
      return;
    }

    // Benchmark scan submissions
    const scanPayload = {
      repository_id: repositoryId,
      scan_type: 'incremental',
      agents: ['semgrep']
    };

    const result = await this.benchmarkEndpoint(
      'Scan Submission',
      `${this.apiBase}/scans`,
      {
        method: 'POST',
        body: JSON.stringify(scanPayload),
        connections: 3, // Limited concurrent scans
        duration: 20
      }
    );

    // Cleanup
    try {
      await axios.delete(`${this.apiBase}/repositories/${repositoryId}`, {
        headers: this.getHeaders()
      });
      console.log('✅ Cleaned up test repository');
    } catch (error) {
      console.warn('⚠️  Failed to cleanup test repository:', error.message);
    }

    return result;
  }

  async benchmarkMemoryUsage() {
    console.log('\n💾 Benchmarking memory-intensive operations...');
    
    const memoryBenchmarks = [
      {
        name: 'Large Pagination',
        url: `${this.apiBase}/findings?limit=1000`,
        connections: 2,
        duration: 15
      },
      {
        name: 'Complex Filtering',
        url: `${this.apiBase}/findings?severity=high,critical&status=open&limit=500`,
        connections: 3,
        duration: 20
      },
      {
        name: 'Aggregation Query',
        url: `${this.apiBase}/dashboard/statistics?include_trends=true&days=90`,
        connections: 2,
        duration: 25
      }
    ];

    const results = {};
    for (const benchmark of memoryBenchmarks) {
      results[benchmark.name] = await this.benchmarkEndpoint(
        benchmark.name,
        benchmark.url,
        {
          connections: benchmark.connections,
          duration: benchmark.duration
        }
      );
    }

    return results;
  }

  generateReport(results) {
    console.log('\n📋 BENCHMARK REPORT');
    console.log('='.repeat(50));
    
    const summary = {
      totalRequests: 0,
      totalErrors: 0,
      avgLatency: 0,
      avgThroughput: 0
    };

    let benchmarkCount = 0;
    
    for (const [name, result] of Object.entries(results)) {
      if (result && result.requests) {
        summary.totalRequests += result.requests.total;
        summary.totalErrors += result.errors;
        summary.avgLatency += result.latency.average;
        summary.avgThroughput += result.throughput.average;
        benchmarkCount++;
      }
    }

    if (benchmarkCount > 0) {
      summary.avgLatency /= benchmarkCount;
      summary.avgThroughput /= benchmarkCount;
    }

    console.log(`Total Requests: ${summary.totalRequests}`);
    console.log(`Total Errors: ${summary.totalErrors}`);
    console.log(`Overall Error Rate: ${((summary.totalErrors / summary.totalRequests) * 100).toFixed(2)}%`);
    console.log(`Average Latency: ${summary.avgLatency.toFixed(2)}ms`);
    console.log(`Average Throughput: ${(summary.avgThroughput / 1024 / 1024).toFixed(2)} MB/sec`);
    
    // Performance grade
    const errorRate = (summary.totalErrors / summary.totalRequests) * 100;
    let grade = 'A';
    
    if (errorRate > 1 || summary.avgLatency > 1000) {
      grade = 'B';
    }
    if (errorRate > 5 || summary.avgLatency > 2000) {
      grade = 'C';
    }
    if (errorRate > 10 || summary.avgLatency > 5000) {
      grade = 'D';
    }
    
    console.log(`\n🎯 Performance Grade: ${grade}`);
    
    return summary;
  }
}

// Main execution
async function main() {
  const benchmark = new ApiBenchmark();
  
  try {
    await benchmark.setup();
    
    console.log('\n🏁 Starting API benchmarks...');
    const results = await benchmark.runAllBenchmarks();
    
    console.log('\n🔄 Running concurrent scan benchmarks...');
    results.concurrentScans = await benchmark.benchmarkConcurrentScans();
    
    console.log('\n💾 Running memory usage benchmarks...');
    const memoryResults = await benchmark.benchmarkMemoryUsage();
    Object.assign(results, memoryResults);
    
    // Generate final report
    benchmark.generateReport(results);
    
    console.log('\n✅ All benchmarks completed successfully!');
    
  } catch (error) {
    console.error('❌ Benchmark suite failed:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = ApiBenchmark;