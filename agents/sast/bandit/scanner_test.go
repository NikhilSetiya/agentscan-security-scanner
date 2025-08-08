package bandit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/agentscan/agentscan/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBanditOutput_EmptyResults(t *testing.T) {
	a := NewAgent()
	
	emptyOutput := `{"results": [], "metrics": {}}`
	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseBanditOutput([]byte(emptyOutput), config)
	
	assert.NoError(t, err)
	assert.Empty(t, findings)
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 0, metadata.FilesScanned)
}

func TestParseBanditOutput_InvalidJSON(t *testing.T) {
	a := NewAgent()
	
	invalidOutput := `invalid json`
	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseBanditOutput([]byte(invalidOutput), config)
	
	assert.Error(t, err)
	assert.Nil(t, findings)
	assert.Equal(t, agent.Metadata{}, metadata)
}

func TestParseBanditOutput_WithFindings(t *testing.T) {
	a := NewAgent()
	
	banditOutput := `{
  "results": [
    {
      "code": "password = 'hardcoded_password'",
      "col_number": 11,
      "filename": "/src/app.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 259,
        "link": "https://cwe.mitre.org/data/definitions/259.html"
      },
      "issue_severity": "LOW",
      "issue_text": "Possible hardcoded password: 'hardcoded_password'",
      "line_number": 10,
      "line_range": [10, 10],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b105_hardcoded_password_string.html",
      "test_id": "B105",
      "test_name": "hardcoded_password_string"
    },
    {
      "code": "subprocess.call(user_input, shell=True)",
      "col_number": 1,
      "filename": "/src/utils.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 78,
        "link": "https://cwe.mitre.org/data/definitions/78.html"
      },
      "issue_severity": "HIGH",
      "issue_text": "subprocess call with shell=True identified, security issue.",
      "line_number": 25,
      "line_range": [25, 25],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b602_subprocess_popen_with_shell_equals_true.html",
      "test_id": "B602",
      "test_name": "subprocess_popen_with_shell_equals_true"
    },
    {
      "code": "eval(user_code)",
      "col_number": 5,
      "filename": "/src/dangerous.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 94,
        "link": "https://cwe.mitre.org/data/definitions/94.html"
      },
      "issue_severity": "MEDIUM",
      "issue_text": "Use of eval detected.",
      "line_number": 15,
      "line_range": [15, 15],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b307_eval.html",
      "test_id": "B307",
      "test_name": "eval"
    }
  ],
  "metrics": {
    "_totals": {
      "CONFIDENCE.HIGH": 3,
      "CONFIDENCE.LOW": 0,
      "CONFIDENCE.MEDIUM": 0,
      "CONFIDENCE.UNDEFINED": 0,
      "SEVERITY.HIGH": 1,
      "SEVERITY.LOW": 1,
      "SEVERITY.MEDIUM": 1,
      "SEVERITY.UNDEFINED": 0,
      "files_skipped": 2,
      "loc": 150,
      "nosec": 0,
      "skipped_tests": 0
    }
  }
}`

	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseBanditOutput([]byte(banditOutput), config)
	
	assert.NoError(t, err)
	
	// Should have 3 findings
	assert.Len(t, findings, 3)
	
	// Check first finding (hardcoded password)
	finding1 := findings[0]
	assert.Equal(t, AgentName, finding1.Tool)
	assert.Equal(t, "B105", finding1.RuleID)
	assert.Equal(t, agent.SeverityLow, finding1.Severity)
	assert.Equal(t, agent.CategoryHardcodedSecrets, finding1.Category)
	assert.Equal(t, "Hardcoded Password String", finding1.Title)
	assert.Equal(t, "Possible hardcoded password: 'hardcoded_password'", finding1.Description)
	assert.Equal(t, "app.py", finding1.File)
	assert.Equal(t, 10, finding1.Line)
	assert.Equal(t, 11, finding1.Column)
	assert.Equal(t, "password = 'hardcoded_password'", finding1.Code)
	assert.Equal(t, 0.9, finding1.Confidence)
	assert.NotEmpty(t, finding1.References)
	
	// Check second finding (command injection)
	finding2 := findings[1]
	assert.Equal(t, "B602", finding2.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding2.Severity)
	assert.Equal(t, agent.CategoryCommandInjection, finding2.Category)
	assert.Equal(t, "utils.py", finding2.File)
	assert.Equal(t, 25, finding2.Line)
	
	// Check third finding (eval)
	finding3 := findings[2]
	assert.Equal(t, "B307", finding3.RuleID)
	assert.Equal(t, agent.SeverityMedium, finding3.Severity)
	assert.Equal(t, agent.CategoryCommandInjection, finding3.Category)
	assert.Equal(t, "dangerous.py", finding3.File)
	assert.Equal(t, 15, finding3.Line)
	
	// Check metadata
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 3, metadata.FilesScanned) // 3 unique files
	assert.Equal(t, 150, metadata.LinesScanned)
	assert.Contains(t, metadata.Environment, "files_skipped")
	assert.Contains(t, metadata.Environment, "nosec_comments")
	assert.Equal(t, "2", metadata.Environment["files_skipped"])
	assert.Equal(t, "0", metadata.Environment["nosec_comments"])
}

func TestParseBanditOutput_RealWorldExample(t *testing.T) {
	a := NewAgent()
	
	// More realistic Bandit output with various vulnerability types
	banditOutput := `{
  "results": [
    {
      "code": "import pickle\ndata = pickle.loads(user_data)",
      "col_number": 8,
      "filename": "/src/serialization.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 502,
        "link": "https://cwe.mitre.org/data/definitions/502.html"
      },
      "issue_severity": "HIGH",
      "issue_text": "Pickle library appears to be in use, possible security issue.",
      "line_number": 5,
      "line_range": [5, 6],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b301_pickle.html",
      "test_id": "B301",
      "test_name": "pickle"
    },
    {
      "code": "hashlib.md5(data).hexdigest()",
      "col_number": 1,
      "filename": "/src/crypto.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 327,
        "link": "https://cwe.mitre.org/data/definitions/327.html"
      },
      "issue_severity": "MEDIUM",
      "issue_text": "Use of insecure MD5 hash function.",
      "line_number": 12,
      "line_range": [12, 12],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b303_md5.html",
      "test_id": "B303",
      "test_name": "md5"
    },
    {
      "code": "query = \"SELECT * FROM users WHERE id = %s\" % user_id",
      "col_number": 9,
      "filename": "/src/database.py",
      "issue_confidence": "MEDIUM",
      "issue_cwe": {
        "id": 89,
        "link": "https://cwe.mitre.org/data/definitions/89.html"
      },
      "issue_severity": "MEDIUM",
      "issue_text": "Possible SQL injection vector through string-based query construction.",
      "line_number": 20,
      "line_range": [20, 20],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b608_hardcoded_sql_expressions.html",
      "test_id": "B608",
      "test_name": "hardcoded_sql_expressions"
    },
    {
      "code": "app.run(debug=True)",
      "col_number": 1,
      "filename": "/src/app.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 489,
        "link": "https://cwe.mitre.org/data/definitions/489.html"
      },
      "issue_severity": "LOW",
      "issue_text": "A Flask app appears to be run with debug=True, which exposes the Werkzeug debugger and allows the execution of arbitrary code.",
      "line_number": 50,
      "line_range": [50, 50],
      "more_info": "https://bandit.readthedocs.io/en/latest/plugins/b201_flask_debug_true.html",
      "test_id": "B201",
      "test_name": "flask_debug_true"
    }
  ],
  "metrics": {
    "_totals": {
      "CONFIDENCE.HIGH": 3,
      "CONFIDENCE.LOW": 0,
      "CONFIDENCE.MEDIUM": 1,
      "CONFIDENCE.UNDEFINED": 0,
      "SEVERITY.HIGH": 1,
      "SEVERITY.LOW": 1,
      "SEVERITY.MEDIUM": 2,
      "SEVERITY.UNDEFINED": 0,
      "files_skipped": 5,
      "loc": 500,
      "nosec": 2,
      "skipped_tests": 1
    }
  }
}`

	config := agent.ScanConfig{RepoURL: "https://github.com/test/vulnerable-python-app"}
	
	findings, metadata, err := a.parseBanditOutput([]byte(banditOutput), config)
	
	require.NoError(t, err)
	assert.Len(t, findings, 4)
	
	// Verify different vulnerability categories are properly mapped
	categories := make(map[agent.VulnCategory]int)
	severities := make(map[agent.Severity]int)
	
	for _, finding := range findings {
		categories[finding.Category]++
		severities[finding.Severity]++
		
		// Verify all findings have required fields
		assert.NotEmpty(t, finding.ID)
		assert.Equal(t, AgentName, finding.Tool)
		assert.NotEmpty(t, finding.RuleID)
		assert.NotEmpty(t, finding.Title)
		assert.NotEmpty(t, finding.Description)
		assert.NotEmpty(t, finding.File)
		assert.Greater(t, finding.Line, 0)
		assert.Greater(t, finding.Confidence, 0.0)
	}
	
	// Verify we have different categories
	assert.Contains(t, categories, agent.CategoryInsecureDeserialization) // pickle
	assert.Contains(t, categories, agent.CategoryInsecureCrypto)           // md5
	assert.Contains(t, categories, agent.CategorySQLInjection)             // sql injection
	assert.Contains(t, categories, agent.CategoryMisconfiguration)         // flask debug
	
	// Verify we have different severities
	assert.Contains(t, severities, agent.SeverityHigh)   // pickle
	assert.Contains(t, severities, agent.SeverityMedium) // md5, sql
	assert.Contains(t, severities, agent.SeverityLow)    // flask debug
	
	// Check metadata
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 4, metadata.FilesScanned)
	assert.Equal(t, 500, metadata.LinesScanned)
	assert.Equal(t, "5", metadata.Environment["files_skipped"])
	assert.Equal(t, "2", metadata.Environment["nosec_comments"])
	assert.Equal(t, "1", metadata.Environment["skipped_tests"])
}

func TestBanditResult_JSONUnmarshaling(t *testing.T) {
	jsonData := `{
  "results": [
    {
      "code": "test_code",
      "col_number": 1,
      "filename": "/test.py",
      "issue_confidence": "HIGH",
      "issue_cwe": {
        "id": 123,
        "link": "https://example.com"
      },
      "issue_severity": "MEDIUM",
      "issue_text": "Test issue",
      "line_number": 5,
      "line_range": [5, 5],
      "more_info": "https://info.com",
      "test_id": "B999",
      "test_name": "test_rule"
    }
  ],
  "metrics": {
    "_totals": {
      "files_skipped": 1,
      "loc": 100,
      "nosec": 0,
      "skipped_tests": 0
    }
  }
}`

	var result BanditResult
	err := json.Unmarshal([]byte(jsonData), &result)
	
	require.NoError(t, err)
	assert.Len(t, result.Results, 1)
	
	finding := result.Results[0]
	assert.Equal(t, "test_code", finding.Code)
	assert.Equal(t, 1, finding.ColNumber)
	assert.Equal(t, "/test.py", finding.Filename)
	assert.Equal(t, "HIGH", finding.IssueConfidence)
	assert.Equal(t, 123, finding.IssueCwe.ID)
	assert.Equal(t, "MEDIUM", finding.IssueSeverity)
	assert.Equal(t, "Test issue", finding.IssueText)
	assert.Equal(t, 5, finding.LineNumber)
	assert.Equal(t, []int{5, 5}, finding.LineRange)
	assert.Equal(t, "https://info.com", finding.MoreInfo)
	assert.Equal(t, "B999", finding.TestID)
	assert.Equal(t, "test_rule", finding.TestName)
	
	assert.Equal(t, 1, result.Metrics.FilesSkipped)
	assert.Equal(t, 100, result.Metrics.LinesOfCode)
	assert.Equal(t, 0, result.Metrics.NoSec)
	assert.Equal(t, 0, result.Metrics.SkippedTests)
}

func TestBuildBanditCommand(t *testing.T) {
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo",
		Branch:  "main",
	}
	
	cmd := a.buildBanditCommand(context.Background(), config, "/tmp/repo", "/tmp/output")
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "docker", cmd.Path)
	
	// Check that the command contains expected arguments
	args := cmd.Args
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, a.config.DockerImage)
	
	// Check memory and CPU limits are set
	found := false
	for i, arg := range args {
		if arg == "--memory" && i+1 < len(args) {
			assert.Equal(t, "512m", args[i+1])
			found = true
			break
		}
	}
	assert.True(t, found, "Memory limit should be set")
	
	found = false
	for i, arg := range args {
		if arg == "--cpus" && i+1 < len(args) {
			assert.Equal(t, "1.0", args[i+1])
			found = true
			break
		}
	}
	assert.True(t, found, "CPU limit should be set")
}

func TestBuildBanditCommand_WithCustomConfig(t *testing.T) {
	config := AgentConfig{
		DockerImage:    "python:3.11-slim",
		MaxMemoryMB:    1024,
		MaxCPUCores:    2.0,
		Severity:       "high",
		Confidence:     "medium",
		SkipTests:      []string{"B101", "B102"},
		ExcludePaths:   []string{"*/tests/*", "*/venv/*"},
	}
	
	a := NewAgentWithConfig(config)
	
	scanConfig := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo",
		Branch:  "main",
	}
	
	cmd := a.buildBanditCommand(context.Background(), scanConfig, "/tmp/repo", "/tmp/output")
	
	assert.NotNil(t, cmd)
	
	// Check that environment variables are set for custom configuration
	envVars := cmd.Env
	severityFound := false
	for _, env := range envVars {
		if env == "BANDIT_SEVERITY=high" {
			severityFound = true
			break
		}
	}
	assert.True(t, severityFound, "BANDIT_SEVERITY environment variable should be set")
	
	// Note: cmd.Env might be nil if no custom env vars are set, so we check the args instead
	args := cmd.Args
	assert.Contains(t, args, "python:3.11-slim")
}