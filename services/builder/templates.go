package builder

import (
	"time"
)

// getDefaultTemplates returns default Dockerfile templates for supported languages/frameworks
func getDefaultTemplates() map[string]*DockerfileTemplate {
	templates := make(map[string]*DockerfileTemplate)

	// Python FastAPI template
	templates["python-fastapi"] = &DockerfileTemplate{
		Language:  "python",
		Framework: "fastapi",
		BaseImage: "python:3.11-slim",
		WorkDir:   "/app",
		Dependencies: []string{
			"apt-get update && apt-get install -y --no-install-recommends gcc build-essential && rm -rf /var/lib/apt/lists/*",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"requirements.txt"}},
			{Type: "RUN", Command: "pip install --no-cache-dir --upgrade pip"},
			{Type: "RUN", Command: "pip install --no-cache-dir -r requirements.txt"},
		},
		ExposedPorts: []int{8000},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:8000/health", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"PYTHONUNBUFFERED": "1",
			"PYTHONDONTWRITEBYTECODE": "1",
		},
		User: "appuser",
	}

	// Python Django template
	templates["python-django"] = &DockerfileTemplate{
		Language:  "python",
		Framework: "django",
		BaseImage: "python:3.11-slim",
		WorkDir:   "/app",
		Dependencies: []string{
			"apt-get update && apt-get install -y --no-install-recommends gcc build-essential && rm -rf /var/lib/apt/lists/*",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"requirements.txt"}},
			{Type: "RUN", Command: "pip install --no-cache-dir --upgrade pip"},
			{Type: "RUN", Command: "pip install --no-cache-dir -r requirements.txt"},
			{Type: "RUN", Command: "python manage.py collectstatic --noinput"},
		},
		ExposedPorts: []int{8000},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:8000/health/", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"PYTHONUNBUFFERED": "1",
			"DJANGO_SETTINGS_MODULE": "myproject.settings",
		},
		User: "appuser",
	}

	// Python Flask template
	templates["python-flask"] = &DockerfileTemplate{
		Language:  "python",
		Framework: "flask",
		BaseImage: "python:3.11-slim",
		WorkDir:   "/app",
		Dependencies: []string{
			"apt-get update && apt-get install -y --no-install-recommends gcc build-essential && rm -rf /var/lib/apt/lists/*",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"requirements.txt"}},
			{Type: "RUN", Command: "pip install --no-cache-dir --upgrade pip"},
			{Type: "RUN", Command: "pip install --no-cache-dir -r requirements.txt"},
		},
		ExposedPorts: []int{5000},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:5000/health", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"PYTHONUNBUFFERED": "1",
			"FLASK_ENV": "production",
		},
		User: "appuser",
	}

	// Go Gin template
	templates["go-gin"] = &DockerfileTemplate{
		Language:  "go",
		Framework: "gin",
		BaseImage: "golang:1.21-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache git ca-certificates",
		},
		BuildSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"go.mod", "go.sum"}},
			{Type: "RUN", Command: "go mod download"},
			{Type: "COPY", Command: "/app/", Args: []string{"."}},
			{Type: "RUN", Command: "CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ."},
		},
		ExposedPorts: []int{8080},
		HealthCheck: &HealthCheck{
			Command:     []string{"wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"GIN_MODE": "release",
		},
		User: "appuser",
	}

	// Node.js Express template
	templates["nodejs-express"] = &DockerfileTemplate{
		Language:  "nodejs",
		Framework: "express",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci --only=production && npm cache clean --force"},
		},
		ExposedPorts: []int{3000},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:3000/health", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	// React template
	templates["javascript-react"] = &DockerfileTemplate{
		Language:  "javascript",
		Framework: "react",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		BuildSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci"},
			{Type: "COPY", Command: "/app/", Args: []string{"."}},
			{Type: "RUN", Command: "npm run build"},
		},
		RunSteps: []BuildStep{
			{Type: "RUN", Command: "npm install -g serve"},
		},
		ExposedPorts: []int{3000},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:3000", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	// Vue template
	templates["javascript-vue"] = &DockerfileTemplate{
		Language:  "javascript",
		Framework: "vue",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		BuildSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci"},
			{Type: "COPY", Command: "/app/", Args: []string{"."}},
			{Type: "RUN", Command: "npm run build"},
		},
		RunSteps: []BuildStep{
			{Type: "RUN", Command: "npm install -g serve"},
		},
		ExposedPorts: []int{3000},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:3000", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	// Angular template
	templates["typescript-angular"] = &DockerfileTemplate{
		Language:  "typescript",
		Framework: "angular",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		BuildSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci"},
			{Type: "COPY", Command: "/app/", Args: []string{"."}},
			{Type: "RUN", Command: "npm run build"},
		},
		RunSteps: []BuildStep{
			{Type: "RUN", Command: "npm install -g serve"},
		},
		ExposedPorts: []int{4200},
		HealthCheck: &HealthCheck{
			Command:     []string{"curl", "-f", "http://localhost:4200", "||", "exit", "1"},
			Interval:    30 * time.Second,
			Timeout:     10 * time.Second,
			StartPeriod: 40 * time.Second,
			Retries:     3,
		},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	// Generic Python template
	templates["python"] = &DockerfileTemplate{
		Language:  "python",
		Framework: "",
		BaseImage: "python:3.11-slim",
		WorkDir:   "/app",
		Dependencies: []string{
			"apt-get update && apt-get install -y --no-install-recommends gcc build-essential && rm -rf /var/lib/apt/lists/*",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"requirements.txt"}},
			{Type: "RUN", Command: "pip install --no-cache-dir --upgrade pip"},
			{Type: "RUN", Command: "pip install --no-cache-dir -r requirements.txt"},
		},
		ExposedPorts: []int{8000},
		Environment: map[string]string{
			"PYTHONUNBUFFERED": "1",
		},
		User: "appuser",
	}

	// Generic Go template
	templates["go"] = &DockerfileTemplate{
		Language:  "go",
		Framework: "",
		BaseImage: "golang:1.21-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache git ca-certificates",
		},
		BuildSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"go.mod", "go.sum"}},
			{Type: "RUN", Command: "go mod download"},
			{Type: "COPY", Command: "/app/", Args: []string{"."}},
			{Type: "RUN", Command: "CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ."},
		},
		ExposedPorts: []int{8080},
		Environment: map[string]string{
			"CGO_ENABLED": "0",
		},
		User: "appuser",
	}

	// Generic Node.js template
	templates["nodejs"] = &DockerfileTemplate{
		Language:  "nodejs",
		Framework: "",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci --only=production && npm cache clean --force"},
		},
		ExposedPorts: []int{3000},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	// Generic JavaScript template
	templates["javascript"] = &DockerfileTemplate{
		Language:  "javascript",
		Framework: "",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci --only=production && npm cache clean --force"},
		},
		ExposedPorts: []int{3000},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	// Generic TypeScript template
	templates["typescript"] = &DockerfileTemplate{
		Language:  "typescript",
		Framework: "",
		BaseImage: "node:18-alpine",
		WorkDir:   "/app",
		Dependencies: []string{
			"apk add --no-cache curl",
		},
		BuildSteps: []BuildStep{
			{Type: "COPY", Command: "/app/", Args: []string{"package*.json", "tsconfig.json"}},
			{Type: "RUN", Command: "npm ci"},
			{Type: "COPY", Command: "/app/", Args: []string{"."}},
			{Type: "RUN", Command: "npm run build"},
		},
		RunSteps: []BuildStep{
			{Type: "COPY", Command: "/app/node_modules/", Args: []string{"package*.json"}},
			{Type: "RUN", Command: "npm ci --only=production && npm cache clean --force"},
		},
		ExposedPorts: []int{3000},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		User: "appuser",
	}

	return templates
}

// GetSupportedLanguages returns a map of supported languages and their frameworks
func GetSupportedLanguages() map[string][]string {
	return map[string][]string{
		"python": {"fastapi", "django", "flask"},
		"go":     {"gin"},
		"nodejs": {"express"},
		"javascript": {"react", "vue"},
		"typescript": {"angular"},
		"java":   {"spring"},
		"csharp": {"aspnet"},
	}
}