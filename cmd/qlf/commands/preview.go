package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/builder"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/deploy"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/preview"
)

func NewPreviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Manage preview deployments",
		Long:  "Create, manage, and monitor ephemeral preview deployments",
	}

	cmd.AddCommand(NewPreviewListCmd())
	cmd.AddCommand(NewPreviewCreateCmd())
	cmd.AddCommand(NewPreviewDeleteCmd())
	cmd.AddCommand(NewPreviewStatusCmd())
	cmd.AddCommand(NewPreviewLogsCmd())
	cmd.AddCommand(NewPreviewExtendCmd())

	return cmd
}

func NewPreviewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all preview deployments",
		RunE:  runPreviewList,
	}
}

func NewPreviewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [project-path]",
		Short: "Create a preview deployment",
		Args:  cobra.ExactArgs(1),
		RunE:  runPreviewCreate,
	}

	cmd.Flags().String("app-name", "", "Application name")
	cmd.Flags().String("language", "", "Programming language")
	cmd.Flags().String("framework", "", "Framework")
	cmd.Flags().Int("port", 0, "Application port")
	cmd.Flags().String("ttl", "24h", "Time-to-live (e.g., 1h, 24h, 3d)")
	cmd.Flags().String("subdomain", "", "Custom subdomain")
	cmd.Flags().Bool("tls", true, "Enable TLS")

	return cmd
}

func NewPreviewDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete [preview-id]",
		Short: "Delete a preview deployment",
		Args:  cobra.ExactArgs(1),
		RunE:  runPreviewDelete,
	}
}

func NewPreviewStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [preview-id]",
		Short: "Get preview deployment status",
		Args:  cobra.ExactArgs(1),
		RunE:  runPreviewStatus,
	}
}

func NewPreviewLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [preview-id]",
		Short: "Get preview deployment logs",
		Args:  cobra.ExactArgs(1),
		RunE:  runPreviewLogs,
	}

	cmd.Flags().Int("lines", 100, "Number of log lines to retrieve")
	cmd.Flags().Bool("follow", false, "Follow log output")

	return cmd
}

func NewPreviewExtendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extend [preview-id] [ttl]",
		Short: "Extend preview deployment TTL",
		Args:  cobra.ExactArgs(2),
		RunE:  runPreviewExtend,
	}

	return cmd
}

func runPreviewList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Initialize preview manager
	previewManager, err := createPreviewManager()
	if err != nil {
		return fmt.Errorf("failed to create preview manager: %w", err)
	}

	previews, err := previewManager.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list previews: %w", err)
	}

	if len(previews) == 0 {
		fmt.Println("No preview deployments found")
		return nil
	}

	// Print header
	fmt.Printf("%-12s %-20s %-10s %-50s %-20s\n", "ID", "APP NAME", "STATUS", "URL", "EXPIRES")
	fmt.Printf("%-12s %-20s %-10s %-50s %-20s\n", "──", "────────", "──────", "───", "───────")

	// Print previews
	for _, p := range previews {
		status := p.Status.Phase
		if status == "Running" {
			status = color.GreenString(status)
		} else if status == "Failed" {
			status = color.RedString(status)
		} else {
			status = color.YellowString(status)
		}

		url := p.URL
		if len(url) > 47 {
			url = url[:44] + "..."
		}

		expires := time.Until(p.ExpiresAt).Truncate(time.Minute).String()
		if time.Now().After(p.ExpiresAt) {
			expires = color.RedString("EXPIRED")
		}

		fmt.Printf("%-12s %-20s %-10s %-50s %-20s\n",
			p.ID[:12], p.AppName, status, url, expires)
	}

	return nil
}

func runPreviewCreate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	projectPath := args[0]

	// Get flags
	appName, _ := cmd.Flags().GetString("app-name")
	language, _ := cmd.Flags().GetString("language")
	framework, _ := cmd.Flags().GetString("framework")
	port, _ := cmd.Flags().GetInt("port")
	ttlStr, _ := cmd.Flags().GetString("ttl")
	subdomain, _ := cmd.Flags().GetString("subdomain")
	tls, _ := cmd.Flags().GetBool("tls")

	// Parse TTL
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return fmt.Errorf("invalid TTL format: %w", err)
	}

	// Auto-detect missing fields
	if appName == "" {
		appName = strings.TrimSuffix(projectPath, "/")
		if idx := strings.LastIndex(appName, "/"); idx != -1 {
			appName = appName[idx+1:]
		}
	}

	if language == "" {
		language = detectLanguage(projectPath)
	}

	if framework == "" {
		framework = detectFramework(projectPath, language)
	}

	if port == 0 {
		port = detectPort(framework)
	}

	// Initialize preview manager
	previewManager, err := createPreviewManager()
	if err != nil {
		return fmt.Errorf("failed to create preview manager: %w", err)
	}

	// Create preview request
	req := &preview.PreviewRequest{
		AppName:     appName,
		ProjectPath: projectPath,
		Language:    language,
		Framework:   framework,
		Port:        port,
		TTL:         ttl,
		Subdomain:   subdomain,
		TLS:         tls,
	}

	fmt.Printf("Creating preview deployment...\n")
	fmt.Printf("  App: %s\n", appName)
	fmt.Printf("  Language: %s\n", language)
	fmt.Printf("  Framework: %s\n", framework)
	fmt.Printf("  Port: %d\n", port)
	fmt.Printf("  TTL: %s\n", ttl)

	result, err := previewManager.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create preview: %w", err)
	}

	fmt.Printf("\n%s Preview deployment created!\n", color.GreenString("✓"))
	fmt.Printf("  ID: %s\n", result.ID)
	fmt.Printf("  Status: %s\n", result.Status.Phase)
	if result.URL != "" {
		fmt.Printf("  URL: %s\n", color.CyanString(result.URL))
	}
	fmt.Printf("  Expires: %s\n", result.ExpiresAt.Format(time.RFC3339))

	fmt.Printf("\nUse 'qlf preview status %s' to monitor deployment progress\n", result.ID)

	return nil
}

func runPreviewDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	previewID := args[0]

	previewManager, err := createPreviewManager()
	if err != nil {
		return fmt.Errorf("failed to create preview manager: %w", err)
	}

	// Get preview info before deletion
	result, err := previewManager.Get(ctx, previewID)
	if err != nil {
		return fmt.Errorf("failed to get preview: %w", err)
	}

	fmt.Printf("Deleting preview deployment %s (%s)...\n", previewID, result.AppName)

	err = previewManager.Delete(ctx, previewID)
	if err != nil {
		return fmt.Errorf("failed to delete preview: %w", err)
	}

	fmt.Printf("%s Preview deployment deleted successfully\n", color.GreenString("✓"))

	return nil
}

func runPreviewStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	previewID := args[0]

	previewManager, err := createPreviewManager()
	if err != nil {
		return fmt.Errorf("failed to create preview manager: %w", err)
	}

	result, err := previewManager.Get(ctx, previewID)
	if err != nil {
		return fmt.Errorf("failed to get preview: %w", err)
	}

	// Print detailed status
	fmt.Printf("Preview Deployment: %s\n", result.ID)
	fmt.Printf("App Name: %s\n", result.AppName)
	fmt.Printf("Status: %s\n", getColoredStatus(result.Status.Phase))
	fmt.Printf("Progress: %d%%\n", result.Status.Progress)
	if result.Status.Message != "" {
		fmt.Printf("Message: %s\n", result.Status.Message)
	}
	fmt.Printf("Created: %s\n", result.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", result.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("Expires: %s\n", result.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("TTL: %s\n", result.TTL)

	if result.URL != "" {
		fmt.Printf("URL: %s\n", color.CyanString(result.URL))
	}

	if result.InternalURL != "" {
		fmt.Printf("Internal URL: %s\n", result.InternalURL)
	}

	// Build info
	if result.BuildResult != nil {
		fmt.Printf("\nBuild Information:\n")
		fmt.Printf("  Image: %s:%s\n", result.BuildResult.ImageName, result.BuildResult.ImageTag)
		fmt.Printf("  Build Time: %s\n", result.BuildResult.BuildTime)
		fmt.Printf("  Image Size: %s\n", formatBytes(result.BuildResult.ImageSize))

		if result.BuildResult.SecurityScan != nil {
			sec := result.BuildResult.SecurityScan
			fmt.Printf("  Security Scan: %s\n", getColoredSecurityStatus(sec.Passed))
			fmt.Printf("    Scanner: %s\n", sec.Scanner)
			fmt.Printf("    Vulnerabilities: %d (Critical: %d, High: %d, Medium: %d, Low: %d)\n",
				sec.TotalVulns, sec.Critical, sec.High, sec.Medium, sec.Low)
		}
	}

	// Deploy info
	if result.DeployResult != nil {
		fmt.Printf("\nDeployment Information:\n")
		fmt.Printf("  Namespace: %s\n", result.DeployResult.Namespace)
		fmt.Printf("  Deployment: %s\n", result.DeployResult.DeploymentName)
		fmt.Printf("  Service: %s\n", result.DeployResult.ServiceName)
		if result.DeployResult.IngressName != "" {
			fmt.Printf("  Ingress: %s\n", result.DeployResult.IngressName)
		}
		fmt.Printf("  Replicas: %d/%d ready\n", result.DeployResult.ReadyReplicas, result.DeployResult.Replicas)
	}

	// Health check
	if result.HealthCheck != nil {
		fmt.Printf("\nHealth Check:\n")
		fmt.Printf("  Status: %s\n", getColoredHealthStatus(result.HealthCheck.Healthy))
		fmt.Printf("  Last Check: %s\n", result.HealthCheck.LastCheck.Format(time.RFC3339))
		fmt.Printf("  Response Time: %s\n", result.HealthCheck.ResponseTime)
		fmt.Printf("  Status Code: %d\n", result.HealthCheck.StatusCode)
		if result.HealthCheck.ErrorMessage != "" {
			fmt.Printf("  Error: %s\n", result.HealthCheck.ErrorMessage)
		}
	}

	// Analytics
	if result.Analytics != nil {
		fmt.Printf("\nAnalytics:\n")
		fmt.Printf("  Total Requests: %d\n", result.Analytics.TotalRequests)
		fmt.Printf("  Unique Visitors: %d\n", result.Analytics.UniqueVisitors)
		fmt.Printf("  Last Access: %s\n", result.Analytics.LastAccess.Format(time.RFC3339))
		fmt.Printf("  Avg Response Time: %s\n", result.Analytics.AvgResponseTime)
	}

	// Warnings and errors
	if len(result.Warnings) > 0 {
		fmt.Printf("\nWarnings:\n")
		for _, warning := range result.Warnings {
			fmt.Printf("  ⚠ %s\n", color.YellowString(warning))
		}
	}

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, err := range result.Errors {
			fmt.Printf("  ✗ %s\n", color.RedString(err))
		}
	}

	return nil
}

func runPreviewLogs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	previewID := args[0]

	lines, _ := cmd.Flags().GetInt("lines")
	follow, _ := cmd.Flags().GetBool("follow")

	previewManager, err := createPreviewManager()
	if err != nil {
		return fmt.Errorf("failed to create preview manager: %w", err)
	}

	logs, err := previewManager.GetLogs(ctx, previewID, lines)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	for _, log := range logs {
		fmt.Println(log)
	}

	if follow {
		// TODO: Implement log following
		fmt.Printf("Log following not yet implemented\n")
	}

	return nil
}

func runPreviewExtend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	previewID := args[0]
	ttlStr := args[1]

	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return fmt.Errorf("invalid TTL format: %w", err)
	}

	previewManager, err := createPreviewManager()
	if err != nil {
		return fmt.Errorf("failed to create preview manager: %w", err)
	}

	err = previewManager.Extend(ctx, previewID, ttl)
	if err != nil {
		return fmt.Errorf("failed to extend preview: %w", err)
	}

	fmt.Printf("%s Preview TTL extended to %s\n", color.GreenString("✓"), ttl)

	return nil
}

// Helper functions

func createPreviewManager() (*preview.PreviewManagerImpl, error) {
	// This would be properly configured from config in a real implementation
	config := preview.DefaultPreviewConfig()

	// Create builder
	builderConfig := builder.DefaultBuilderConfig()
	containerBuilder, err := builder.NewContainerBuilder(builderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create container builder: %w", err)
	}

	// Create deployer
	deployerConfig := deploy.DefaultDeployerConfig()
	deployer, err := deploy.NewKubernetesDeployer(deployerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployer: %w", err)
	}

	// Create URL manager
	urlManager := preview.NewInMemoryURLManager(config)

	// Create preview manager (simplified - in production would have all components)
	previewManager := preview.NewPreviewManager(
		config,
		containerBuilder,
		deployer,
		urlManager,
		nil, // TLS provider
		nil, // Health monitor
		nil, // Analytics
		NewInMemoryStorage(), // Storage
	)

	return previewManager, nil
}

func detectLanguage(projectPath string) string {
	// Simple language detection based on files
	if _, err := os.Stat(projectPath + "/requirements.txt"); err == nil {
		return "python"
	}
	if _, err := os.Stat(projectPath + "/go.mod"); err == nil {
		return "go"
	}
	if _, err := os.Stat(projectPath + "/package.json"); err == nil {
		return "nodejs"
	}
	if _, err := os.Stat(projectPath + "/tsconfig.json"); err == nil {
		return "typescript"
	}
	return "unknown"
}

func detectFramework(projectPath, language string) string {
	switch language {
	case "python":
		// Check for FastAPI, Django, Flask
		if checkFileContains(projectPath+"/requirements.txt", "fastapi") {
			return "fastapi"
		}
		if checkFileContains(projectPath+"/requirements.txt", "django") {
			return "django"
		}
		if checkFileContains(projectPath+"/requirements.txt", "flask") {
			return "flask"
		}
		return "fastapi" // default
	case "nodejs", "typescript":
		// Check package.json
		if checkFileContains(projectPath+"/package.json", "express") {
			return "express"
		}
		if checkFileContains(projectPath+"/package.json", "react") {
			return "react"
		}
		if checkFileContains(projectPath+"/package.json", "vue") {
			return "vue"
		}
		if checkFileContains(projectPath+"/package.json", "angular") {
			return "angular"
		}
		return "express" // default
	case "go":
		return "gin" // default
	}
	return "unknown"
}

func detectPort(framework string) int {
	ports := map[string]int{
		"fastapi":  8000,
		"django":   8000,
		"flask":    5000,
		"express":  3000,
		"react":    3000,
		"vue":      3000,
		"angular":  4200,
		"gin":      8080,
	}
	if port, exists := ports[framework]; exists {
		return port
	}
	return 8080
}

func checkFileContains(filePath, content string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), content)
}

func getColoredStatus(status string) string {
	switch status {
	case "Running":
		return color.GreenString(status)
	case "Failed":
		return color.RedString(status)
	case "Creating", "Building", "Deploying":
		return color.YellowString(status)
	default:
		return status
	}
}

func getColoredSecurityStatus(passed bool) string {
	if passed {
		return color.GreenString("PASSED")
	}
	return color.RedString("FAILED")
}

func getColoredHealthStatus(healthy bool) string {
	if healthy {
		return color.GreenString("HEALTHY")
	}
	return color.RedString("UNHEALTHY")
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Simple in-memory storage for demonstration
type InMemoryStorage struct {
	previews map[string]*preview.PreviewResult
}

func NewInMemoryStorage() preview.PreviewStorage {
	return &InMemoryStorage{
		previews: make(map[string]*preview.PreviewResult),
	}
}

func (s *InMemoryStorage) Store(ctx context.Context, result *preview.PreviewResult) error {
	s.previews[result.ID] = result
	return nil
}

func (s *InMemoryStorage) Get(ctx context.Context, id string) (*preview.PreviewResult, error) {
	if result, exists := s.previews[id]; exists {
		return result, nil
	}
	return nil, fmt.Errorf("preview not found")
}

func (s *InMemoryStorage) List(ctx context.Context) ([]*preview.PreviewResult, error) {
	var results []*preview.PreviewResult
	for _, result := range s.previews {
		results = append(results, result)
	}
	return results, nil
}

func (s *InMemoryStorage) Delete(ctx context.Context, id string) error {
	delete(s.previews, id)
	return nil
}