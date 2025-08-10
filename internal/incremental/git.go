package incremental

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitRepositoryImpl implements the GitRepository interface
type GitRepositoryImpl struct{}

// NewGitRepository creates a new Git repository implementation
func NewGitRepository() *GitRepositoryImpl {
	return &GitRepositoryImpl{}
}

// GetFileChanges gets the list of changed files between two commits
func (g *GitRepositoryImpl) GetFileChanges(ctx context.Context, workingDir, fromCommit, toCommit string) ([]*FileChange, error) {
	// Use git diff to get changed files
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-status", fromCommit, toCommit)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run git diff: %w", err)
	}

	return g.parseGitDiffOutput(string(output))
}

// GetFileContent gets the content of a file
func (g *GitRepositoryImpl) GetFileContent(ctx context.Context, workingDir, filePath string) ([]byte, error) {
	fullPath := filepath.Join(workingDir, filePath)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return content, nil
}

// GetLastCommit gets the last commit hash for a branch
func (g *GitRepositoryImpl) GetLastCommit(ctx context.Context, workingDir, branch string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", branch)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get last commit: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// HasConfigChanges checks if there are configuration changes between commits
func (g *GitRepositoryImpl) HasConfigChanges(ctx context.Context, workingDir, fromCommit, toCommit string) (bool, error) {
	// Define configuration files that would trigger a full scan
	configFiles := []string{
		".agentscan.yml",
		".agentscan.yaml",
		"agentscan.yml",
		"agentscan.yaml",
		".semgrepignore",
		".eslintrc*",
		"eslint.config.*",
		"pyproject.toml",
		"setup.cfg",
		"bandit.yml",
		"bandit.yaml",
		".bandit",
		"go.mod",
		"go.sum",
		"package.json",
		"package-lock.json",
		"yarn.lock",
		"requirements.txt",
		"Pipfile",
		"Pipfile.lock",
		"poetry.lock",
		"Cargo.toml",
		"Cargo.lock",
	}

	// Check if any config files changed
	changes, err := g.GetFileChanges(ctx, workingDir, fromCommit, toCommit)
	if err != nil {
		return false, err
	}

	for _, change := range changes {
		for _, configFile := range configFiles {
			// Check exact match or pattern match
			if g.matchesConfigPattern(change.Path, configFile) {
				return true, nil
			}
		}
	}

	return false, nil
}

// parseGitDiffOutput parses the output of git diff --name-status
func (g *GitRepositoryImpl) parseGitDiffOutput(output string) ([]*FileChange, error) {
	var changes []*FileChange
	
	if strings.TrimSpace(output) == "" {
		return []*FileChange{}, nil // Return empty slice, not nil
	}
	
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		change := &FileChange{
			Path: filePath,
		}

		// Parse git status codes
		switch {
		case status == "A":
			change.ChangeType = ChangeTypeAdded
		case status == "M":
			change.ChangeType = ChangeTypeModified
		case status == "D":
			change.ChangeType = ChangeTypeDeleted
		case strings.HasPrefix(status, "R"):
			change.ChangeType = ChangeTypeRenamed
			if len(parts) >= 3 {
				change.OldPath = filePath
				change.Path = parts[2]
			}
		case strings.HasPrefix(status, "C"):
			// Copied file, treat as added
			change.ChangeType = ChangeTypeAdded
			if len(parts) >= 3 {
				change.Path = parts[2] // Use the destination file for copies
			}
		default:
			// Unknown status, treat as modified
			change.ChangeType = ChangeTypeModified
		}

		changes = append(changes, change)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing git diff output: %w", err)
	}

	return changes, nil
}

// matchesConfigPattern checks if a file path matches a config file pattern
func (g *GitRepositoryImpl) matchesConfigPattern(filePath, pattern string) bool {
	// Handle exact matches
	if filePath == pattern {
		return true
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		matched, err := filepath.Match(pattern, filepath.Base(filePath))
		if err == nil && matched {
			return true
		}
	}

	// Handle directory-based matches
	if strings.HasSuffix(filePath, "/"+pattern) {
		return true
	}

	return false
}

// GetCommitsBetween gets the list of commits between two commit hashes
func (g *GitRepositoryImpl) GetCommitsBetween(ctx context.Context, workingDir, fromCommit, toCommit string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--reverse", fromCommit+".."+toCommit)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits between %s and %s: %w", fromCommit, toCommit, err)
	}

	var commits []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		commit := strings.TrimSpace(scanner.Text())
		if commit != "" {
			commits = append(commits, commit)
		}
	}

	return commits, scanner.Err()
}

// GetFileAtCommit gets the content of a file at a specific commit
func (g *GitRepositoryImpl) GetFileAtCommit(ctx context.Context, workingDir, filePath, commit string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "show", commit+":"+filePath)
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		// File might not exist at this commit
		if strings.Contains(err.Error(), "does not exist") {
			return nil, fmt.Errorf("file %s does not exist at commit %s", filePath, commit)
		}
		return nil, fmt.Errorf("failed to get file %s at commit %s: %w", filePath, commit, err)
	}

	return output, nil
}

// IsGitRepository checks if the directory is a git repository
func (g *GitRepositoryImpl) IsGitRepository(ctx context.Context, workingDir string) bool {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	cmd.Dir = workingDir

	err := cmd.Run()
	return err == nil
}

// GetCurrentBranch gets the current branch name
func (g *GitRepositoryImpl) GetCurrentBranch(ctx context.Context, workingDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetRepositoryRoot gets the root directory of the git repository
func (g *GitRepositoryImpl) GetRepositoryRoot(ctx context.Context, workingDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get repository root: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetModifiedFiles gets the list of modified files in the working directory
func (g *GitRepositoryImpl) GetModifiedFiles(ctx context.Context, workingDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "HEAD")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get modified files: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file != "" {
			files = append(files, file)
		}
	}

	return files, scanner.Err()
}

// GetUntrackedFiles gets the list of untracked files
func (g *GitRepositoryImpl) GetUntrackedFiles(ctx context.Context, workingDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get untracked files: %w", err)
	}

	var files []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file != "" {
			files = append(files, file)
		}
	}

	return files, scanner.Err()
}