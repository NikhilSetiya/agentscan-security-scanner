package eslint

import (
	"encoding/json"
	"testing"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
	"github.com/stretchr/testify/assert"
)

func TestGenerateFindingID(t *testing.T) {
	tests := []struct {
		name     string
		ruleID   string
		file     string
		line     int
		expected string
	}{
		{
			name:     "basic finding ID",
			ruleID:   "security/detect-eval-with-expression",
			file:     "/app/src/test.js",
			line:     42,
			expected: "eslint-security/detect-eval-with-expression-test.js-42",
		},
		{
			name:     "core rule finding ID",
			ruleID:   "no-eval",
			file:     "/app/index.js",
			line:     1,
			expected: "eslint-no-eval-index.js-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFindingID(tt.ruleID, tt.file, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseESLintOutput_EmptyResults(t *testing.T) {
	a := NewAgent()
	
	emptyOutput := `[]`
	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseESLintOutput([]byte(emptyOutput), config)
	
	assert.NoError(t, err)
	assert.Empty(t, findings)
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 0, metadata.FilesScanned)
}

func TestParseESLintOutput_InvalidJSON(t *testing.T) {
	a := NewAgent()
	
	invalidOutput := `invalid json`
	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseESLintOutput([]byte(invalidOutput), config)
	
	assert.Error(t, err)
	assert.Nil(t, findings)
	assert.Equal(t, agent.Metadata{}, metadata)
}

func TestParseESLintOutput_WithFindings(t *testing.T) {
	a := NewAgent()
	
	eslintOutput := `[
  {
    "filePath": "/app/vulnerable.js",
    "messages": [
      {
        "ruleId": "security/detect-eval-with-expression",
        "severity": 2,
        "message": "eval can be harmful.",
        "line": 10,
        "column": 8,
        "nodeType": "CallExpression",
        "endLine": 10,
        "endColumn": 25,
        "fix": {
          "range": [150, 175],
          "text": "// Safe alternative"
        }
      },
      {
        "ruleId": "security/detect-object-injection",
        "severity": 2,
        "message": "Variable assignment based on user input.",
        "line": 15,
        "column": 5,
        "nodeType": "AssignmentExpression",
        "endLine": 15,
        "endColumn": 20
      },
      {
        "ruleId": "no-unused-vars",
        "severity": 1,
        "message": "'unused' is defined but never used.",
        "line": 5,
        "column": 7,
        "nodeType": "Identifier",
        "endLine": 5,
        "endColumn": 13
      }
    ],
    "errorCount": 2,
    "warningCount": 1,
    "fixableErrorCount": 1,
    "fixableWarningCount": 0,
    "source": "function test() {\n  var unused = 'test';\n  \n  // Dangerous eval\n  eval('console.log(\"test\")');\n  \n  // Object injection\n  obj[userInput] = value;\n}"
  }
]`

	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseESLintOutput([]byte(eslintOutput), config)
	
	assert.NoError(t, err)
	
	// Should only include security-related findings (2 security issues, not the unused var)
	assert.Len(t, findings, 2)
	
	// Check first finding (eval)
	finding1 := findings[0]
	assert.Equal(t, AgentName, finding1.Tool)
	assert.Equal(t, "security/detect-eval-with-expression", finding1.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding1.Severity)
	assert.Equal(t, agent.CategoryCommandInjection, finding1.Category)
	assert.Equal(t, "Dangerous eval() usage detected", finding1.Title)
	assert.Equal(t, "eval can be harmful.", finding1.Description)
	assert.Equal(t, "vulnerable.js", finding1.File)
	assert.Equal(t, 10, finding1.Line)
	assert.Equal(t, 8, finding1.Column)
	assert.Equal(t, 0.9, finding1.Confidence)
	assert.NotNil(t, finding1.Fix)
	assert.Equal(t, "Replace with: // Safe alternative", finding1.Fix.Description)
	assert.NotEmpty(t, finding1.References)
	
	// Check second finding (object injection)
	finding2 := findings[1]
	assert.Equal(t, "security/detect-object-injection", finding2.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding2.Severity)
	assert.Equal(t, agent.CategoryCommandInjection, finding2.Category)
	assert.Equal(t, 15, finding2.Line)
	assert.Nil(t, finding2.Fix) // No fix provided for this one
	
	// Check metadata
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 1, metadata.FilesScanned)
	assert.Greater(t, metadata.LinesScanned, 0)
	assert.Contains(t, metadata.Environment, "errors")
	assert.Contains(t, metadata.Environment, "warnings")
	assert.Equal(t, "2", metadata.Environment["errors"])
	assert.Equal(t, "1", metadata.Environment["warnings"])
}

func TestParseESLintOutput_NoSecurityFindings(t *testing.T) {
	a := NewAgent()
	
	eslintOutput := `[
  {
    "filePath": "/app/clean.js",
    "messages": [
      {
        "ruleId": "no-unused-vars",
        "severity": 1,
        "message": "'unused' is defined but never used.",
        "line": 5,
        "column": 7,
        "nodeType": "Identifier",
        "endLine": 5,
        "endColumn": 13
      },
      {
        "ruleId": "semi",
        "severity": 2,
        "message": "Missing semicolon.",
        "line": 8,
        "column": 20,
        "nodeType": "ExpressionStatement",
        "endLine": 8,
        "endColumn": 20
      }
    ],
    "errorCount": 1,
    "warningCount": 1,
    "fixableErrorCount": 0,
    "fixableWarningCount": 1
  }
]`

	config := agent.ScanConfig{RepoURL: "https://github.com/test/repo"}
	
	findings, metadata, err := a.parseESLintOutput([]byte(eslintOutput), config)
	
	assert.NoError(t, err)
	
	// Should have no security findings
	assert.Empty(t, findings)
	
	// Metadata should still be populated
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 1, metadata.FilesScanned)
	assert.Equal(t, "1", metadata.Environment["errors"])
	assert.Equal(t, "1", metadata.Environment["warnings"])
}

func TestExtractCodeSnippet(t *testing.T) {
	a := NewAgent()
	
	tests := []struct {
		name     string
		source   string
		line     int
		expected string
	}{
		{
			name: "extract middle line",
			source: `function test() {
    var x = 1;
    eval("dangerous");
    return x;
}`,
			line:     3,
			expected: `eval("dangerous");`,
		},
		{
			name: "extract first line",
			source: `console.log("first");
console.log("second");`,
			line:     1,
			expected: `console.log("first");`,
		},
		{
			name:     "empty source",
			source:   "",
			line:     1,
			expected: "",
		},
		{
			name:     "line out of bounds",
			source:   "single line",
			line:     5,
			expected: "",
		},
		{
			name:     "zero line",
			source:   "single line",
			line:     0,
			expected: "",
		},
		{
			name: "line with whitespace",
			source: `function test() {
    
    eval("test");
    
}`,
			line:     3,
			expected: `eval("test");`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := a.extractCodeSnippet(tt.source, tt.line)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateESLintConfig(t *testing.T) {
	a := NewAgent()
	
	config := a.generateESLintConfig()
	
	assert.NotEmpty(t, config)
	assert.Contains(t, config, `"security"`)
	assert.Contains(t, config, `"no-eval"`)
	assert.Contains(t, config, `"security/detect-eval-with-expression"`)
	assert.Contains(t, config, `"error"`)
	
	// Verify it's valid JSON by attempting to parse
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(config), &parsed)
	assert.NoError(t, err)
	
	// Check structure
	assert.Contains(t, parsed, "env")
	assert.Contains(t, parsed, "plugins")
	assert.Contains(t, parsed, "rules")
	
	plugins, ok := parsed["plugins"].([]interface{})
	assert.True(t, ok)
	assert.Contains(t, plugins, "security")
	
	rules, ok := parsed["rules"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, rules, "no-eval")
	assert.Contains(t, rules, "security/detect-eval-with-expression")
}