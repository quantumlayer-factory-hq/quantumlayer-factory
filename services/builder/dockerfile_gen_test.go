package builder

import (
	"strings"
	"testing"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDockerfileGenerator(t *testing.T) {
	generator := NewDockerfileGenerator()
	assert.NotNil(t, generator)
	assert.NotNil(t, generator.templates)

	// Check that default templates are loaded
	templates := getDefaultTemplates()
	assert.NotEmpty(t, templates)
	assert.Equal(t, templates, generator.templates)
}

func TestDockerfileGenerator_Generate_PythonFastAPI(t *testing.T) {
	generator := NewDockerfileGenerator()

	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "test-fastapi",
		ImageTag:  "latest",
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)

	// Check Python-specific content
	assert.Contains(t, dockerfile, "FROM python:3.11-slim")
	assert.Contains(t, dockerfile, "WORKDIR /app")
	assert.Contains(t, dockerfile, "requirements.txt")
	assert.Contains(t, dockerfile, "pip install")
	assert.Contains(t, dockerfile, "EXPOSE 8000")
	assert.Contains(t, dockerfile, "USER appuser")
	assert.Contains(t, dockerfile, "uvicorn")
}

func TestDockerfileGenerator_Generate_GoGin(t *testing.T) {
	generator := NewDockerfileGenerator()

	req := &BuildRequest{
		Language:  "go",
		Framework: "gin",
		ImageName: "test-gin",
		ImageTag:  "latest",
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)

	// Check Go-specific content
	assert.Contains(t, dockerfile, "FROM golang:1.21-alpine as builder")
	assert.Contains(t, dockerfile, "FROM alpine:3.18") // Runtime stage
	assert.Contains(t, dockerfile, "go mod download")
	assert.Contains(t, dockerfile, "go build")
	assert.Contains(t, dockerfile, "EXPOSE 8080")
	assert.Contains(t, dockerfile, "USER appuser")
}

func TestDockerfileGenerator_Generate_NodeExpress(t *testing.T) {
	generator := NewDockerfileGenerator()

	req := &BuildRequest{
		Language:  "nodejs",
		Framework: "express",
		ImageName: "test-express",
		ImageTag:  "latest",
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)

	// Check Node.js-specific content
	assert.Contains(t, dockerfile, "FROM node:18-alpine")
	assert.Contains(t, dockerfile, "package*.json")
	assert.Contains(t, dockerfile, "npm ci")
	assert.Contains(t, dockerfile, "EXPOSE 3000")
	assert.Contains(t, dockerfile, "USER appuser")
}

func TestDockerfileGenerator_Generate_React(t *testing.T) {
	generator := NewDockerfileGenerator()

	req := &BuildRequest{
		Language:  "javascript",
		Framework: "react",
		ImageName: "test-react",
		ImageTag:  "latest",
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)
	assert.NotEmpty(t, dockerfile)

	// Check React-specific content (multi-stage build)
	assert.Contains(t, dockerfile, "FROM node:18-alpine as builder")
	assert.Contains(t, dockerfile, "npm run build")
	assert.Contains(t, dockerfile, "serve")
	assert.Contains(t, dockerfile, "EXPOSE 3000")
}

func TestDockerfileGenerator_Generate_UnsupportedLanguage(t *testing.T) {
	generator := NewDockerfileGenerator()

	req := &BuildRequest{
		Language:  "unsupported",
		Framework: "unknown",
		ImageName: "test-app",
		ImageTag:  "latest",
	}

	_, err := generator.Generate(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no template found")
}

func TestDockerfileGenerator_CustomizeTemplate(t *testing.T) {
	generator := NewDockerfileGenerator()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-api",
			Type: "api",
			Stack: ir.TechStack{
				Database: ir.DatabaseStack{
					Type: "postgresql",
				},
			},
		},
		API: ir.APISpec{
			Endpoints: []ir.Endpoint{
				{Path: "/health", Method: "GET"},
				{Path: "/users", Method: "GET"},
			},
		},
		Data: ir.DataSpec{
			Entities: []ir.Entity{
				{Name: "User", Fields: []ir.Field{{Name: "ID", Type: "int"}}},
			},
		},
	}

	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "test-api",
		ImageTag:  "latest",
		Spec:      spec,
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)

	// Check that customization based on spec was applied
	assert.Contains(t, dockerfile, "LABEL app.name=\"test-api\"")
	assert.Contains(t, dockerfile, "ENV APP_NAME=test-api")
	assert.Contains(t, dockerfile, "EXPOSE 8000") // FastAPI default port
}

func TestDockerfileGenerator_HealthCheck(t *testing.T) {
	templates := getDefaultTemplates()
	fastAPITemplate := templates["python-fastapi"]

	assert.NotNil(t, fastAPITemplate.HealthCheck)
	assert.Contains(t, fastAPITemplate.HealthCheck.Command, "curl")
	assert.Equal(t, 30*time.Second, fastAPITemplate.HealthCheck.Interval)
	assert.Equal(t, 10*time.Second, fastAPITemplate.HealthCheck.Timeout)
	assert.Equal(t, 3, fastAPITemplate.HealthCheck.Retries)
}

func TestDockerfileGenerator_SecurityBestPractices(t *testing.T) {
	generator := NewDockerfileGenerator()

	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "secure-app",
		ImageTag:  "latest",
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)

	// Check security best practices
	assert.Contains(t, dockerfile, "USER appuser", "Should use non-root user")
	assert.Contains(t, dockerfile, "addgroup", "Should create user group")
	assert.Contains(t, dockerfile, "chown", "Should change ownership")

	// Check environment variables for security
	assert.Contains(t, dockerfile, "PYTHONUNBUFFERED=1")
	assert.Contains(t, dockerfile, "PYTHONDONTWRITEBYTECODE=1")
}

func TestDockerfileGenerator_MultiStageBuilds(t *testing.T) {
	generator := NewDockerfileGenerator()

	// Test languages that should use multi-stage builds
	multiStageLanguages := []string{"go", "typescript"}

	for _, language := range multiStageLanguages {
		t.Run(language, func(t *testing.T) {
			req := &BuildRequest{
				Language:  language,
				Framework: getDefaultFramework(language),
				ImageName: "test-multistage",
				ImageTag:  "latest",
			}

			dockerfile, err := generator.Generate(req)
			require.NoError(t, err)

			// Should have build stage
			assert.Contains(t, dockerfile, "as builder")
			// Should have runtime stage
			assert.Regexp(t, `FROM .* as builder.*FROM .*`, strings.ReplaceAll(dockerfile, "\n", ""))
		})
	}
}

func TestDockerfileGenerator_SingleStageBuilds(t *testing.T) {
	generator := NewDockerfileGenerator()

	// Test languages that should use single-stage builds
	singleStageLanguages := []string{"python", "nodejs"}

	for _, language := range singleStageLanguages {
		t.Run(language, func(t *testing.T) {
			req := &BuildRequest{
				Language:  language,
				Framework: getDefaultFramework(language),
				ImageName: "test-singlestage",
				ImageTag:  "latest",
			}

			dockerfile, err := generator.Generate(req)
			require.NoError(t, err)

			// Should not have multi-stage build
			assert.NotContains(t, dockerfile, "as builder")
		})
	}
}

func TestDockerfileGenerator_GetDefaultPort(t *testing.T) {
	generator := NewDockerfileGenerator()

	tests := []struct {
		framework    string
		expectedPort int
	}{
		{"fastapi", 8000},
		{"django", 8000},
		{"flask", 5000},
		{"gin", 8080},
		{"express", 3000},
		{"react", 3000},
		{"vue", 3000},
		{"angular", 4200},
		{"unknown", 8080}, // default
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			port := generator.getDefaultPort(tt.framework)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestDockerfileGenerator_GetStartCommand(t *testing.T) {
	generator := NewDockerfileGenerator()

	tests := []struct {
		language  string
		framework string
		expected  string
	}{
		{"python", "fastapi", `["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]`},
		{"python", "django", `["python", "manage.py", "runserver", "0.0.0.0:8000"]`},
		{"python", "flask", `["python", "app.py"]`},
		{"go", "gin", `["./main"]`},
		{"nodejs", "express", `["node", "index.js"]`},
		{"typescript", "", `["node", "dist/index.js"]`},
	}

	for _, tt := range tests {
		t.Run(tt.language+"-"+tt.framework, func(t *testing.T) {
			cmd := generator.getStartCommand(tt.language, tt.framework)
			assert.Equal(t, tt.expected, cmd)
		})
	}
}

func TestDockerfileGenerator_DatabaseSupport(t *testing.T) {
	generator := NewDockerfileGenerator()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "db-app",
			Stack: ir.TechStack{
				Database: ir.DatabaseStack{
					Type: "postgresql",
				},
			},
		},
		Data: ir.DataSpec{
			Entities: []ir.Entity{
				{Name: "User", Fields: []ir.Field{{Name: "ID", Type: "int"}}},
			},
		},
	}

	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "db-app",
		ImageTag:  "latest",
		Spec:      spec,
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)

	// Should include PostgreSQL client library for Python
	assert.Contains(t, dockerfile, "psycopg2-binary")
}

func TestDockerfileGenerator_Labels(t *testing.T) {
	generator := NewDockerfileGenerator()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "labeled-app",
			Type: "api",
		},
	}

	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "labeled-app",
		ImageTag:  "v1.0.0",
		Spec:      spec,
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)

	// Check for proper labels
	assert.Contains(t, dockerfile, "LABEL app.name=\"labeled-app\"")
	assert.Contains(t, dockerfile, "LABEL app.type=\"api\"")
	assert.Contains(t, dockerfile, "LABEL language=\"python\"")
	assert.Contains(t, dockerfile, "LABEL framework=\"fastapi\"")
	assert.Contains(t, dockerfile, "LABEL generated.by=\"quantumlayer-factory\"")
}

func TestDockerfileGenerator_EnvironmentVariables(t *testing.T) {
	generator := NewDockerfileGenerator()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "env-app",
		},
	}

	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "env-app",
		ImageTag:  "latest",
		Spec:      spec,
	}

	dockerfile, err := generator.Generate(req)
	require.NoError(t, err)

	// Check for environment variables
	assert.Contains(t, dockerfile, "ENV APP_NAME=env-app")
	assert.Contains(t, dockerfile, "ENV APP_VERSION=1.0.0")
	assert.Contains(t, dockerfile, "ENV PYTHONUNBUFFERED=1")
}

func TestDockerfileGenerator_GetSupportedLanguages(t *testing.T) {
	supported := GetSupportedLanguages()

	expectedLanguages := []string{"python", "go", "nodejs", "javascript", "typescript"}
	for _, lang := range expectedLanguages {
		assert.Contains(t, supported, lang, "Should support language: %s", lang)
	}

	// Check Python frameworks
	pythonFrameworks := supported["python"]
	assert.Contains(t, pythonFrameworks, "fastapi")
	assert.Contains(t, pythonFrameworks, "django")
	assert.Contains(t, pythonFrameworks, "flask")

	// Check JavaScript frameworks
	jsFrameworks := supported["javascript"]
	assert.Contains(t, jsFrameworks, "react")
	assert.Contains(t, jsFrameworks, "vue")
}

func TestDockerfileGenerator_ErrorHandling(t *testing.T) {
	generator := NewDockerfileGenerator()

	// Test with nil request
	_, err := generator.Generate(nil)
	assert.Error(t, err)

	// Test with empty language
	req := &BuildRequest{
		Language:  "",
		Framework: "fastapi",
		ImageName: "test",
		ImageTag:  "latest",
	}
	_, err = generator.Generate(req)
	assert.Error(t, err)

	// Test with unsupported language/framework combination
	req = &BuildRequest{
		Language:  "cobol",
		Framework: "unsupported",
		ImageName: "test",
		ImageTag:  "latest",
	}
	_, err = generator.Generate(req)
	assert.Error(t, err)
}


// Benchmark tests

func BenchmarkDockerfileGeneration_Python(b *testing.B) {
	generator := NewDockerfileGenerator()
	req := &BuildRequest{
		Language:  "python",
		Framework: "fastapi",
		ImageName: "benchmark-app",
		ImageTag:  "latest",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDockerfileGeneration_Go(b *testing.B) {
	generator := NewDockerfileGenerator()
	req := &BuildRequest{
		Language:  "go",
		Framework: "gin",
		ImageName: "benchmark-app",
		ImageTag:  "latest",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.Generate(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper functions for tests

func getDefaultFramework(language string) string {
	frameworks := map[string]string{
		"python":     "fastapi",
		"go":         "gin",
		"nodejs":     "express",
		"javascript": "react",
		"typescript": "angular",
	}
	return frameworks[language]
}