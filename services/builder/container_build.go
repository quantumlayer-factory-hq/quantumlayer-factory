package builder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

// ContainerBuilder implements container building functionality
type ContainerBuilder struct {
	client    *client.Client
	generator *DockerfileGenerator
	scanner   Scanner
	config    *BuilderConfig
}

// NewContainerBuilder creates a new container builder
func NewContainerBuilder(config *BuilderConfig) (*ContainerBuilder, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(config.DockerHost),
		client.WithVersion(config.DockerVersion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	var scanner Scanner
	if config.EnableScanning {
		scanner, err = NewTrivyScanner()
		if err != nil {
			return nil, fmt.Errorf("failed to create security scanner: %w", err)
		}
	}

	return &ContainerBuilder{
		client:    cli,
		generator: NewDockerfileGenerator(),
		scanner:   scanner,
		config:    config,
	}, nil
}

// Build builds a container image from the project
func (cb *ContainerBuilder) Build(ctx context.Context, req *BuildRequest) (*BuildResult, error) {
	startTime := time.Now()

	result := &BuildResult{
		ImageName: req.ImageName,
		ImageTag:  req.ImageTag,
		BuildLogs: []string{},
		Warnings:  []string{},
		Errors:    []string{},
		Metadata:  make(map[string]interface{}),
	}

	// Validate project
	if err := cb.ValidateProject(req.ProjectPath); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Project validation failed: %v", err))
		return result, err
	}

	// Generate Dockerfile
	dockerfile, err := cb.GenerateDockerfile(req)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Dockerfile generation failed: %v", err))
		return result, err
	}
	result.Dockerfile = dockerfile

	// Create build context
	buildContext, err := cb.createBuildContext(req.ProjectPath, dockerfile)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Build context creation failed: %v", err))
		return result, err
	}
	defer buildContext.Close()

	// Build image
	imageID, logs, err := cb.buildImage(ctx, req, buildContext)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Image build failed: %v", err))
		result.BuildLogs = logs
		return result, err
	}

	result.ImageID = imageID
	result.BuildLogs = logs
	result.BuildTime = time.Since(startTime)

	// Get image size
	inspect, _, err := cb.client.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to get image size: %v", err))
	} else {
		result.ImageSize = inspect.Size
	}

	// Security scan if enabled
	if cb.config.EnableScanning && cb.scanner != nil {
		scanResult, err := cb.ScanImage(ctx, imageID)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Security scan failed: %v", err))
		} else {
			result.SecurityScan = scanResult
		}
	}

	// Push to registry if requested
	if req.PushToRegistry && req.Registry != "" {
		err = cb.PushImage(ctx, imageID, req.Registry, req.ImageName, req.ImageTag)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Registry push failed: %v", err))
		} else {
			result.RegistryURL = fmt.Sprintf("%s/%s:%s", req.Registry, req.ImageName, req.ImageTag)
		}
	}

	// Validate image size
	if result.ImageSize > cb.config.MaxImageSize {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Image size (%d bytes) exceeds maximum allowed size (%d bytes)",
				result.ImageSize, cb.config.MaxImageSize))
	}

	result.Success = len(result.Errors) == 0
	return result, nil
}

// GenerateDockerfile generates a Dockerfile for the project
func (cb *ContainerBuilder) GenerateDockerfile(req *BuildRequest) (string, error) {
	return cb.generator.Generate(req)
}

// ValidateProject validates that the project can be built
func (cb *ContainerBuilder) ValidateProject(projectPath string) error {
	// Check if project path exists
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return fmt.Errorf("project path does not exist: %s", projectPath)
	}

	// Check for common project files
	requiredFiles := map[string][]string{
		"python":     {"requirements.txt", "main.py", "app.py", "manage.py"},
		"go":         {"go.mod", "main.go"},
		"nodejs":     {"package.json"},
		"javascript": {"package.json"},
		"typescript": {"package.json", "tsconfig.json"},
	}

	// Try to detect language
	language := cb.detectLanguage(projectPath)
	if language == "" {
		return fmt.Errorf("unable to detect project language")
	}

	// Check for at least one required file
	if files, exists := requiredFiles[language]; exists {
		found := false
		for _, file := range files {
			if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no required files found for %s project", language)
		}
	}

	return nil
}

// GetSupportedLanguages returns supported languages and frameworks
func (cb *ContainerBuilder) GetSupportedLanguages() map[string][]string {
	return GetSupportedLanguages()
}

// ScanImage scans the built image for vulnerabilities
func (cb *ContainerBuilder) ScanImage(ctx context.Context, imageID string) (*SecurityScanResult, error) {
	if cb.scanner == nil {
		return nil, fmt.Errorf("security scanner not configured")
	}

	return cb.scanner.Scan(ctx, imageID, cb.config.DefaultThreshold)
}

// PushImage pushes the image to a registry
func (cb *ContainerBuilder) PushImage(ctx context.Context, imageID, registry, name, tag string) error {
	// Tag image for registry
	registryTag := fmt.Sprintf("%s/%s:%s", registry, name, tag)

	err := cb.client.ImageTag(ctx, imageID, registryTag)
	if err != nil {
		return fmt.Errorf("failed to tag image: %w", err)
	}

	// Push image
	pushOptions := image.PushOptions{}

	pushResponse, err := cb.client.ImagePush(ctx, registryTag, pushOptions)
	if err != nil {
		return fmt.Errorf("failed to push image: %w", err)
	}
	defer pushResponse.Close()

	// Read push response
	_, err = io.ReadAll(pushResponse)
	if err != nil {
		return fmt.Errorf("failed to read push response: %w", err)
	}

	return nil
}

// createBuildContext creates a build context with the Dockerfile
func (cb *ContainerBuilder) createBuildContext(projectPath, dockerfile string) (io.ReadCloser, error) {
	// Create a temporary directory for build context
	buildDir := filepath.Join(os.TempDir(), fmt.Sprintf("qlf-build-%d", time.Now().Unix()))

	// Copy project files to build directory
	err := cb.copyProjectFiles(projectPath, buildDir)
	if err != nil {
		return nil, fmt.Errorf("failed to copy project files: %w", err)
	}

	// Write Dockerfile to build directory
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	err = os.WriteFile(dockerfilePath, []byte(dockerfile), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Create tar archive
	return archive.TarWithOptions(buildDir, &archive.TarOptions{})
}

// copyProjectFiles copies project files to build directory
func (cb *ContainerBuilder) copyProjectFiles(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git and other unwanted directories
		if info.IsDir() {
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "__pycache__" ||
			   name == ".pytest_cache" || name == "target" || name == "dist" ||
			   strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy file
		return cb.copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func (cb *ContainerBuilder) copyFile(src, dst string) error {
	// Create destination directory
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	return err
}

// buildImage builds the Docker image
func (cb *ContainerBuilder) buildImage(ctx context.Context, req *BuildRequest, buildContext io.Reader) (string, []string, error) {
	imageName := fmt.Sprintf("%s:%s", req.ImageName, req.ImageTag)

	buildOptions := types.ImageBuildOptions{
		Tags:           []string{imageName},
		SuppressOutput: false,
		Remove:         true,
		ForceRemove:    true,
		PullParent:     true,
		BuildArgs:      convertBuildArgs(req.BuildArgs),
		Platform:       strings.Join(req.Platform, ","),
	}

	// Set build timeout
	buildCtx, cancel := context.WithTimeout(ctx, cb.config.BuildTimeout)
	defer cancel()

	response, err := cb.client.ImageBuild(buildCtx, buildContext, buildOptions)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start image build: %w", err)
	}
	defer response.Body.Close()

	// Parse build response
	var imageID string
	var logs []string

	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := scanner.Text()
		logs = append(logs, line)

		// Parse JSON response to get image ID
		var buildMsg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &buildMsg); err == nil {
			if aux, exists := buildMsg["aux"]; exists {
				if auxMap, ok := aux.(map[string]interface{}); ok {
					if id, exists := auxMap["ID"]; exists {
						if idStr, ok := id.(string); ok {
							imageID = idStr
						}
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", logs, fmt.Errorf("error reading build response: %w", err)
	}

	if imageID == "" {
		// Try to get image ID from Docker API
		images, err := cb.client.ImageList(ctx, image.ListOptions{})
		if err == nil {
			for _, img := range images {
				for _, tag := range img.RepoTags {
					if tag == imageName {
						imageID = img.ID
						break
					}
				}
				if imageID != "" {
					break
				}
			}
		}
	}

	if imageID == "" {
		return "", logs, fmt.Errorf("failed to get image ID after build")
	}

	return imageID, logs, nil
}

// detectLanguage detects the project language based on files
func (cb *ContainerBuilder) detectLanguage(projectPath string) string {
	// Check for language-specific files
	languageFiles := map[string][]string{
		"python":     {"requirements.txt", "setup.py", "pyproject.toml", "main.py", "app.py"},
		"go":         {"go.mod", "go.sum", "main.go"},
		"nodejs":     {"package.json"},
		"javascript": {"package.json"},
		"typescript": {"tsconfig.json"},
		"java":       {"pom.xml", "build.gradle"},
		"csharp":     {"*.csproj", "*.sln"},
	}

	for language, files := range languageFiles {
		for _, file := range files {
			if strings.Contains(file, "*") {
				// Handle glob patterns
				matches, _ := filepath.Glob(filepath.Join(projectPath, file))
				if len(matches) > 0 {
					return language
				}
			} else {
				if _, err := os.Stat(filepath.Join(projectPath, file)); err == nil {
					return language
				}
			}
		}
	}

	return ""
}

// convertBuildArgs converts map[string]string to map[string]*string for Docker API
func convertBuildArgs(args map[string]string) map[string]*string {
	result := make(map[string]*string)
	for k, v := range args {
		val := v
		result[k] = &val
	}
	return result
}