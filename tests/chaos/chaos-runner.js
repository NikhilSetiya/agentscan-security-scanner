const axios = require('axios');
const { v4: uuidv4 } = require('uuid');

class ChaosRunner {
  constructor(config = {}) {
    this.baseUrl = config.baseUrl || 'http://localhost:8080';
    this.apiBase = `${this.baseUrl}/api/v1`;
    this.experiments = [];
    this.results = {
      experiments: [],
      summary: {},
      startTime: new Date(),
      endTime: null
    };
    this.authToken = null;
  }

  async setup() {
    console.log('🔧 Setting up chaos engineering environment...');
    
    // Authenticate
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
    
    // Verify system is healthy before starting chaos
    await this.verifySystemHealth();
  }

  async verifySystemHealth() {
    console.log('🏥 Verifying system health...');
    
    const healthChecks = [
      { name: 'API Health', url: `${this.baseUrl}/health` },
      { name: 'Database Connection', url: `${this.apiBase}/repositories` },
      { name: 'Authentication', url: `${this.apiBase}/users/profile` }
    ];
    
    for (const check of healthChecks) {
      try {
        const headers = check.name === 'Authentication' ? 
          { 'Authorization': `Bearer ${this.authToken}` } : {};
        
        const response = await axios.get(check.url, { 
          headers,
          timeout: 10000 
        });
        
        if (response.status === 200) {
          console.log(`✅ ${check.name}: Healthy`);
        } else {
          throw new Error(`Unexpected status: ${response.status}`);
        }
      } catch (error) {
        console.error(`❌ ${check.name}: Unhealthy - ${error.message}`);
        throw new Error(`System not healthy: ${check.name} failed`);
      }
    }
    
    console.log('✅ System health verification completed');
  }

  async runExperiment(experiment) {
    console.log(`\n🧪 Running experiment: ${experiment.name}`);
    
    const experimentResult = {
      id: uuidv4(),
      name: experiment.name,
      description: experiment.description,
      startTime: new Date(),
      endTime: null,
      status: 'running',
      hypothesis: experiment.hypothesis,
      steps: [],
      metrics: {},
      success: false,
      error: null
    };
    
    try {
      // Record baseline metrics
      const baselineMetrics = await this.collectMetrics();
      experimentResult.metrics.baseline = baselineMetrics;
      
      // Execute experiment steps
      for (const step of experiment.steps) {
        console.log(`  📋 Executing step: ${step.name}`);
        
        const stepResult = {
          name: step.name,
          startTime: new Date(),
          endTime: null,
          success: false,
          error: null,
          metrics: {}
        };
        
        try {
          await step.execute();
          stepResult.success = true;
          console.log(`  ✅ Step completed: ${step.name}`);
        } catch (error) {
          stepResult.error = error.message;
          console.log(`  ❌ Step failed: ${step.name} - ${error.message}`);
        }
        
        stepResult.endTime = new Date();
        stepResult.metrics = await this.collectMetrics();
        experimentResult.steps.push(stepResult);
        
        // Wait between steps if specified
        if (step.waitAfter) {
          console.log(`  ⏳ Waiting ${step.waitAfter}ms...`);
          await this.sleep(step.waitAfter);
        }
      }
      
      // Collect final metrics
      experimentResult.metrics.final = await this.collectMetrics();
      
      // Verify hypothesis
      experimentResult.success = await experiment.verifyHypothesis(experimentResult);
      experimentResult.status = experimentResult.success ? 'passed' : 'failed';
      
      console.log(`  🎯 Hypothesis verification: ${experimentResult.success ? 'PASSED' : 'FAILED'}`);
      
    } catch (error) {
      experimentResult.error = error.message;
      experimentResult.status = 'error';
      console.log(`  💥 Experiment error: ${error.message}`);
    }
    
    experimentResult.endTime = new Date();
    this.results.experiments.push(experimentResult);
    
    // Cleanup after experiment
    if (experiment.cleanup) {
      try {
        console.log(`  🧹 Running cleanup...`);
        await experiment.cleanup();
      } catch (error) {
        console.log(`  ⚠️  Cleanup failed: ${error.message}`);
      }
    }
    
    return experimentResult;
  }

  async collectMetrics() {
    const metrics = {
      timestamp: new Date(),
      api: {},
      system: {}
    };
    
    try {
      // API response time
      const apiStart = Date.now();
      const apiResponse = await axios.get(`${this.apiBase}/health`, {
        timeout: 5000
      });
      metrics.api.responseTime = Date.now() - apiStart;
      metrics.api.status = apiResponse.status;
      
      // System metrics (would integrate with monitoring system)
      metrics.system.cpu = Math.random() * 100; // Mock data
      metrics.system.memory = Math.random() * 100;
      metrics.system.disk = Math.random() * 100;
      
    } catch (error) {
      metrics.api.error = error.message;
      metrics.api.responseTime = null;
    }
    
    return metrics;
  }

  sleep(ms) {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  async generateReport() {
    this.results.endTime = new Date();
    const duration = this.results.endTime - this.results.startTime;
    
    console.log('\n' + '='.repeat(60));
    console.log('🧪 CHAOS ENGINEERING REPORT');
    console.log('='.repeat(60));
    
    console.log(`\n📊 SUMMARY:`);
    console.log(`   Total Experiments: ${this.results.experiments.length}`);
    console.log(`   Passed: ${this.results.experiments.filter(e => e.success).length}`);
    console.log(`   Failed: ${this.results.experiments.filter(e => !e.success && e.status !== 'error').length}`);
    console.log(`   Errors: ${this.results.experiments.filter(e => e.status === 'error').length}`);
    console.log(`   Duration: ${Math.round(duration / 1000)}s`);
    
    // Calculate resilience score
    const totalExperiments = this.results.experiments.length;
    const passedExperiments = this.results.experiments.filter(e => e.success).length;
    const resilienceScore = totalExperiments > 0 ? (passedExperiments / totalExperiments) * 100 : 0;
    
    console.log(`\n🎯 RESILIENCE SCORE: ${resilienceScore.toFixed(1)}%`);
    
    let resilienceGrade = 'F';
    if (resilienceScore >= 90) resilienceGrade = 'A';
    else if (resilienceScore >= 80) resilienceGrade = 'B';
    else if (resilienceScore >= 70) resilienceGrade = 'C';
    else if (resilienceScore >= 60) resilienceGrade = 'D';
    
    console.log(`   Grade: ${resilienceGrade}`);
    
    // Experiment details
    console.log(`\n🧪 EXPERIMENT RESULTS:`);
    this.results.experiments.forEach((exp, index) => {
      const duration = exp.endTime - exp.startTime;
      const status = exp.success ? '✅ PASSED' : 
                    exp.status === 'error' ? '💥 ERROR' : '❌ FAILED';
      
      console.log(`\n${index + 1}. ${exp.name} - ${status}`);
      console.log(`   Duration: ${Math.round(duration / 1000)}s`);
      console.log(`   Hypothesis: ${exp.hypothesis}`);
      
      if (exp.error) {
        console.log(`   Error: ${exp.error}`);
      }
      
      // Show step results
      exp.steps.forEach((step, stepIndex) => {
        const stepStatus = step.success ? '✅' : '❌';
        console.log(`     ${stepIndex + 1}. ${stepStatus} ${step.name}`);
        if (step.error) {
          console.log(`        Error: ${step.error}`);
        }
      });
    });
    
    // Recommendations
    console.log(`\n💡 RECOMMENDATIONS:`);
    const failedExperiments = this.results.experiments.filter(e => !e.success);
    
    if (failedExperiments.length === 0) {
      console.log('   🎉 Excellent! All chaos experiments passed.');
      console.log('   Consider increasing experiment complexity or adding new scenarios.');
    } else {
      console.log('   Based on failed experiments, consider:');
      failedExperiments.forEach((exp, index) => {
        console.log(`   ${index + 1}. Improve resilience for: ${exp.name}`);
      });
    }
    
    this.results.summary = {
      totalExperiments,
      passedExperiments,
      failedExperiments: failedExperiments.length,
      errorExperiments: this.results.experiments.filter(e => e.status === 'error').length,
      resilienceScore,
      resilienceGrade,
      duration: Math.round(duration / 1000)
    };
    
    return this.results;
  }
}

// Example chaos experiments
const networkLatencyExperiment = {
  name: 'Network Latency Injection',
  description: 'Inject network latency to test system resilience',
  hypothesis: 'System should handle increased network latency gracefully',
  
  steps: [
    {
      name: 'Inject 500ms latency',
      execute: async () => {
        // In a real implementation, this would use tools like tc (traffic control)
        // or Kubernetes network policies to inject latency
        console.log('    Simulating network latency injection...');
        await new Promise(resolve => setTimeout(resolve, 1000));
      },
      waitAfter: 5000
    },
    {
      name: 'Test API responsiveness',
      execute: async () => {
        const start = Date.now();
        const response = await axios.get('http://localhost:8080/api/v1/health', {
          timeout: 10000
        });
        const responseTime = Date.now() - start;
        
        if (responseTime > 2000) {
          throw new Error(`Response time too high: ${responseTime}ms`);
        }
      }
    }
  ],
  
  verifyHypothesis: async (result) => {
    // Check if API remained responsive during latency injection
    const finalMetrics = result.metrics.final;
    return finalMetrics.api.responseTime < 2000;
  },
  
  cleanup: async () => {
    // Remove network latency injection
    console.log('    Removing network latency...');
  }
};

const databaseConnectionExperiment = {
  name: 'Database Connection Pool Exhaustion',
  description: 'Exhaust database connections to test connection handling',
  hypothesis: 'System should handle database connection exhaustion gracefully',
  
  steps: [
    {
      name: 'Create multiple concurrent requests',
      execute: async () => {
        const requests = [];
        for (let i = 0; i < 50; i++) {
          requests.push(
            axios.get('http://localhost:8080/api/v1/repositories', {
              headers: { 'Authorization': 'Bearer test-token' },
              timeout: 5000
            }).catch(err => ({ error: err.message }))
          );
        }
        
        const responses = await Promise.all(requests);
        const errors = responses.filter(r => r.error).length;
        
        if (errors > responses.length * 0.5) {
          throw new Error(`Too many errors: ${errors}/${responses.length}`);
        }
      }
    }
  ],
  
  verifyHypothesis: async (result) => {
    // System should not crash and should recover
    const finalMetrics = result.metrics.final;
    return finalMetrics.api.status === 200;
  }
};

const memoryPressureExperiment = {
  name: 'Memory Pressure Test',
  description: 'Apply memory pressure to test system stability',
  hypothesis: 'System should handle memory pressure without crashing',
  
  steps: [
    {
      name: 'Generate memory pressure',
      execute: async () => {
        // Simulate memory pressure by making large requests
        const largeRequests = [];
        for (let i = 0; i < 10; i++) {
          largeRequests.push(
            axios.get('http://localhost:8080/api/v1/findings?limit=1000', {
              timeout: 10000
            }).catch(err => ({ error: err.message }))
          );
        }
        
        await Promise.all(largeRequests);
      },
      waitAfter: 3000
    }
  ],
  
  verifyHypothesis: async (result) => {
    const finalMetrics = result.metrics.final;
    return finalMetrics.api.responseTime < 5000;
  }
};

// Main execution
async function main() {
  const runner = new ChaosRunner();
  
  try {
    await runner.setup();
    
    const experiments = [
      networkLatencyExperiment,
      databaseConnectionExperiment,
      memoryPressureExperiment
    ];
    
    console.log(`\n🚀 Starting ${experiments.length} chaos experiments...\n`);
    
    for (const experiment of experiments) {
      await runner.runExperiment(experiment);
      
      // Wait between experiments
      await runner.sleep(2000);
    }
    
    const results = await runner.generateReport();
    
    // Save results
    const fs = require('fs');
    fs.writeFileSync(
      'chaos-test-results.json',
      JSON.stringify(results, null, 2)
    );
    
    console.log('\n📄 Results saved to chaos-test-results.json');
    
    // Exit with error if resilience score is too low
    if (results.summary.resilienceScore < 70) {
      console.log('\n⚠️  Resilience score below threshold (70%)');
      process.exit(1);
    }
    
  } catch (error) {
    console.error('❌ Chaos engineering failed:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = ChaosRunner;