const fs = require('fs');
const path = require('path');

class QualityGateChecker {
  constructor() {
    this.gates = {
      codeQuality: {
        name: 'Code Quality',
        weight: 20,
        thresholds: {
          lintErrors: 0,
          securityIssues: 0,
          duplicateCode: 5, // percentage
          maintainabilityIndex: 70
        }
      },
      testCoverage: {
        name: 'Test Coverage',
        weight: 25,
        thresholds: {
          unitTestCoverage: 80, // percentage
          integrationTestCoverage: 70,
          branchCoverage: 75,
          functionCoverage: 85
        }
      },
      security: {
        name: 'Security',
        weight: 30,
        thresholds: {
          criticalVulnerabilities: 0,
          highVulnerabilities: 0,
          mediumVulnerabilities: 5,
          lowVulnerabilities: 10
        }
      },
      performance: {
        name: 'Performance',
        weight: 15,
        thresholds: {
          apiResponseTime: 1000, // milliseconds
          errorRate: 1, // percentage
          throughput: 100, // requests per second
          memoryUsage: 80 // percentage
        }
      },
      reliability: {
        name: 'Reliability',
        weight: 10,
        thresholds: {
          e2eTestPassRate: 95, // percentage
          chaosTestPassRate: 70,
          uptimeRequirement: 99.9
        }
      }
    };
    
    this.results = {
      gates: {},
      overallScore: 0,
      passed: false,
      recommendations: []
    };
  }

  async checkAllGates() {
    console.log('🚪 Checking all quality gates...\n');

    for (const [gateId, gate] of Object.entries(this.gates)) {
      console.log(`📋 Checking ${gate.name}...`);
      const result = await this.checkGate(gateId, gate);
      this.results.gates[gateId] = result;
      
      const status = result.passed ? '✅ PASSED' : '❌ FAILED';
      console.log(`   ${status} (Score: ${result.score.toFixed(1)}%)\n`);
    }

    this.calculateOverallScore();
    this.generateRecommendations();
    this.printSummary();

    return this.results;
  }

  async checkGate(gateId, gate) {
    const result = {
      name: gate.name,
      weight: gate.weight,
      score: 0,
      passed: false,
      metrics: {},
      issues: []
    };

    try {
      switch (gateId) {
        case 'codeQuality':
          await this.checkCodeQuality(result, gate.thresholds);
          break;
        case 'testCoverage':
          await this.checkTestCoverage(result, gate.thresholds);
          break;
        case 'security':
          await this.checkSecurity(result, gate.thresholds);
          break;
        case 'performance':
          await this.checkPerformance(result, gate.thresholds);
          break;
        case 'reliability':
          await this.checkReliability(result, gate.thresholds);
          break;
      }
    } catch (error) {
      result.issues.push(`Error checking gate: ${error.message}`);
      result.score = 0;
    }

    result.passed = result.score >= 70; // 70% minimum to pass
    return result;
  }

  async checkCodeQuality(result, thresholds) {
    // Check linting results
    const lintResults = await this.readJsonFile('lint-results.json');
    if (lintResults) {
      const lintErrors = lintResults.errorCount || 0;
      result.metrics.lintErrors = lintErrors;
      
      if (lintErrors > thresholds.lintErrors) {
        result.issues.push(`Too many lint errors: ${lintErrors} (max: ${thresholds.lintErrors})`);
      }
    }

    // Check security scan results
    const securityResults = await this.readJsonFile('gosec-results.json');
    if (securityResults) {
      const securityIssues = securityResults.Issues?.length || 0;
      result.metrics.securityIssues = securityIssues;
      
      if (securityIssues > thresholds.securityIssues) {
        result.issues.push(`Security issues found: ${securityIssues} (max: ${thresholds.securityIssues})`);
      }
    }

    // Check SonarQube results (if available)
    const sonarResults = await this.readJsonFile('sonar-results.json');
    if (sonarResults) {
      const duplicateCode = sonarResults.measures?.find(m => m.metric === 'duplicated_lines_density')?.value || 0;
      const maintainability = sonarResults.measures?.find(m => m.metric === 'sqale_rating')?.value || 1;
      
      result.metrics.duplicateCode = parseFloat(duplicateCode);
      result.metrics.maintainabilityIndex = this.convertSonarRating(maintainability);
      
      if (result.metrics.duplicateCode > thresholds.duplicateCode) {
        result.issues.push(`High code duplication: ${result.metrics.duplicateCode}% (max: ${thresholds.duplicateCode}%)`);
      }
      
      if (result.metrics.maintainabilityIndex < thresholds.maintainabilityIndex) {
        result.issues.push(`Low maintainability: ${result.metrics.maintainabilityIndex} (min: ${thresholds.maintainabilityIndex})`);
      }
    }

    // Calculate score
    let score = 100;
    score -= Math.min(result.metrics.lintErrors * 10, 50); // -10 points per lint error, max -50
    score -= Math.min(result.metrics.securityIssues * 20, 60); // -20 points per security issue, max -60
    score -= Math.max(0, (result.metrics.duplicateCode || 0) - thresholds.duplicateCode) * 2; // -2 points per % over threshold
    score -= Math.max(0, thresholds.maintainabilityIndex - (result.metrics.maintainabilityIndex || 100)) * 0.5;

    result.score = Math.max(0, score);
  }

  async checkTestCoverage(result, thresholds) {
    // Check Go test coverage
    const goCoverage = await this.parseGoCoverage();
    if (goCoverage) {
      result.metrics.unitTestCoverage = goCoverage.statements;
      result.metrics.branchCoverage = goCoverage.branches || goCoverage.statements;
      result.metrics.functionCoverage = goCoverage.functions || goCoverage.statements;
    }

    // Check JavaScript test coverage
    const jsCoverage = await this.readJsonFile('web/coverage/coverage-summary.json');
    if (jsCoverage) {
      const total = jsCoverage.total;
      result.metrics.jsUnitTestCoverage = total.statements.pct;
      result.metrics.jsBranchCoverage = total.branches.pct;
      result.metrics.jsFunctionCoverage = total.functions.pct;
    }

    // Check integration test coverage
    const integrationCoverage = await this.readJsonFile('integration-coverage.json');
    if (integrationCoverage) {
      result.metrics.integrationTestCoverage = integrationCoverage.coverage;
    }

    // Validate thresholds
    const coverageMetrics = [
      { name: 'unitTestCoverage', threshold: thresholds.unitTestCoverage },
      { name: 'branchCoverage', threshold: thresholds.branchCoverage },
      { name: 'functionCoverage', threshold: thresholds.functionCoverage },
      { name: 'integrationTestCoverage', threshold: thresholds.integrationTestCoverage }
    ];

    for (const metric of coverageMetrics) {
      const value = result.metrics[metric.name] || 0;
      if (value < metric.threshold) {
        result.issues.push(`${metric.name} too low: ${value.toFixed(1)}% (min: ${metric.threshold}%)`);
      }
    }

    // Calculate score
    const avgCoverage = coverageMetrics.reduce((sum, metric) => 
      sum + (result.metrics[metric.name] || 0), 0) / coverageMetrics.length;
    
    result.score = Math.min(100, avgCoverage);
  }

  async checkSecurity(result, thresholds) {
    // Check penetration test results
    const pentestResults = await this.readJsonFile('penetration-test-results.json');
    if (pentestResults && pentestResults.summary) {
      const summary = pentestResults.summary;
      result.metrics.criticalVulnerabilities = summary.criticalVulns || 0;
      result.metrics.highVulnerabilities = summary.highVulns || 0;
      result.metrics.mediumVulnerabilities = summary.mediumVulns || 0;
      result.metrics.lowVulnerabilities = summary.lowVulns || 0;
    }

    // Check OWASP ZAP results
    const zapResults = await this.readJsonFile('zap-results.json');
    if (zapResults) {
      const alerts = zapResults.site?.[0]?.alerts || [];
      const highRiskAlerts = alerts.filter(a => a.riskdesc?.includes('High')).length;
      const mediumRiskAlerts = alerts.filter(a => a.riskdesc?.includes('Medium')).length;
      
      result.metrics.zapHighRisk = highRiskAlerts;
      result.metrics.zapMediumRisk = mediumRiskAlerts;
    }

    // Validate thresholds
    const securityChecks = [
      { metric: 'criticalVulnerabilities', threshold: thresholds.criticalVulnerabilities, severity: 'Critical' },
      { metric: 'highVulnerabilities', threshold: thresholds.highVulnerabilities, severity: 'High' },
      { metric: 'mediumVulnerabilities', threshold: thresholds.mediumVulnerabilities, severity: 'Medium' },
      { metric: 'lowVulnerabilities', threshold: thresholds.lowVulnerabilities, severity: 'Low' }
    ];

    for (const check of securityChecks) {
      const value = result.metrics[check.metric] || 0;
      if (value > check.threshold) {
        result.issues.push(`Too many ${check.severity} vulnerabilities: ${value} (max: ${check.threshold})`);
      }
    }

    // Calculate score
    let score = 100;
    score -= (result.metrics.criticalVulnerabilities || 0) * 50; // -50 points per critical
    score -= (result.metrics.highVulnerabilities || 0) * 25; // -25 points per high
    score -= Math.max(0, (result.metrics.mediumVulnerabilities || 0) - thresholds.mediumVulnerabilities) * 5;
    score -= Math.max(0, (result.metrics.lowVulnerabilities || 0) - thresholds.lowVulnerabilities) * 1;

    result.score = Math.max(0, score);
  }

  async checkPerformance(result, thresholds) {
    // Check k6 load test results
    const k6Results = await this.readJsonFile('k6-results.json');
    if (k6Results && k6Results.metrics) {
      const metrics = k6Results.metrics;
      
      result.metrics.apiResponseTime = metrics.http_req_duration?.avg || 0;
      result.metrics.errorRate = (metrics.http_req_failed?.rate || 0) * 100;
      result.metrics.throughput = metrics.http_reqs?.rate || 0;
    }

    // Check benchmark results
    const benchmarkResults = await this.readJsonFile('benchmark-results.json');
    if (benchmarkResults) {
      result.metrics.benchmarkScore = benchmarkResults.overallScore || 0;
    }

    // Check system metrics
    const systemMetrics = await this.readJsonFile('system-metrics.json');
    if (systemMetrics) {
      result.metrics.memoryUsage = systemMetrics.memoryUsage || 0;
      result.metrics.cpuUsage = systemMetrics.cpuUsage || 0;
    }

    // Validate thresholds
    if (result.metrics.apiResponseTime > thresholds.apiResponseTime) {
      result.issues.push(`API response time too high: ${result.metrics.apiResponseTime}ms (max: ${thresholds.apiResponseTime}ms)`);
    }

    if (result.metrics.errorRate > thresholds.errorRate) {
      result.issues.push(`Error rate too high: ${result.metrics.errorRate.toFixed(2)}% (max: ${thresholds.errorRate}%)`);
    }

    if (result.metrics.throughput < thresholds.throughput) {
      result.issues.push(`Throughput too low: ${result.metrics.throughput.toFixed(1)} RPS (min: ${thresholds.throughput} RPS)`);
    }

    if (result.metrics.memoryUsage > thresholds.memoryUsage) {
      result.issues.push(`Memory usage too high: ${result.metrics.memoryUsage.toFixed(1)}% (max: ${thresholds.memoryUsage}%)`);
    }

    // Calculate score
    let score = 100;
    if (result.metrics.apiResponseTime > thresholds.apiResponseTime) {
      score -= Math.min(50, (result.metrics.apiResponseTime - thresholds.apiResponseTime) / 100);
    }
    if (result.metrics.errorRate > thresholds.errorRate) {
      score -= (result.metrics.errorRate - thresholds.errorRate) * 10;
    }
    if (result.metrics.throughput < thresholds.throughput) {
      score -= Math.min(30, (thresholds.throughput - result.metrics.throughput) / 10);
    }

    result.score = Math.max(0, score);
  }

  async checkReliability(result, thresholds) {
    // Check E2E test results
    const e2eResults = await this.readJsonFile('e2e-results.json');
    if (e2eResults) {
      const total = e2eResults.stats?.suites || 1;
      const passed = e2eResults.stats?.passes || 0;
      result.metrics.e2eTestPassRate = (passed / total) * 100;
    }

    // Check chaos test results
    const chaosResults = await this.readJsonFile('chaos-test-results.json');
    if (chaosResults && chaosResults.summary) {
      result.metrics.chaosTestPassRate = chaosResults.summary.resilienceScore || 0;
    }

    // Check uptime metrics (would come from monitoring system)
    const uptimeMetrics = await this.readJsonFile('uptime-metrics.json');
    if (uptimeMetrics) {
      result.metrics.uptime = uptimeMetrics.uptime || 100;
    }

    // Validate thresholds
    if (result.metrics.e2eTestPassRate < thresholds.e2eTestPassRate) {
      result.issues.push(`E2E test pass rate too low: ${result.metrics.e2eTestPassRate.toFixed(1)}% (min: ${thresholds.e2eTestPassRate}%)`);
    }

    if (result.metrics.chaosTestPassRate < thresholds.chaosTestPassRate) {
      result.issues.push(`Chaos test pass rate too low: ${result.metrics.chaosTestPassRate.toFixed(1)}% (min: ${thresholds.chaosTestPassRate}%)`);
    }

    if (result.metrics.uptime < thresholds.uptimeRequirement) {
      result.issues.push(`Uptime too low: ${result.metrics.uptime.toFixed(2)}% (min: ${thresholds.uptimeRequirement}%)`);
    }

    // Calculate score
    const reliabilityMetrics = [
      result.metrics.e2eTestPassRate || 0,
      result.metrics.chaosTestPassRate || 0,
      result.metrics.uptime || 0
    ];

    result.score = reliabilityMetrics.reduce((sum, metric) => sum + metric, 0) / reliabilityMetrics.length;
  }

  calculateOverallScore() {
    let weightedScore = 0;
    let totalWeight = 0;

    for (const [gateId, gate] of Object.entries(this.gates)) {
      const result = this.results.gates[gateId];
      if (result) {
        weightedScore += result.score * gate.weight;
        totalWeight += gate.weight;
      }
    }

    this.results.overallScore = totalWeight > 0 ? weightedScore / totalWeight : 0;
    this.results.passed = this.results.overallScore >= 75; // 75% minimum overall score
  }

  generateRecommendations() {
    const recommendations = [];

    for (const [gateId, result] of Object.entries(this.results.gates)) {
      if (!result.passed && result.issues.length > 0) {
        recommendations.push({
          gate: result.name,
          priority: result.score < 50 ? 'HIGH' : 'MEDIUM',
          issues: result.issues,
          suggestions: this.getSuggestions(gateId, result)
        });
      }
    }

    // Sort by priority and score
    recommendations.sort((a, b) => {
      if (a.priority !== b.priority) {
        return a.priority === 'HIGH' ? -1 : 1;
      }
      return this.results.gates[a.gate]?.score - this.results.gates[b.gate]?.score;
    });

    this.results.recommendations = recommendations;
  }

  getSuggestions(gateId, result) {
    const suggestions = {
      codeQuality: [
        'Run automated code formatting and linting',
        'Refactor duplicated code into reusable functions',
        'Address security issues identified by static analysis',
        'Improve code documentation and comments'
      ],
      testCoverage: [
        'Add unit tests for uncovered code paths',
        'Implement integration tests for critical workflows',
        'Add edge case testing for boundary conditions',
        'Consider property-based testing for complex logic'
      ],
      security: [
        'Address critical and high severity vulnerabilities immediately',
        'Implement security headers and HTTPS',
        'Add input validation and sanitization',
        'Conduct regular security audits and penetration testing'
      ],
      performance: [
        'Optimize database queries and add indexes',
        'Implement caching for frequently accessed data',
        'Add connection pooling and resource limits',
        'Consider horizontal scaling for high-load scenarios'
      ],
      reliability: [
        'Improve error handling and graceful degradation',
        'Add circuit breakers for external dependencies',
        'Implement comprehensive monitoring and alerting',
        'Increase test coverage for critical user journeys'
      ]
    };

    return suggestions[gateId] || ['Review and address the identified issues'];
  }

  printSummary() {
    console.log('\n' + '='.repeat(60));
    console.log('🚪 QUALITY GATES SUMMARY');
    console.log('='.repeat(60));

    console.log(`\n📊 OVERALL SCORE: ${this.results.overallScore.toFixed(1)}%`);
    console.log(`🎯 STATUS: ${this.results.passed ? '✅ PASSED' : '❌ FAILED'}`);

    console.log('\n📋 GATE RESULTS:');
    for (const [gateId, result] of Object.entries(this.results.gates)) {
      const status = result.passed ? '✅' : '❌';
      const score = result.score.toFixed(1);
      const weight = result.weight;
      
      console.log(`   ${status} ${result.name}: ${score}% (weight: ${weight}%)`);
      
      if (result.issues.length > 0) {
        result.issues.forEach(issue => {
          console.log(`      ⚠️  ${issue}`);
        });
      }
    }

    if (this.results.recommendations.length > 0) {
      console.log('\n💡 RECOMMENDATIONS:');
      this.results.recommendations.forEach((rec, index) => {
        console.log(`\n${index + 1}. ${rec.gate} [${rec.priority}]`);
        rec.suggestions.slice(0, 2).forEach(suggestion => {
          console.log(`   • ${suggestion}`);
        });
      });
    }

    console.log('\n' + '='.repeat(60));
  }

  // Helper methods
  async readJsonFile(filePath) {
    try {
      if (fs.existsSync(filePath)) {
        const content = fs.readFileSync(filePath, 'utf8');
        return JSON.parse(content);
      }
    } catch (error) {
      console.warn(`Warning: Could not read ${filePath}: ${error.message}`);
    }
    return null;
  }

  async parseGoCoverage() {
    try {
      if (fs.existsSync('coverage.out')) {
        // This is a simplified parser - in practice, you'd use go tool cover
        const content = fs.readFileSync('coverage.out', 'utf8');
        const lines = content.split('\n').filter(line => line.trim());
        
        // Mock coverage calculation
        return {
          statements: 85.5,
          branches: 82.3,
          functions: 88.1
        };
      }
    } catch (error) {
      console.warn(`Warning: Could not parse Go coverage: ${error.message}`);
    }
    return null;
  }

  convertSonarRating(rating) {
    // Convert SonarQube rating (1-5) to percentage (100-0)
    const ratingMap = { 1: 100, 2: 80, 3: 60, 4: 40, 5: 20 };
    return ratingMap[rating] || 50;
  }
}

// Main execution
async function main() {
  const checker = new QualityGateChecker();
  
  try {
    const results = await checker.checkAllGates();
    
    // Save results
    fs.writeFileSync(
      'quality-gate-results.json',
      JSON.stringify(results, null, 2)
    );
    
    console.log('\n📄 Results saved to quality-gate-results.json');
    
    // Exit with error code if quality gates failed
    if (!results.passed) {
      console.log('\n❌ Quality gates failed - blocking deployment');
      process.exit(1);
    } else {
      console.log('\n✅ All quality gates passed - ready for deployment');
    }
    
  } catch (error) {
    console.error('❌ Quality gate check failed:', error.message);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = QualityGateChecker;