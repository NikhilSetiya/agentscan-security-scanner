package zap

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeNodeJS(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		name        string
		packageJSON string
		expected    *WebAppConfig
	}{
		{
			name: "Express.js application",
			packageJSON: `{
				"name": "my-app",
				"dependencies": {
					"express": "^4.18.0"
				},
				"scripts": {
					"start": "node server.js"
				}
			}`,
			expected: &WebAppConfig{
				Framework:    "express",
				StartCommand: "npm run start",
				Port:         3000,
				HealthCheck:  "/",
			},
		},
		{
			name: "Next.js application",
			packageJSON: `{
				"name": "my-next-app",
				"dependencies": {
					"next": "^12.0.0",
					"react": "^18.0.0"
				},
				"scripts": {
					"dev": "next dev",
					"build": "next build"
				}
			}`,
			expected: &WebAppConfig{
				Framework:    "next",
				StartCommand: "npm run dev",
				Port:         3000,
				HealthCheck:  "/",
				BuildCommand: "npm run build",
			},
		},
		{
			name: "Non-web Node.js application",
			packageJSON: `{
				"name": "cli-tool",
				"dependencies": {
					"commander": "^8.0.0"
				}
			}`,
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary package.json file
			tempDir, err := os.MkdirTemp("", "test-nodejs-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			packageJSONPath := filepath.Join(tempDir, "package.json")
			err = ioutil.WriteFile(packageJSONPath, []byte(tt.packageJSON), 0644)
			require.NoError(t, err)
			
			result := zapAgent.analyzeNodeJS(packageJSONPath)
			
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Framework, result.Framework)
				assert.Equal(t, tt.expected.StartCommand, result.StartCommand)
				assert.Equal(t, tt.expected.Port, result.Port)
				assert.Equal(t, tt.expected.HealthCheck, result.HealthCheck)
				if tt.expected.BuildCommand != "" {
					assert.Equal(t, tt.expected.BuildCommand, result.BuildCommand)
				}
			}
		})
	}
}

func TestAnalyzePython(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		name         string
		requirements string
		expected     *WebAppConfig
	}{
		{
			name: "Django application",
			requirements: `Django==4.0.0
psycopg2-binary==2.9.0`,
			expected: &WebAppConfig{
				Framework:    "django",
				StartCommand: "python manage.py runserver 0.0.0.0:8000",
				Port:         8000,
				HealthCheck:  "/",
			},
		},
		{
			name: "Flask application",
			requirements: `Flask==2.0.0
Werkzeug==2.0.0`,
			expected: &WebAppConfig{
				Framework:    "flask",
				StartCommand: "python app.py",
				Port:         5000,
				HealthCheck:  "/",
			},
		},
		{
			name: "FastAPI application",
			requirements: `fastapi==0.70.0
uvicorn==0.15.0`,
			expected: &WebAppConfig{
				Framework:    "fastapi",
				StartCommand: "uvicorn main:app --host 0.0.0.0 --port 8000",
				Port:         8000,
				HealthCheck:  "/",
			},
		},
		{
			name: "Non-web Python application",
			requirements: `requests==2.26.0
click==8.0.0`,
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary requirements.txt file
			tempDir, err := os.MkdirTemp("", "test-python-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			requirementsPath := filepath.Join(tempDir, "requirements.txt")
			err = ioutil.WriteFile(requirementsPath, []byte(tt.requirements), 0644)
			require.NoError(t, err)
			
			result := zapAgent.analyzePython(tempDir)
			
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Framework, result.Framework)
				assert.Equal(t, tt.expected.StartCommand, result.StartCommand)
				assert.Equal(t, tt.expected.Port, result.Port)
				assert.Equal(t, tt.expected.HealthCheck, result.HealthCheck)
			}
		})
	}
}

func TestAnalyzeJava(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		name     string
		pomXML   string
		expected *WebAppConfig
	}{
		{
			name: "Spring Boot application",
			pomXML: `<?xml version="1.0" encoding="UTF-8"?>
<project>
	<dependencies>
		<dependency>
			<groupId>org.springframework.boot</groupId>
			<artifactId>spring-boot-starter-web</artifactId>
		</dependency>
	</dependencies>
</project>`,
			expected: &WebAppConfig{
				Framework:    "java-web",
				StartCommand: "mvn spring-boot:run",
				Port:         8080,
				HealthCheck:  "/",
				BuildCommand: "mvn clean package -DskipTests",
			},
		},
		{
			name: "Non-web Java application",
			pomXML: `<?xml version="1.0" encoding="UTF-8"?>
<project>
	<dependencies>
		<dependency>
			<groupId>junit</groupId>
			<artifactId>junit</artifactId>
		</dependency>
	</dependencies>
</project>`,
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary pom.xml file
			tempDir, err := os.MkdirTemp("", "test-java-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			pomPath := filepath.Join(tempDir, "pom.xml")
			err = ioutil.WriteFile(pomPath, []byte(tt.pomXML), 0644)
			require.NoError(t, err)
			
			result := zapAgent.analyzeJava(tempDir)
			
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Framework, result.Framework)
				assert.Equal(t, tt.expected.StartCommand, result.StartCommand)
				assert.Equal(t, tt.expected.Port, result.Port)
				assert.Equal(t, tt.expected.HealthCheck, result.HealthCheck)
				assert.Equal(t, tt.expected.BuildCommand, result.BuildCommand)
			}
		})
	}
}

func TestAnalyzeDockerfile(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		name       string
		dockerfile string
		expected   *WebAppConfig
	}{
		{
			name: "Node.js Dockerfile with EXPOSE",
			dockerfile: `FROM node:16-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
EXPOSE 3000
CMD ["npm", "start"]`,
			expected: &WebAppConfig{
				Framework:    "docker-web",
				StartCommand: "docker build -t webapp . && docker run -p 3000:3000 webapp",
				Port:         3000,
				HealthCheck:  "/",
			},
		},
		{
			name: "Python Dockerfile with custom port",
			dockerfile: `FROM python:3.9-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["python", "app.py"]`,
			expected: &WebAppConfig{
				Framework:    "docker-web",
				StartCommand: "docker build -t webapp . && docker run -p 8080:8080 webapp",
				Port:         8080,
				HealthCheck:  "/",
			},
		},
		{
			name: "Non-web Dockerfile",
			dockerfile: `FROM alpine:latest
RUN apk add --no-cache curl
CMD ["echo", "hello world"]`,
			expected: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary Dockerfile
			tempDir, err := os.MkdirTemp("", "test-docker-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			dockerfilePath := filepath.Join(tempDir, "Dockerfile")
			err = ioutil.WriteFile(dockerfilePath, []byte(tt.dockerfile), 0644)
			require.NoError(t, err)
			
			result := zapAgent.analyzeDockerfile(tempDir)
			
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Framework, result.Framework)
				assert.Equal(t, tt.expected.Port, result.Port)
				assert.Equal(t, tt.expected.HealthCheck, result.HealthCheck)
				assert.Contains(t, result.StartCommand, "docker build")
				assert.Contains(t, result.StartCommand, "docker run")
			}
		})
	}
}

func TestCreateDockerfile(t *testing.T) {
	zapAgent := NewAgent()
	
	tests := []struct {
		name      string
		config    *WebAppConfig
		expected  []string // Strings that should be in the Dockerfile
	}{
		{
			name: "Node.js application",
			config: &WebAppConfig{
				Framework: "nodejs",
				Port:      3000,
			},
			expected: []string{"FROM node:16-alpine", "EXPOSE 3000", "\"npm\", \"start\""},
		},
		{
			name: "Python Django application",
			config: &WebAppConfig{
				Framework: "django",
				Port:      8000,
			},
			expected: []string{"FROM python:3.9-alpine", "EXPOSE 8000", "pip install"},
		},
		{
			name: "Java application",
			config: &WebAppConfig{
				Framework: "java-web",
				Port:      8080,
			},
			expected: []string{"FROM openjdk:11-jre-slim", "EXPOSE 8080", "\"java\", \"-jar\""},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := os.MkdirTemp("", "test-dockerfile-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			err = zapAgent.createDockerfile(tempDir, tt.config)
			require.NoError(t, err)
			
			// Read the created Dockerfile
			dockerfilePath := filepath.Join(tempDir, "Dockerfile")
			content, err := ioutil.ReadFile(dockerfilePath)
			require.NoError(t, err)
			
			dockerfileContent := string(content)
			for _, expected := range tt.expected {
				assert.Contains(t, dockerfileContent, expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tempDir, err := os.MkdirTemp("", "test-fileexists-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	existingFile := filepath.Join(tempDir, "existing.txt")
	err = ioutil.WriteFile(existingFile, []byte("test"), 0644)
	require.NoError(t, err)
	
	nonExistingFile := filepath.Join(tempDir, "nonexisting.txt")
	
	assert.True(t, fileExists(existingFile))
	assert.False(t, fileExists(nonExistingFile))
}