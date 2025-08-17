package semgrep

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSARIFOutput provides a sample SARIF output for testing
const mockSARIFOutput = `{
  "$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
  "version": "2.1.0",
  "runs": [
    {
      "tool": {
        "driver": {
          "name": "Semgrep",
          "version": "1.45.0",
          "informationUri": "https://semgrep.dev/",
          "rules": [
            {
              "id": "javascript.lang.security.audit.xss.react-dangerously-set-inner-html",
              "name": "react-dangerously-set-inner-html",
              "shortDescription": {
                "text": "Detected usage of dangerouslySetInnerHTML"
              },
              "fullDescription": {
                "text": "Detected usage of dangerouslySetInnerHTML which can lead to XSS vulnerabilities"
              },
              "defaultConfiguration": {
                "level": "error"
              },
              "properties": {
                "tags": ["security", "xss"]
              },
              "helpUri": "https://semgrep.dev/r/javascript.lang.security.audit.xss.react-dangerously-set-inner-html"
            }
          ]
        }
      },
      "results": [
        {
          "ruleId": "javascript.lang.security.audit.xss.react-dangerously-set-inner-html",
          "ruleIndex": 0,
          "level": "error",
          "message": {
            "text": "Detected usage of dangerouslySetInnerHTML which can lead to XSS vulnerabilities"
          },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": {
                  "uri": "/src/components/App.js"
                },
                "region": {
                  "startLine": 42,
                  "startColumn": 12,
                  "endLine": 42,
                  "endColumn": 45,
                  "snippet": {
                    "text": "dangerouslySetInnerHTML={{__html: userInput}}"
                  }
                }
              }
            }
          ],
          "properties": {
            "extra": {
              "severity": "high",
              "metadata": {
                "category": "xss",
                "confidence": "high",
                "references": [
                  "https://owasp.org/www-community/attacks/xss/"
                ]
              }
            }
          }
        }
      ]
    }
  ]
}`

func TestParseSARIFOutput(t *testing.T) {
	a := NewAgent()
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	findings, metadata, err := a.parseSARIFOutput([]byte(mockSARIFOutput), config)
	
	require.NoError(t, err)
	assert.Len(t, findings, 1)
	
	finding := findings[0]
	assert.Equal(t, "semgrep", finding.Tool)
	assert.Equal(t, "javascript.lang.security.audit.xss.react-dangerously-set-inner-html", finding.RuleID)
	assert.Equal(t, agent.SeverityHigh, finding.Severity)
	assert.Equal(t, agent.CategoryXSS, finding.Category)
	assert.Equal(t, "Detected usage of dangerouslySetInnerHTML which can lead to XSS vulnerabilities", finding.Title)
	assert.Equal(t, "components/App.js", finding.File)
	assert.Equal(t, 42, finding.Line)
	assert.Equal(t, 12, finding.Column)
	assert.Equal(t, "dangerouslySetInnerHTML={{__html: userInput}}", finding.Code)
	assert.Equal(t, 0.9, finding.Confidence)
	assert.Contains(t, finding.References, "https://owasp.org/www-community/attacks/xss/")
	
	assert.Equal(t, "1.45.0", metadata.ToolVersion)
	assert.Equal(t, "1.45.0", metadata.RulesVersion)
	assert.Equal(t, "sast", metadata.ScanType)
	assert.Equal(t, 1, metadata.FilesScanned)
}

func TestParseSARIFOutput_EmptyResults(t *testing.T) {
	a := NewAgent()
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	emptySARIF := `{
		"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		"version": "2.1.0",
		"runs": [
			{
				"tool": {
					"driver": {
						"name": "Semgrep",
						"version": "1.45.0",
						"rules": []
					}
				},
				"results": []
			}
		]
	}`
	
	findings, metadata, err := a.parseSARIFOutput([]byte(emptySARIF), config)
	
	require.NoError(t, err)
	assert.Len(t, findings, 0)
	assert.Equal(t, "1.45.0", metadata.ToolVersion)
	assert.Equal(t, 0, metadata.FilesScanned)
}

func TestParseSARIFOutput_InvalidJSON(t *testing.T) {
	a := NewAgent()
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	invalidJSON := `{"invalid": json}`
	
	_, _, err := a.parseSARIFOutput([]byte(invalidJSON), config)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse SARIF JSON")
}

func TestBuildSemgrepCommand(t *testing.T) {
	a := NewAgent()
	config := agent.ScanConfig{
		RepoURL:   "https://github.com/test/repo.git",
		Branch:    "main",
		Languages: []string{"javascript", "python"},
		Files:     []string{"*.js", "*.py"},
		Rules:     []string{"p/security-audit", "p/owasp-top-ten"},
	}
	
	repoPath := "/tmp/repo"
	tempDir := "/tmp/semgrep"
	
	cmd := a.buildSemgrepCommand(context.Background(), config, repoPath, tempDir)
	
	args := cmd.Args
	assert.Contains(t, args, "docker")
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, "--memory")
	assert.Contains(t, args, "512m")
	assert.Contains(t, args, "--cpus")
	assert.Contains(t, args, "1.0")
	assert.Contains(t, args, "-v")
	assert.Contains(t, args, "/tmp/repo:/src:ro")
	assert.Contains(t, args, "-v")
	assert.Contains(t, args, "/tmp/semgrep:/tmp/semgrep")
	assert.Contains(t, args, DefaultImage)
	assert.Contains(t, args, "--config")
	assert.Contains(t, args, "auto")
	assert.Contains(t, args, "--sarif")
	assert.Contains(t, args, "--output")
	assert.Contains(t, args, "/tmp/semgrep/results.sarif")
	assert.Contains(t, args, "--lang")
	assert.Contains(t, args, "javascript")
	assert.Contains(t, args, "--lang")
	assert.Contains(t, args, "python")
	assert.Contains(t, args, "--include")
	assert.Contains(t, args, "*.js")
	assert.Contains(t, args, "--include")
	assert.Contains(t, args, "*.py")
	assert.Contains(t, args, "--config")
	assert.Contains(t, args, "p/security-audit")
	assert.Contains(t, args, "--config")
	assert.Contains(t, args, "p/owasp-top-ten")
	assert.Contains(t, args, "/src")
}

func TestBuildSemgrepCommand_MinimalConfig(t *testing.T) {
	a := NewAgent()
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	repoPath := "/tmp/repo"
	tempDir := "/tmp/semgrep"
	
	cmd := a.buildSemgrepCommand(context.Background(), config, repoPath, tempDir)
	
	args := cmd.Args
	assert.Contains(t, args, "docker")
	assert.Contains(t, args, "run")
	assert.Contains(t, args, "--rm")
	assert.Contains(t, args, DefaultImage)
	assert.Contains(t, args, "--config")
	assert.Contains(t, args, "auto")
	assert.Contains(t, args, "--sarif")
	assert.Contains(t, args, "/src")
	
	// Should not contain language or file filters
	assert.NotContains(t, args, "--lang")
	assert.NotContains(t, args, "--include")
}

// TestPrepareRepository tests the repository preparation logic
// Note: This test requires git to be available and will make actual network calls
func TestPrepareRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	a := NewAgent()
	ctx := context.Background()
	
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "semgrep-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	repoPath := filepath.Join(tempDir, "repo")
	
	config := agent.ScanConfig{
		RepoURL: "https://github.com/octocat/Hello-World.git", // Small public repo
		Branch:  "master",
	}
	
	err = a.prepareRepository(ctx, config, repoPath)
	require.NoError(t, err)
	
	// Verify repository was cloned
	assert.DirExists(t, repoPath)
	assert.DirExists(t, filepath.Join(repoPath, ".git"))
}

func TestPrepareRepository_InvalidRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	a := NewAgent()
	ctx := context.Background()
	
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "semgrep-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	repoPath := filepath.Join(tempDir, "repo")
	
	config := agent.ScanConfig{
		RepoURL: "https://github.com/nonexistent/repo.git",
		Branch:  "main",
	}
	
	err = a.prepareRepository(ctx, config, repoPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git clone failed")
}

// Mock command execution for testing without Docker
type mockCmd struct {
	output []byte
	err    error
}

func (m *mockCmd) Output() ([]byte, error) {
	return m.output, m.err
}

func (m *mockCmd) Run() error {
	return m.err
}

// TestExecuteScan_MockedDocker tests the scan execution with mocked Docker
func TestExecuteScan_MockedDocker(t *testing.T) {
	// This test would require more sophisticated mocking of exec.Command
	// For now, we'll test the individual components that don't require Docker
	t.Skip("Mocking exec.Command requires more complex setup - covered by integration tests")
}

// Benchmark tests
func BenchmarkParseSARIFOutput(b *testing.B) {
	a := NewAgent()
	config := agent.ScanConfig{
		RepoURL: "https://github.com/test/repo.git",
		Branch:  "main",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := a.parseSARIFOutput([]byte(mockSARIFOutput), config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMapSeverity(b *testing.B) {
	a := NewAgent()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.mapSeverity("error", "high")
	}
}

func BenchmarkMapCategory(b *testing.B) {
	a := NewAgent()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.mapCategory("xss")
	}
}