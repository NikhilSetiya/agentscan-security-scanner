package zap

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/agent"
)

// WebAppConfig contains configuration for a detected web application
type WebAppConfig struct {
	Framework    string        `json:"framework"`     // react, express, django, etc.
	StartCommand string        `json:"start_command"` // Command to start the app
	Port         int           `json:"port"`          // Port the app runs on
	HealthCheck  string        `json:"health_check"`  // Health check endpoint
	Timeout      time.Duration `json:"timeout"`       // Time to wait for startup
	BuildCommand string        `json:"build_command,omitempty"` // Optional build command
}

// RunningApp represents a running web application
type RunningApp struct {
	ContainerID string `json:"container_id"`
	URL         string `json:"url"`
	Port        int    `json:"port"`
	Process     *os.Process `json:"-"`
}

// detectWebApplication analyzes the repository to determine if it's a web application
func (a *Agent) detectWebApplication(ctx context.Context, config agent.ScanConfig) (*WebAppConfig, error) {
	// Create temporary directory for the scan
	tempDir, err := os.MkdirTemp("", "zap-detect-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		return nil, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Check for common web application indicators
	webAppConfig := a.analyzeRepository(repoPath)
	return webAppConfig, nil
}

// analyzeRepository analyzes the repository structure to detect web applications
func (a *Agent) analyzeRepository(repoPath string) *WebAppConfig {
	// Check for package.json (Node.js/JavaScript)
	if packageJSON := filepath.Join(repoPath, "package.json"); fileExists(packageJSON) {
		if config := a.analyzeNodeJS(packageJSON); config != nil {
			return config
		}
	}

	// Check for requirements.txt or setup.py (Python)
	if fileExists(filepath.Join(repoPath, "requirements.txt")) || fileExists(filepath.Join(repoPath, "setup.py")) {
		if config := a.analyzePython(repoPath); config != nil {
			return config
		}
	}

	// Check for pom.xml or build.gradle (Java)
	if fileExists(filepath.Join(repoPath, "pom.xml")) || fileExists(filepath.Join(repoPath, "build.gradle")) {
		if config := a.analyzeJava(repoPath); config != nil {
			return config
		}
	}

	// Check for composer.json (PHP)
	if fileExists(filepath.Join(repoPath, "composer.json")) {
		if config := a.analyzePHP(repoPath); config != nil {
			return config
		}
	}

	// Check for Gemfile (Ruby)
	if fileExists(filepath.Join(repoPath, "Gemfile")) {
		if config := a.analyzeRuby(repoPath); config != nil {
			return config
		}
	}

	// Check for go.mod (Go)
	if fileExists(filepath.Join(repoPath, "go.mod")) {
		if config := a.analyzeGo(repoPath); config != nil {
			return config
		}
	}

	// Check for Dockerfile
	if fileExists(filepath.Join(repoPath, "Dockerfile")) {
		if config := a.analyzeDockerfile(repoPath); config != nil {
			return config
		}
	}

	return nil // Not a detectable web application
}

// analyzeNodeJS analyzes Node.js applications
func (a *Agent) analyzeNodeJS(packageJSONPath string) *WebAppConfig {
	data, err := ioutil.ReadFile(packageJSONPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	// Check for web framework dependencies
	webFrameworks := []string{"express", "koa", "fastify", "next", "nuxt", "react-scripts", "vue-cli-service"}
	isWebApp := false
	framework := "nodejs"

	for _, fw := range webFrameworks {
		if _, exists := pkg.Dependencies[fw]; exists {
			isWebApp = true
			framework = fw
			break
		}
		if _, exists := pkg.DevDependencies[fw]; exists {
			isWebApp = true
			framework = fw
			break
		}
	}

	if !isWebApp {
		return nil
	}

	// Determine start command
	startCommand := "npm start"
	if start, exists := pkg.Scripts["start"]; exists {
		startCommand = "npm run start"
		_ = start // Use the variable
	} else if dev, exists := pkg.Scripts["dev"]; exists {
		startCommand = "npm run dev"
		_ = dev // Use the variable
	}

	// Determine build command if needed
	buildCommand := ""
	if _, exists := pkg.Scripts["build"]; exists {
		buildCommand = "npm run build"
	}

	return &WebAppConfig{
		Framework:    framework,
		StartCommand: startCommand,
		Port:         3000, // Common default for Node.js apps
		HealthCheck:  "/",
		Timeout:      60 * time.Second,
		BuildCommand: buildCommand,
	}
}

// analyzePython analyzes Python web applications
func (a *Agent) analyzePython(repoPath string) *WebAppConfig {
	// Check for common Python web frameworks
	requirementsPath := filepath.Join(repoPath, "requirements.txt")
	if !fileExists(requirementsPath) {
		return nil
	}

	data, err := ioutil.ReadFile(requirementsPath)
	if err != nil {
		return nil
	}

	requirements := string(data)
	webFrameworks := map[string]string{
		"django":  "django",
		"flask":   "flask",
		"fastapi": "fastapi",
		"tornado": "tornado",
	}

	framework := ""
	for fw, name := range webFrameworks {
		if strings.Contains(strings.ToLower(requirements), fw) {
			framework = name
			break
		}
	}

	if framework == "" {
		return nil
	}

	// Determine start command based on framework
	startCommand := "python app.py"
	port := 8000

	switch framework {
	case "django":
		startCommand = "python manage.py runserver 0.0.0.0:8000"
		port = 8000
	case "flask":
		startCommand = "python app.py"
		port = 5000
	case "fastapi":
		startCommand = "uvicorn main:app --host 0.0.0.0 --port 8000"
		port = 8000
	}

	return &WebAppConfig{
		Framework:    framework,
		StartCommand: startCommand,
		Port:         port,
		HealthCheck:  "/",
		Timeout:      60 * time.Second,
	}
}

// analyzeJava analyzes Java web applications
func (a *Agent) analyzeJava(repoPath string) *WebAppConfig {
	// Check for Spring Boot or other Java web frameworks
	pomPath := filepath.Join(repoPath, "pom.xml")
	gradlePath := filepath.Join(repoPath, "build.gradle")

	var buildFile string
	if fileExists(pomPath) {
		buildFile = pomPath
	} else if fileExists(gradlePath) {
		buildFile = gradlePath
	} else {
		return nil
	}

	data, err := ioutil.ReadFile(buildFile)
	if err != nil {
		return nil
	}

	content := string(data)
	webFrameworks := []string{"spring-boot", "spring-web", "servlet-api", "jersey"}
	
	isWebApp := false
	for _, fw := range webFrameworks {
		if strings.Contains(strings.ToLower(content), fw) {
			isWebApp = true
			break
		}
	}

	if !isWebApp {
		return nil
	}

	startCommand := "java -jar target/*.jar"
	if strings.Contains(content, "spring-boot") {
		startCommand = "mvn spring-boot:run"
		if fileExists(gradlePath) {
			startCommand = "./gradlew bootRun"
		}
	}

	return &WebAppConfig{
		Framework:    "java-web",
		StartCommand: startCommand,
		Port:         8080,
		HealthCheck:  "/",
		Timeout:      90 * time.Second,
		BuildCommand: "mvn clean package -DskipTests",
	}
}

// analyzePHP analyzes PHP web applications
func (a *Agent) analyzePHP(repoPath string) *WebAppConfig {
	composerPath := filepath.Join(repoPath, "composer.json")
	data, err := ioutil.ReadFile(composerPath)
	if err != nil {
		return nil
	}

	var composer struct {
		Require map[string]string `json:"require"`
	}

	if err := json.Unmarshal(data, &composer); err != nil {
		return nil
	}

	// Check for PHP web frameworks
	webFrameworks := []string{"laravel/framework", "symfony/symfony", "slim/slim"}
	framework := "php"

	for _, fw := range webFrameworks {
		if _, exists := composer.Require[fw]; exists {
			framework = strings.Split(fw, "/")[0]
			break
		}
	}

	return &WebAppConfig{
		Framework:    framework,
		StartCommand: "php -S 0.0.0.0:8000 -t public",
		Port:         8000,
		HealthCheck:  "/",
		Timeout:      60 * time.Second,
	}
}

// analyzeRuby analyzes Ruby web applications
func (a *Agent) analyzeRuby(repoPath string) *WebAppConfig {
	gemfilePath := filepath.Join(repoPath, "Gemfile")
	data, err := ioutil.ReadFile(gemfilePath)
	if err != nil {
		return nil
	}

	content := string(data)
	webFrameworks := []string{"rails", "sinatra", "rack"}
	
	framework := ""
	for _, fw := range webFrameworks {
		if strings.Contains(strings.ToLower(content), fw) {
			framework = fw
			break
		}
	}

	if framework == "" {
		return nil
	}

	startCommand := "bundle exec ruby app.rb"
	if framework == "rails" {
		startCommand = "bundle exec rails server -b 0.0.0.0"
	}

	return &WebAppConfig{
		Framework:    framework,
		StartCommand: startCommand,
		Port:         3000,
		HealthCheck:  "/",
		Timeout:      60 * time.Second,
	}
}

// analyzeGo analyzes Go web applications
func (a *Agent) analyzeGo(repoPath string) *WebAppConfig {
	goModPath := filepath.Join(repoPath, "go.mod")
	data, err := ioutil.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	content := string(data)
	webFrameworks := []string{"gin-gonic/gin", "gorilla/mux", "echo", "fiber"}
	
	isWebApp := false
	for _, fw := range webFrameworks {
		if strings.Contains(content, fw) {
			isWebApp = true
			break
		}
	}

	if !isWebApp {
		return nil
	}

	return &WebAppConfig{
		Framework:    "go-web",
		StartCommand: "go run main.go",
		Port:         8080,
		HealthCheck:  "/",
		Timeout:      60 * time.Second,
	}
}

// analyzeDockerfile analyzes Dockerfile for web applications
func (a *Agent) analyzeDockerfile(repoPath string) *WebAppConfig {
	dockerfilePath := filepath.Join(repoPath, "Dockerfile")
	data, err := ioutil.ReadFile(dockerfilePath)
	if err != nil {
		return nil
	}

	content := strings.ToLower(string(data))
	
	// Look for EXPOSE directives
	lines := strings.Split(content, "\n")
	port := 8080 // Default port
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "expose ") {
			portStr := strings.TrimSpace(strings.TrimPrefix(line, "expose "))
			if p, err := strconv.Atoi(portStr); err == nil {
				port = p
				break
			}
		}
	}

	// Check if it looks like a web application
	webIndicators := []string{"nginx", "apache", "node", "python", "java", "php", "ruby", "go"}
	isWebApp := false
	
	for _, indicator := range webIndicators {
		if strings.Contains(content, indicator) {
			isWebApp = true
			break
		}
	}

	if !isWebApp {
		return nil
	}

	return &WebAppConfig{
		Framework:    "docker-web",
		StartCommand: "docker build -t webapp . && docker run -p " + strconv.Itoa(port) + ":" + strconv.Itoa(port) + " webapp",
		Port:         port,
		HealthCheck:  "/",
		Timeout:      120 * time.Second,
	}
}

// prepareRepository clones the repository to the specified path
func (a *Agent) prepareRepository(ctx context.Context, config agent.ScanConfig, repoPath string) error {
	var cmd *exec.Cmd
	if config.Commit != "" {
		cmd = exec.CommandContext(ctx, "git", "clone", "--depth", "1", config.RepoURL, repoPath)
	} else {
		branch := config.Branch
		if branch == "" {
			branch = "main"
		}
		cmd = exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, config.RepoURL, repoPath)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	if config.Commit != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "-C", repoPath, "checkout", config.Commit)
		if err := checkoutCmd.Run(); err != nil {
			return fmt.Errorf("git checkout failed: %w", err)
		}
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// startApplication starts the web application
func (a *Agent) startApplication(ctx context.Context, config agent.ScanConfig, webAppConfig *WebAppConfig) (*RunningApp, error) {
	// Create temporary directory for the application
	tempDir, err := os.MkdirTemp("", "zap-app-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Clone repository to temp directory
	repoPath := filepath.Join(tempDir, "repo")
	if err := a.prepareRepository(ctx, config, repoPath); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("failed to prepare repository: %w", err)
	}

	// Build application if needed
	if webAppConfig.BuildCommand != "" {
		buildCmd := exec.CommandContext(ctx, "sh", "-c", webAppConfig.BuildCommand)
		buildCmd.Dir = repoPath
		if err := buildCmd.Run(); err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("build command failed: %w", err)
		}
	}

	// Start application using Docker for isolation
	containerName := fmt.Sprintf("zap-app-%d", time.Now().Unix())
	
	// Create a simple Dockerfile if one doesn't exist
	if !fileExists(filepath.Join(repoPath, "Dockerfile")) {
		if err := a.createDockerfile(repoPath, webAppConfig); err != nil {
			os.RemoveAll(tempDir)
			return nil, fmt.Errorf("failed to create Dockerfile: %w", err)
		}
	}

	// Build Docker image
	buildCmd := exec.CommandContext(ctx, "docker", "build", "-t", containerName, ".")
	buildCmd.Dir = repoPath
	if err := buildCmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("docker build failed: %w", err)
	}

	// Run Docker container
	runCmd := exec.CommandContext(ctx, "docker", "run", "-d", "--name", containerName, 
		"-p", fmt.Sprintf("%d:%d", webAppConfig.Port, webAppConfig.Port), containerName)
	
	output, err := runCmd.Output()
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, fmt.Errorf("docker run failed: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	
	return &RunningApp{
		ContainerID: containerID,
		URL:         fmt.Sprintf("http://localhost:%d", webAppConfig.Port),
		Port:        webAppConfig.Port,
	}, nil
}

// createDockerfile creates a basic Dockerfile for the application
func (a *Agent) createDockerfile(repoPath string, webAppConfig *WebAppConfig) error {
	var dockerfile string
	
	switch webAppConfig.Framework {
	case "nodejs", "express", "next", "react-scripts":
		dockerfile = `FROM node:16-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE ` + strconv.Itoa(webAppConfig.Port) + `
CMD ["npm", "start"]`
	
	case "django", "flask", "fastapi":
		dockerfile = `FROM python:3.9-alpine
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE ` + strconv.Itoa(webAppConfig.Port) + `
CMD ["python", "app.py"]`
	
	case "java-web":
		dockerfile = `FROM openjdk:11-jre-slim
WORKDIR /app
COPY target/*.jar app.jar
EXPOSE ` + strconv.Itoa(webAppConfig.Port) + `
CMD ["java", "-jar", "app.jar"]`
	
	case "php", "laravel":
		dockerfile = `FROM php:8.0-apache
COPY . /var/www/html/
EXPOSE 80
CMD ["apache2-foreground"]`
	
	case "go-web":
		dockerfile = `FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE ` + strconv.Itoa(webAppConfig.Port) + `
CMD ["./main"]`
	
	default:
		dockerfile = `FROM alpine:latest
WORKDIR /app
COPY . .
EXPOSE ` + strconv.Itoa(webAppConfig.Port) + `
CMD ["sh", "-c", "` + webAppConfig.StartCommand + `"]`
	}

	return ioutil.WriteFile(filepath.Join(repoPath, "Dockerfile"), []byte(dockerfile), 0644)
}

// waitForApplication waits for the application to be ready
func (a *Agent) waitForApplication(ctx context.Context, app *RunningApp) error {
	client := &http.Client{Timeout: 5 * time.Second}
	
	// Wait up to 60 seconds for the application to start
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("application failed to start within timeout")
		case <-ticker.C:
			resp, err := client.Get(app.URL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil // Application is responding
				}
			}
		}
	}
}

// cleanupApplication stops and removes the application container
func (a *Agent) cleanupApplication(app *RunningApp) {
	if app.ContainerID != "" {
		// Stop container
		stopCmd := exec.Command("docker", "stop", app.ContainerID)
		stopCmd.Run()
		
		// Remove container
		rmCmd := exec.Command("docker", "rm", app.ContainerID)
		rmCmd.Run()
		
		// Remove image
		rmImageCmd := exec.Command("docker", "rmi", app.ContainerID)
		rmImageCmd.Run()
	}
}