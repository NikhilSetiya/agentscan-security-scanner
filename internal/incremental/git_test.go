package incremental

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitRepositoryImpl_parseGitDiffOutput(t *testing.T) {
	git := NewGitRepository()

	tests := []struct {
		name     string
		output   string
		expected []*FileChange
	}{
		{
			name:   "added file",
			output: "A\tsrc/main.go",
			expected: []*FileChange{
				{Path: "src/main.go", ChangeType: ChangeTypeAdded},
			},
		},
		{
			name:   "modified file",
			output: "M\tsrc/utils.go",
			expected: []*FileChange{
				{Path: "src/utils.go", ChangeType: ChangeTypeModified},
			},
		},
		{
			name:   "deleted file",
			output: "D\tsrc/old.go",
			expected: []*FileChange{
				{Path: "src/old.go", ChangeType: ChangeTypeDeleted},
			},
		},
		{
			name:   "renamed file",
			output: "R100\tsrc/old.go\tsrc/new.go",
			expected: []*FileChange{
				{Path: "src/new.go", ChangeType: ChangeTypeRenamed, OldPath: "src/old.go"},
			},
		},
		{
			name:   "multiple changes",
			output: "A\tsrc/new.go\nM\tsrc/main.go\nD\tsrc/old.go",
			expected: []*FileChange{
				{Path: "src/new.go", ChangeType: ChangeTypeAdded},
				{Path: "src/main.go", ChangeType: ChangeTypeModified},
				{Path: "src/old.go", ChangeType: ChangeTypeDeleted},
			},
		},
		{
			name:     "empty output",
			output:   "",
			expected: []*FileChange{},
		},
		{
			name:   "copied file",
			output: "C100\tsrc/template.go\tsrc/copy.go",
			expected: []*FileChange{
				{Path: "src/copy.go", ChangeType: ChangeTypeAdded},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := git.parseGitDiffOutput(tt.output)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitRepositoryImpl_matchesConfigPattern(t *testing.T) {
	git := NewGitRepository()

	tests := []struct {
		name     string
		filePath string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			filePath: ".agentscan.yml",
			pattern:  ".agentscan.yml",
			expected: true,
		},
		{
			name:     "wildcard match",
			filePath: ".eslintrc.js",
			pattern:  ".eslintrc*",
			expected: true,
		},
		{
			name:     "directory match",
			filePath: "config/.agentscan.yml",
			pattern:  ".agentscan.yml",
			expected: true,
		},
		{
			name:     "no match",
			filePath: "src/main.go",
			pattern:  ".agentscan.yml",
			expected: false,
		},
		{
			name:     "wildcard no match",
			filePath: "src/main.go",
			pattern:  ".eslintrc*",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := git.matchesConfigPattern(tt.filePath, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Integration tests (these would require a real git repository)
// These tests are commented out as they require actual git setup

/*
func TestGitRepositoryImpl_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// This test would require setting up a real git repository
	// with actual commits and files for testing
	git := NewGitRepository()
	ctx := context.Background()

	// Test with a temporary git repository
	tempDir := t.TempDir()
	
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	err := cmd.Run()
	require.NoError(t, err)

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Create and commit initial file
	filePath := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Get initial commit hash
	initialCommit, err := git.GetLastCommit(ctx, tempDir, "HEAD")
	require.NoError(t, err)
	assert.NotEmpty(t, initialCommit)

	// Modify file and commit
	err = os.WriteFile(filePath, []byte("package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}\n"), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", "main.go")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Update main function")
	cmd.Dir = tempDir
	err = cmd.Run()
	require.NoError(t, err)

	// Get second commit hash
	secondCommit, err := git.GetLastCommit(ctx, tempDir, "HEAD")
	require.NoError(t, err)
	assert.NotEmpty(t, secondCommit)
	assert.NotEqual(t, initialCommit, secondCommit)

	// Test GetFileChanges
	changes, err := git.GetFileChanges(ctx, tempDir, initialCommit, secondCommit)
	require.NoError(t, err)
	assert.Len(t, changes, 1)
	assert.Equal(t, "main.go", changes[0].Path)
	assert.Equal(t, ChangeTypeModified, changes[0].ChangeType)

	// Test GetFileContent
	content, err := git.GetFileContent(ctx, tempDir, "main.go")
	require.NoError(t, err)
	assert.Contains(t, string(content), "println")

	// Test IsGitRepository
	assert.True(t, git.IsGitRepository(ctx, tempDir))
	assert.False(t, git.IsGitRepository(ctx, "/tmp"))

	// Test GetCurrentBranch
	branch, err := git.GetCurrentBranch(ctx, tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, branch)

	// Test GetRepositoryRoot
	root, err := git.GetRepositoryRoot(ctx, tempDir)
	require.NoError(t, err)
	assert.Equal(t, tempDir, root)
}
*/

func TestGitRepositoryImpl_HasConfigChanges_MockScenarios(t *testing.T) {
	// This test demonstrates the logic without requiring actual git commands
	git := NewGitRepository()

	// Test the config file patterns
	configFiles := []string{
		".agentscan.yml",
		".eslintrc.js",
		"package.json",
		"go.mod",
		"requirements.txt",
	}

	// Simulate different file change scenarios
	testChanges := []*FileChange{
		{Path: "src/main.go", ChangeType: ChangeTypeModified},
		{Path: ".agentscan.yml", ChangeType: ChangeTypeModified},
		{Path: "package.json", ChangeType: ChangeTypeAdded},
		{Path: "README.md", ChangeType: ChangeTypeModified},
	}

	// Check which changes would trigger config detection
	hasConfigChange := false
	for _, change := range testChanges {
		for _, configFile := range configFiles {
			if git.matchesConfigPattern(change.Path, configFile) {
				hasConfigChange = true
				break
			}
		}
		if hasConfigChange {
			break
		}
	}

	assert.True(t, hasConfigChange, "Should detect config changes in test scenario")
}

// Benchmark tests
func BenchmarkGitRepositoryImpl_parseGitDiffOutput(b *testing.B) {
	git := NewGitRepository()
	output := "A\tsrc/new.go\nM\tsrc/main.go\nD\tsrc/old.go\nR100\tsrc/old.go\tsrc/renamed.go"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := git.parseGitDiffOutput(output)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGitRepositoryImpl_matchesConfigPattern(b *testing.B) {
	git := NewGitRepository()
	filePath := ".eslintrc.js"
	pattern := ".eslintrc*"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = git.matchesConfigPattern(filePath, pattern)
	}
}