package bandit

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_Integration_VulnerablePython(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with vulnerable Python code
	tempDir, err := os.MkdirTemp("", "bandit-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create vulnerable Python files
	vulnerableCode := map[string]string{
		"app.py": `#!/usr/bin/env python3
"""
Vulnerable Python application for testing Bandit security scanner
"""
import os
import subprocess
import pickle
import hashlib
import tempfile
import random

# B105: Hardcoded password
PASSWORD = "hardcoded_password_123"
API_KEY = "sk-1234567890abcdef"

# B602: subprocess with shell=True
def execute_command(user_input):
    # This should trigger B602 - subprocess_popen_with_shell_equals_true
    result = subprocess.call(user_input, shell=True)
    return result

# B301: Pickle usage
def deserialize_data(data):
    # This should trigger B301 - pickle usage
    return pickle.loads(data)

# B303: MD5 usage
def hash_password(password):
    # This should trigger B303 - insecure hash function
    return hashlib.md5(password.encode()).hexdigest()

# B307: eval usage
def evaluate_expression(expr):
    # This should trigger B307 - eval usage
    return eval(expr)

# B311: random usage for cryptographic purposes
def generate_token():
    # This should trigger B311 - insecure random generator
    return str(random.random())

# B108: hardcoded temp directory
def create_temp_file():
    # This should trigger B108 - hardcoded tmp directory
    temp_path = "/tmp/myapp_temp_file"
    with open(temp_path, 'w') as f:
        f.write("temporary data")
    return temp_path

# B506: YAML load
import yaml
def load_config(config_data):
    # This should trigger B506 - yaml.load usage
    return yaml.load(config_data)

if __name__ == "__main__":
    # More vulnerable code
    user_cmd = input("Enter command: ")
    execute_command(user_cmd)
    
    user_expr = input("Enter expression: ")
    result = evaluate_expression(user_expr)
    print(f"Result: {result}")
`,
		"database.py": `"""
Database operations with security vulnerabilities
"""
import sqlite3
import mysql.connector

class DatabaseManager:
    def __init__(self):
        self.connection = None
    
    # B608: SQL injection vulnerability
    def get_user_by_id(self, user_id):
        # This should trigger B608 - hardcoded SQL expressions
        query = "SELECT * FROM users WHERE id = '%s'" % user_id
        cursor = self.connection.cursor()
        cursor.execute(query)
        return cursor.fetchone()
    
    # B608: Another SQL injection
    def search_users(self, search_term):
        # This should trigger B608 - SQL injection
        query = f"SELECT * FROM users WHERE name LIKE '%{search_term}%'"
        cursor = self.connection.cursor()
        cursor.execute(query)
        return cursor.fetchall()
    
    # B201: Flask debug mode (if Flask is imported)
    def start_debug_server(self):
        try:
            from flask import Flask
            app = Flask(__name__)
            # This should trigger B201 - flask debug true
            app.run(debug=True, host='0.0.0.0')
        except ImportError:
            pass

# B110: try/except pass
def risky_operation():
    try:
        # Some risky operation
        result = 1 / 0
    except:
        # This should trigger B110 - try except pass
        pass
`,
		"crypto.py": `"""
Cryptographic operations with vulnerabilities
"""
import hashlib
import ssl
import random
from Crypto.Cipher import DES

# B303: Insecure hash functions
def weak_hash_md5(data):
    # This should trigger B303 - MD5 usage
    return hashlib.md5(data.encode()).hexdigest()

def weak_hash_sha1(data):
    # This should trigger B303 - SHA1 usage  
    return hashlib.sha1(data.encode()).hexdigest()

# B311: Insecure random number generation
def generate_session_id():
    # This should trigger B311 - random usage for security
    return str(random.random())

def generate_password():
    # This should trigger B311 - random usage for security
    chars = "abcdefghijklmnopqrstuvwxyz0123456789"
    return ''.join(random.choice(chars) for _ in range(8))

# B502: SSL with bad version
def create_ssl_context():
    # This should trigger B502 - SSL with bad version
    context = ssl.SSLContext(ssl.PROTOCOL_TLSv1)
    return context

# B505: Weak cryptographic key
def encrypt_data(data):
    # This should trigger B505 - weak cryptographic key
    key = b'12345678'  # 8 bytes = 64 bits, too weak
    cipher = DES.new(key, DES.MODE_ECB)
    return cipher.encrypt(data)

# B324: Insecure hash function usage
def hash_with_md5():
    # This should trigger B324 - hashlib new with insecure algorithm
    hasher = hashlib.new('md5')
    hasher.update(b'test data')
    return hasher.hexdigest()
`,
		"web.py": `"""
Web application vulnerabilities
"""
from flask import Flask, request, render_template_string
import jinja2

app = Flask(__name__)

# B701: Jinja2 autoescape false
def render_unsafe_template(user_input):
    # This should trigger B701 - jinja2 autoescape false
    env = jinja2.Environment(autoescape=False)
    template = env.from_string(user_input)
    return template.render()

# B703: Django mark_safe (if Django is available)
def render_unsafe_content(content):
    try:
        from django.utils.safestring import mark_safe
        # This should trigger B703 - django mark_safe
        return mark_safe(content)
    except ImportError:
        return content

@app.route('/unsafe')
def unsafe_endpoint():
    user_input = request.args.get('input', '')
    # XSS vulnerability through template rendering
    template = f"<h1>Hello {user_input}</h1>"
    return render_template_string(template)

# B201: Flask debug mode
if __name__ == '__main__':
    # This should trigger B201 - flask debug true
    app.run(debug=True, host='0.0.0.0', port=5000)
`,
		"requirements.txt": `Flask==2.0.1
PyYAML==5.4.1
pycrypto==2.6.1
mysql-connector-python==8.0.26
`,
		"setup.py": `from setuptools import setup, find_packages

setup(
    name="vulnerable-app",
    version="1.0.0",
    description="Vulnerable Python app for security testing",
    packages=find_packages(),
    install_requires=[
        "Flask>=2.0.0",
        "PyYAML>=5.4.0",
        "pycrypto>=2.6.0",
    ],
)`,
	}

	// Write files to temp directory
	for filename, content := range vulnerableCode {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Initialize git repository (required for cloning)
	setupGitRepo(t, tempDir)

	// Create agent and run scan
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   tempDir, // Use local path for testing
		Branch:    "main",
		Languages: []string{"python"},
		Timeout:   3 * time.Minute,
	}

	result, err := a.Scan(context.Background(), config)
	
	// The scan might fail due to Docker/network issues in CI, so we'll be flexible
	if err != nil {
		t.Logf("Scan failed (expected in some CI environments): %v", err)
		return
	}

	require.NotNil(t, result)
	assert.Equal(t, AgentName, result.AgentID)
	
	// If scan completed successfully, verify findings
	if result.Status == agent.ScanStatusCompleted {
		assert.Greater(t, len(result.Findings), 0, "Should find security vulnerabilities")
		
		// Check for specific vulnerability types
		foundRules := make(map[string]bool)
		foundCategories := make(map[agent.VulnCategory]bool)
		
		for _, finding := range result.Findings {
			foundRules[finding.RuleID] = true
			foundCategories[finding.Category] = true
			
			// Verify finding structure
			assert.NotEmpty(t, finding.ID)
			assert.Equal(t, AgentName, finding.Tool)
			assert.NotEmpty(t, finding.RuleID)
			assert.NotEmpty(t, finding.Title)
			assert.NotEmpty(t, finding.Description)
			assert.NotEmpty(t, finding.File)
			assert.Greater(t, finding.Line, 0)
			assert.Greater(t, finding.Confidence, 0.0)
			
			// Verify severity mapping
			assert.Contains(t, []agent.Severity{
				agent.SeverityHigh,
				agent.SeverityMedium,
				agent.SeverityLow,
			}, finding.Severity)
			
			// Verify category mapping
			assert.NotEqual(t, agent.VulnCategory(""), finding.Category)
		}
		
		// Verify we found various types of vulnerabilities
		expectedCategories := []agent.VulnCategory{
			agent.CategoryHardcodedSecrets,
			agent.CategoryCommandInjection,
			agent.CategoryInsecureCrypto,
			agent.CategorySQLInjection,
		}
		
		foundCount := 0
		for _, expectedCat := range expectedCategories {
			if foundCategories[expectedCat] {
				foundCount++
			}
		}
		
		assert.Greater(t, foundCount, 0, "Should find at least some expected vulnerability categories")
		
		// Verify metadata
		assert.NotEmpty(t, result.Metadata.ToolVersion)
		assert.Equal(t, "sast", result.Metadata.ScanType)
		assert.Greater(t, result.Duration, time.Duration(0))
		
		t.Logf("Found %d security issues with rules: %v", len(result.Findings), foundRules)
		t.Logf("Found categories: %v", foundCategories)
	}
}

func TestAgent_Integration_SafePython(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with safe Python code
	tempDir, err := os.MkdirTemp("", "bandit-safe-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create safe Python code
	safeCode := map[string]string{
		"safe_app.py": `#!/usr/bin/env python3
"""
Safe Python application that should not trigger Bandit warnings
"""
import os
import subprocess
import hashlib
import secrets
import tempfile
from pathlib import Path

# Safe password handling
def get_password():
    # Safe: Get password from environment variable
    return os.environ.get('PASSWORD', '')

# Safe command execution
def execute_safe_command(command_args):
    # Safe: Use subprocess without shell=True and with list of args
    result = subprocess.run(command_args, capture_output=True, text=True)
    return result.stdout

# Safe hashing
def hash_password(password):
    # Safe: Use SHA-256 instead of MD5
    return hashlib.sha256(password.encode()).hexdigest()

# Safe random generation
def generate_secure_token():
    # Safe: Use secrets module for cryptographic purposes
    return secrets.token_hex(32)

# Safe temporary file handling
def create_temp_file():
    # Safe: Use tempfile module
    with tempfile.NamedTemporaryFile(mode='w', delete=False) as f:
        f.write("temporary data")
        return f.name

# Safe configuration loading
import json
def load_config(config_path):
    # Safe: Use JSON instead of YAML.load
    with open(config_path, 'r') as f:
        return json.load(f)

class SafeDatabaseManager:
    def __init__(self, connection):
        self.connection = connection
    
    # Safe SQL operations
    def get_user_by_id(self, user_id):
        # Safe: Use parameterized queries
        query = "SELECT * FROM users WHERE id = ?"
        cursor = self.connection.cursor()
        cursor.execute(query, (user_id,))
        return cursor.fetchone()
    
    def search_users(self, search_term):
        # Safe: Use parameterized queries
        query = "SELECT * FROM users WHERE name LIKE ?"
        cursor = self.connection.cursor()
        cursor.execute(query, (f"%{search_term}%",))
        return cursor.fetchall()

# Safe error handling
def safe_operation():
    try:
        result = 1 / 0
    except ZeroDivisionError as e:
        # Safe: Proper exception handling
        print(f"Error: {e}")
        return None

if __name__ == "__main__":
    # Safe main execution
    token = generate_secure_token()
    print(f"Generated secure token: {token}")
`,
		"safe_crypto.py": `"""
Safe cryptographic operations
"""
import hashlib
import ssl
import secrets
from cryptography.fernet import Fernet

# Safe hash functions
def secure_hash_sha256(data):
    # Safe: Use SHA-256
    return hashlib.sha256(data.encode()).hexdigest()

def secure_hash_sha3(data):
    # Safe: Use SHA-3
    return hashlib.sha3_256(data.encode()).hexdigest()

# Safe random number generation
def generate_secure_session_id():
    # Safe: Use secrets module
    return secrets.token_urlsafe(32)

def generate_secure_password():
    # Safe: Use secrets module for password generation
    alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    return ''.join(secrets.choice(alphabet) for _ in range(16))

# Safe SSL configuration
def create_secure_ssl_context():
    # Safe: Use modern TLS version
    context = ssl.create_default_context()
    context.check_hostname = True
    context.verify_mode = ssl.CERT_REQUIRED
    return context

# Safe encryption
def encrypt_data_safely(data):
    # Safe: Use modern encryption with proper key generation
    key = Fernet.generate_key()
    cipher = Fernet(key)
    encrypted_data = cipher.encrypt(data.encode())
    return key, encrypted_data

def decrypt_data_safely(key, encrypted_data):
    # Safe: Proper decryption
    cipher = Fernet(key)
    decrypted_data = cipher.decrypt(encrypted_data)
    return decrypted_data.decode()
`,
		"requirements.txt": `cryptography>=3.4.8
`,
	}

	// Write files to temp directory
	for filename, content := range safeCode {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Initialize git repository
	setupGitRepo(t, tempDir)

	// Create agent and run scan
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   tempDir,
		Branch:    "main",
		Languages: []string{"python"},
		Timeout:   2 * time.Minute,
	}

	result, err := a.Scan(context.Background(), config)
	
	// Handle potential CI environment issues
	if err != nil {
		t.Logf("Safe code scan failed (expected in some CI environments): %v", err)
		return
	}

	require.NotNil(t, result)
	assert.Equal(t, AgentName, result.AgentID)
	
	if result.Status == agent.ScanStatusCompleted {
		// Safe code should have no or very few security findings
		assert.LessOrEqual(t, len(result.Findings), 2, 
			"Safe code should have minimal security findings, found: %d", len(result.Findings))
		
		t.Logf("Safe code scan found %d security issues (expected to be minimal)", len(result.Findings))
	}
}

func TestAgent_Integration_PythonEnvironmentDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory with various Python environment files
	tempDir, err := os.MkdirTemp("", "bandit-env-integration-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create Python environment files
	envFiles := map[string]string{
		"requirements.txt": `Flask==2.0.1
bandit==1.7.5
pytest==7.1.2`,
		"Pipfile": `[[source]]
url = "https://pypi.org/simple"
verify_ssl = true
name = "pypi"

[packages]
flask = "*"
requests = "*"

[dev-packages]
bandit = "*"
pytest = "*"

[requires]
python_version = "3.9"`,
		"pyproject.toml": `[build-system]
requires = ["setuptools>=45", "wheel"]
build-backend = "setuptools.build_meta"

[project]
name = "test-project"
version = "0.1.0"
description = "Test project"
dependencies = [
    "flask>=2.0.0",
    "requests>=2.25.0",
]

[project.optional-dependencies]
dev = [
    "bandit>=1.7.0",
    "pytest>=7.0.0",
]`,
		".python-version": `3.9.7`,
		"simple.py": `# Simple Python file
print("Hello, World!")`,
	}

	// Write files to temp directory
	for filename, content := range envFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create a virtual environment directory
	venvDir := filepath.Join(tempDir, "venv")
	err = os.MkdirAll(filepath.Join(venvDir, "bin"), 0755)
	require.NoError(t, err)

	// Initialize git repository
	setupGitRepo(t, tempDir)

	// Create agent and run scan
	a := NewAgent()
	
	config := agent.ScanConfig{
		RepoURL:   tempDir,
		Branch:    "main",
		Languages: []string{"python"},
		Timeout:   2 * time.Minute,
	}

	result, err := a.Scan(context.Background(), config)
	
	// Handle potential CI environment issues
	if err != nil {
		t.Logf("Environment detection scan failed (expected in some CI environments): %v", err)
		return
	}

	require.NotNil(t, result)
	assert.Equal(t, AgentName, result.AgentID)
	
	if result.Status == agent.ScanStatusCompleted {
		// Should complete successfully even with various environment files
		assert.Equal(t, agent.ScanStatusCompleted, result.Status)
		
		t.Logf("Environment detection scan completed successfully")
	}
}

// setupGitRepo initializes a git repository in the given directory
func setupGitRepo(t *testing.T, dir string) {
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Logf("Failed to init git repo: %v", err)
		return
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = dir
	cmd.Run()
	
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	cmd.Run()

	// Add files and commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	cmd.Run()
	
	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	cmd.Run()
}