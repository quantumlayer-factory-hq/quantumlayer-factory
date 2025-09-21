package builder

import (
	"fmt"
	"strings"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// DockerfileGenerator generates Dockerfiles for different languages and frameworks
type DockerfileGenerator struct {
	templates map[string]*DockerfileTemplate
}

// NewDockerfileGenerator creates a new Dockerfile generator
func NewDockerfileGenerator() *DockerfileGenerator {
	return &DockerfileGenerator{
		templates: getDefaultTemplates(),
	}
}

// Generate generates a Dockerfile for the given build request
func (g *DockerfileGenerator) Generate(req *BuildRequest) (string, error) {
	if req == nil {
		return "", fmt.Errorf("build request cannot be nil")
	}

	templateKey := fmt.Sprintf("%s-%s", req.Language, req.Framework)

	// Try exact match first
	tmpl, exists := g.templates[templateKey]
	if !exists {
		// Try language-only match
		tmpl, exists = g.templates[req.Language]
		if !exists {
			return "", fmt.Errorf("no template found for language: %s, framework: %s", req.Language, req.Framework)
		}
	}

	// Customize template based on IR spec if available
	if req.Spec != nil {
		tmpl = g.customizeTemplate(tmpl, req)
	}

	return g.renderDockerfile(tmpl, req)
}

// customizeTemplate customizes the template based on IR specification
func (g *DockerfileGenerator) customizeTemplate(tmpl *DockerfileTemplate, req *BuildRequest) *DockerfileTemplate {
	// Clone template to avoid modifying original
	customTmpl := *tmpl

	spec := req.Spec

	// Set exposed ports from API spec
	if spec.API.Endpoints != nil && len(spec.API.Endpoints) > 0 {
		// Default port for different frameworks
		port := g.getDefaultPort(req.Framework)
		customTmpl.ExposedPorts = []int{port}
	}

	// Add database configuration if specified
	if spec.Data.Entities != nil && len(spec.Data.Entities) > 0 {
		g.addDatabaseSupport(&customTmpl, spec)
	}

	// Add environment variables from spec
	if customTmpl.Environment == nil {
		customTmpl.Environment = make(map[string]string)
	}

	customTmpl.Environment["APP_NAME"] = spec.App.Name
	customTmpl.Environment["APP_VERSION"] = "1.0.0"
	customTmpl.Environment["NODE_ENV"] = "production"
	customTmpl.Environment["PYTHONUNBUFFERED"] = "1"

	// Add labels
	if customTmpl.Labels == nil {
		customTmpl.Labels = make(map[string]string)
	}

	customTmpl.Labels["app.name"] = spec.App.Name
	customTmpl.Labels["app.type"] = spec.App.Type
	customTmpl.Labels["language"] = req.Language
	customTmpl.Labels["framework"] = req.Framework
	customTmpl.Labels["generated.by"] = "quantumlayer-factory"
	customTmpl.Labels["generated.at"] = time.Now().Format(time.RFC3339)

	return &customTmpl
}

// renderDockerfile renders the Dockerfile from template
func (g *DockerfileGenerator) renderDockerfile(tmpl *DockerfileTemplate, req *BuildRequest) (string, error) {
	var dockerfile strings.Builder

	// Multi-stage build for production optimizations
	if g.requiresMultiStage(tmpl) {
		dockerfile.WriteString(g.renderBuildStage(tmpl, req))
		dockerfile.WriteString("\n")
	}

	// Production stage
	dockerfile.WriteString(g.renderProductionStage(tmpl, req))

	return dockerfile.String(), nil
}

// renderBuildStage renders the build stage for multi-stage builds
func (g *DockerfileGenerator) renderBuildStage(tmpl *DockerfileTemplate, req *BuildRequest) string {
	var stage strings.Builder

	// Build stage
	stage.WriteString(fmt.Sprintf("FROM %s as builder\n", tmpl.BaseImage))
	stage.WriteString(fmt.Sprintf("WORKDIR %s\n", tmpl.WorkDir))

	// Install build dependencies
	for _, dep := range tmpl.Dependencies {
		stage.WriteString(fmt.Sprintf("RUN %s\n", dep))
	}

	// Copy source files
	stage.WriteString("COPY . .\n")

	// Execute build steps
	for _, step := range tmpl.BuildSteps {
		stage.WriteString(g.renderBuildStep(step))
	}

	return stage.String()
}

// renderProductionStage renders the production stage
func (g *DockerfileGenerator) renderProductionStage(tmpl *DockerfileTemplate, req *BuildRequest) string {
	var stage strings.Builder

	// Use multi-stage or single stage
	if g.requiresMultiStage(tmpl) {
		// Use minimal runtime image
		runtimeImage := g.getRuntimeImage(tmpl.Language)
		stage.WriteString(fmt.Sprintf("FROM %s\n", runtimeImage))
	} else {
		stage.WriteString(fmt.Sprintf("FROM %s\n", tmpl.BaseImage))
	}

	// Set working directory
	stage.WriteString(fmt.Sprintf("WORKDIR %s\n", tmpl.WorkDir))

	// Create non-root user for security
	if tmpl.User != "" {
		stage.WriteString(fmt.Sprintf("RUN addgroup -g 1001 -S %s && adduser -S %s -u 1001 -G %s\n",
			tmpl.User, tmpl.User, tmpl.User))
	}

	// Install runtime dependencies only
	runtimeDeps := g.getRuntimeDependencies(tmpl)
	for _, dep := range runtimeDeps {
		stage.WriteString(fmt.Sprintf("RUN %s\n", dep))
	}

	// Copy built artifacts or source
	if g.requiresMultiStage(tmpl) {
		// Copy from build stage
		buildArtifacts := g.getBuildArtifacts(tmpl.Language, tmpl.Framework)
		for src, dest := range buildArtifacts {
			stage.WriteString(fmt.Sprintf("COPY --from=builder %s %s\n", src, dest))
		}
	} else {
		stage.WriteString("COPY . .\n")
	}

	// Change ownership to non-root user
	if tmpl.User != "" {
		stage.WriteString(fmt.Sprintf("RUN chown -R %s:%s %s\n", tmpl.User, tmpl.User, tmpl.WorkDir))
		stage.WriteString(fmt.Sprintf("USER %s\n", tmpl.User))
	}

	// Set environment variables
	for key, value := range tmpl.Environment {
		stage.WriteString(fmt.Sprintf("ENV %s=%s\n", key, value))
	}

	// Add labels
	for key, value := range tmpl.Labels {
		stage.WriteString(fmt.Sprintf("LABEL %s=\"%s\"\n", key, value))
	}

	// Expose ports
	for _, port := range tmpl.ExposedPorts {
		stage.WriteString(fmt.Sprintf("EXPOSE %d\n", port))
	}

	// Health check
	if tmpl.HealthCheck != nil {
		stage.WriteString(g.renderHealthCheck(tmpl.HealthCheck))
	}

	// Execute run steps (install dependencies, etc.)
	for _, step := range tmpl.RunSteps {
		stage.WriteString(g.renderBuildStep(step))
	}

	// CMD instruction
	cmd := g.getStartCommand(tmpl.Language, tmpl.Framework)
	stage.WriteString(fmt.Sprintf("CMD %s\n", cmd))

	return stage.String()
}

// renderBuildStep renders a single build step
func (g *DockerfileGenerator) renderBuildStep(step BuildStep) string {
	switch step.Type {
	case "RUN":
		return fmt.Sprintf("RUN %s\n", step.Command)
	case "COPY":
		args := strings.Join(step.Args, " ")
		return fmt.Sprintf("COPY %s %s\n", args, step.Command)
	case "ADD":
		args := strings.Join(step.Args, " ")
		return fmt.Sprintf("ADD %s %s\n", args, step.Command)
	case "ENV":
		return fmt.Sprintf("ENV %s\n", step.Command)
	case "WORKDIR":
		return fmt.Sprintf("WORKDIR %s\n", step.Command)
	default:
		return fmt.Sprintf("%s %s\n", step.Type, step.Command)
	}
}

// renderHealthCheck renders health check instruction
func (g *DockerfileGenerator) renderHealthCheck(hc *HealthCheck) string {
	cmd := strings.Join(hc.Command, " ")
	return fmt.Sprintf("HEALTHCHECK --interval=%s --timeout=%s --start-period=%s --retries=%d CMD %s\n",
		hc.Interval, hc.Timeout, hc.StartPeriod, hc.Retries, cmd)
}

// Helper functions

func (g *DockerfileGenerator) requiresMultiStage(tmpl *DockerfileTemplate) bool {
	// Use multi-stage for languages that require compilation or build steps
	compiledLanguages := map[string]bool{
		"go":         true,
		"java":       true,
		"csharp":     true,
		"rust":       true,
		"typescript": true, // if building to JavaScript
	}

	// Use multi-stage for frontend frameworks that need build steps
	buildFrameworks := map[string]bool{
		"react":   true,
		"vue":     true,
		"angular": true,
	}

	return compiledLanguages[tmpl.Language] || buildFrameworks[tmpl.Framework]
}

func (g *DockerfileGenerator) getRuntimeImage(language string) string {
	runtimeImages := map[string]string{
		"go":         "alpine:3.18",
		"java":       "openjdk:17-jre-alpine",
		"csharp":     "mcr.microsoft.com/dotnet/aspnet:7.0-alpine",
		"rust":       "alpine:3.18",
		"typescript": "node:18-alpine",
		"python":     "python:3.11-slim",
	}

	if runtime, exists := runtimeImages[language]; exists {
		return runtime
	}
	return "alpine:3.18"
}

func (g *DockerfileGenerator) getRuntimeDependencies(tmpl *DockerfileTemplate) []string {
	// Filter out build-only dependencies
	var runtimeDeps []string

	for _, dep := range tmpl.Dependencies {
		if !strings.Contains(dep, "build-essential") &&
		   !strings.Contains(dep, "gcc") &&
		   !strings.Contains(dep, "make") {
			runtimeDeps = append(runtimeDeps, dep)
		}
	}

	return runtimeDeps
}

func (g *DockerfileGenerator) getBuildArtifacts(language, framework string) map[string]string {
	artifacts := make(map[string]string)

	switch language {
	case "go":
		artifacts["/app/main"] = "/app/main"
	case "java":
		artifacts["/app/target/*.jar"] = "/app/app.jar"
	case "csharp":
		artifacts["/app/bin/Release/net7.0/publish/"] = "/app/"
	case "typescript":
		artifacts["/app/dist/"] = "/app/dist/"
		artifacts["/app/package.json"] = "/app/package.json"
		artifacts["/app/node_modules/"] = "/app/node_modules/"
	}

	return artifacts
}

func (g *DockerfileGenerator) getStartCommand(language, framework string) string {
	commands := map[string]string{
		"python-fastapi": `["python", "-m", "uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]`,
		"python-django":  `["python", "manage.py", "runserver", "0.0.0.0:8000"]`,
		"python-flask":   `["python", "app.py"]`,
		"go-gin":         `["./main"]`,
		"go":             `["./main"]`,
		"nodejs-express": `["node", "index.js"]`,
		"typescript":     `["node", "dist/index.js"]`,
		"java":           `["java", "-jar", "app.jar"]`,
	}

	key := fmt.Sprintf("%s-%s", language, framework)
	if cmd, exists := commands[key]; exists {
		return cmd
	}

	if cmd, exists := commands[language]; exists {
		return cmd
	}

	return `["sh", "-c", "echo 'No start command defined'"]`
}

func (g *DockerfileGenerator) getDefaultPort(framework string) int {
	ports := map[string]int{
		"fastapi":  8000,
		"django":   8000,
		"flask":    5000,
		"gin":      8080,
		"express":  3000,
		"react":    3000,
		"vue":      3000,
		"angular":  4200,
	}

	if port, exists := ports[framework]; exists {
		return port
	}
	return 8080
}

func (g *DockerfileGenerator) addDatabaseSupport(tmpl *DockerfileTemplate, spec *ir.IRSpec) {
	// Add database client dependencies based on database type
	if spec.App.Stack.Database.Type != "" {
		switch tmpl.Language {
		case "python":
			g.addPythonDatabaseDeps(tmpl, spec.App.Stack.Database.Type)
		case "go":
			g.addGoDatabaseDeps(tmpl, spec.App.Stack.Database.Type)
		case "nodejs", "typescript":
			g.addNodeDatabaseDeps(tmpl, spec.App.Stack.Database.Type)
		}
	}
}

func (g *DockerfileGenerator) addPythonDatabaseDeps(tmpl *DockerfileTemplate, dbType string) {
	dbDeps := map[string]string{
		"postgresql": "psycopg2-binary",
		"mysql":      "mysql-connector-python",
		"mongodb":    "pymongo",
		"redis":      "redis",
	}

	if dep, exists := dbDeps[dbType]; exists {
		tmpl.RunSteps = append(tmpl.RunSteps, BuildStep{
			Type:    "RUN",
			Command: fmt.Sprintf("pip install %s", dep),
		})
	}
}

func (g *DockerfileGenerator) addGoDatabaseDeps(tmpl *DockerfileTemplate, dbType string) {
	// Go dependencies are handled via go.mod, so we just ensure
	// the database client libraries are available at runtime
	switch dbType {
	case "postgresql":
		tmpl.Environment["DB_DRIVER"] = "postgres"
	case "mysql":
		tmpl.Environment["DB_DRIVER"] = "mysql"
	case "mongodb":
		tmpl.Environment["DB_DRIVER"] = "mongodb"
	}
}

func (g *DockerfileGenerator) addNodeDatabaseDeps(tmpl *DockerfileTemplate, dbType string) {
	dbDeps := map[string]string{
		"postgresql": "pg",
		"mysql":      "mysql2",
		"mongodb":    "mongodb",
		"redis":      "redis",
	}

	if dep, exists := dbDeps[dbType]; exists {
		tmpl.RunSteps = append(tmpl.RunSteps, BuildStep{
			Type:    "RUN",
			Command: fmt.Sprintf("npm install %s", dep),
		})
	}
}