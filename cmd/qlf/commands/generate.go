package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/client"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	wf "github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/workflows"
)

var (
	flagDryRun         bool
	flagVerbose        bool
	flagOutput         string
	flagAsync          bool
	flagOverlays       []string
	flagSuggestOverlay bool
	flagProvider       string
	flagModel          string
	flagCompare        bool
	flagDeploy         string
	flagTTL            string
	flagSubdomain      string
	flagPort           int
	flagPackage        bool
)

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate [brief]",
		Short: "Generate an app from a natural-language brief",
		Long:  "Runs the QuantumLayer Factory workflow: Brief → IR → Generate → Verify → Package.",
		Args:  cobra.ArbitraryArgs,
		RunE:  runGenerate,
	}
	cmd.Flags().BoolVar(&flagDryRun, "dry-run", false, "Print results instead of writing files")
	cmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Verbose progress")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "Output directory (default from config)")
	cmd.Flags().BoolVar(&flagAsync, "async", false, "Return immediately with WorkflowID; use 'qlf status' to follow")
	cmd.Flags().StringSliceVar(&flagOverlays, "overlay", []string{}, "Apply specific overlays (e.g., --overlay fintech,pci)")
	cmd.Flags().BoolVar(&flagSuggestOverlay, "suggest-overlays", false, "Show suggested overlays without generating code")
	cmd.Flags().StringVar(&flagProvider, "provider", "", "LLM provider (bedrock, azure, auto) - defaults to QLF_LLM_PROVIDER env var")
	cmd.Flags().StringVar(&flagModel, "model", "", "Specific model to use (claude-sonnet, gpt-4, etc.) - defaults to QLF_LLM_MODEL env var")
	cmd.Flags().BoolVar(&flagCompare, "compare", false, "Generate with multiple providers for comparison")
	cmd.Flags().StringVar(&flagDeploy, "deploy", "", "Deploy type: 'preview' for ephemeral preview deployment")
	cmd.Flags().StringVar(&flagTTL, "ttl", "24h", "Time-to-live for preview deployment (e.g., 1h, 24h, 3d)")
	cmd.Flags().StringVar(&flagSubdomain, "subdomain", "", "Custom subdomain for preview (auto-generated if not specified)")
	cmd.Flags().IntVar(&flagPort, "port", 0, "Application port (auto-detected if not specified)")
	cmd.Flags().BoolVar(&flagPackage, "package", false, "Create .qlcapsule package after generation")
	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	brief := strings.TrimSpace(strings.Join(args, " "))
	if brief == "" {
		// fallback to stdin
		if stat, _ := os.Stdin.Stat(); (stat.Mode() & os.ModeCharDevice) == 0 {
			sc := bufio.NewScanner(os.Stdin)
			var b strings.Builder
			for sc.Scan() {
				b.WriteString(sc.Text())
				b.WriteByte('\n')
			}
			brief = strings.TrimSpace(b.String())
		}
	}
	if brief == "" {
		return fmt.Errorf("no brief provided (arg or stdin)")
	}

	// Handle overlay suggestion mode
	if flagSuggestOverlay {
		return showOverlaySuggestions(brief)
	}

	if flagOutput == "" {
		flagOutput = cfg.OutputDir
	}
	flagVerbose = flagVerbose || cfg.Verbose

	// Apply environment variable defaults for LLM configuration
	if flagProvider == "" {
		flagProvider = os.Getenv("QLF_LLM_PROVIDER")
	}
	if flagModel == "" {
		flagModel = os.Getenv("QLF_LLM_MODEL")
	}

	c, err := NewTemporalClient(cfg.Temporal.Address, cfg.Temporal.Namespace)
	if err != nil {
		return fmt.Errorf("temporal client: %w", err)
	}
	defer c.Close()

	// Generate unique project ID
	projectID := fmt.Sprintf("project-%d", time.Now().UnixNano())

	opts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("factory-%d", time.Now().UnixNano()),
		TaskQueue: cfg.Temporal.TaskQueue,
	}
	input := wf.FactoryInput{
		Brief:     brief,
		ProjectID: projectID,
		DryRun:    flagDryRun,
		Verbose:   flagVerbose,
		OutputDir: flagOutput,
		Overlays:  flagOverlays,
		Provider:  flagProvider,
		Model:     flagModel,
		Compare:   flagCompare,
	}

	we, err := c.ExecuteWorkflow(context.Background(), opts, wf.FactoryWorkflow, input)
	if err != nil {
		return fmt.Errorf("start workflow: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s Workflow started\n", green("✓"))
	fmt.Printf("  WorkflowID: %s\n  RunID:      %s\n", cyan(we.GetID()), cyan(we.GetRunID()))

	if flagVerbose {
		fmt.Printf("  Brief:      %s\n", brief)
		fmt.Printf("  ProjectID:  %s\n", projectID)
		fmt.Printf("  DryRun:     %v\n", flagDryRun)
	}

	if flagAsync {
		fmt.Println("Use `qlf status --id", we.GetID(), "` to check progress.")
		return nil
	}

	// Wait for completion
	fmt.Printf("%s Processing...\n", yellow("⏳"))

	var res wf.FactoryResult
	if err := we.Get(context.Background(), &res); err != nil {
		return fmt.Errorf("workflow failed: %w", err)
	}

	// Output handling
	fmt.Println()
	if flagDryRun {
		fmt.Println(cyan("=== DRY RUN RESULTS ==="))
		if res.SOCPatch != "" {
			fmt.Println(res.SOCPatch) // already SOC-formatted
		} else {
			fmt.Println(res.Summary)
		}
	} else {
		fmt.Printf("%s %s\n", green("✓"), res.Summary)
		if res.OutputPath != "" {
			fmt.Printf("  Output: %s\n", cyan(res.OutputPath))
		}
	}

	// Package if requested and not dry run (after files are written)
	if flagPackage && !flagDryRun && res.Success && res.OutputPath != "" {
		fmt.Printf("\n%s Creating package...\n", yellow("📦"))

		// Use the package command logic to create .qlcapsule
		packageName := "generated-app"
		if res.IRSpec != nil && res.IRSpec.App.Name != "" {
			packageName = res.IRSpec.App.Name
		}

		// Determine language and framework from IRSpec
		language := "unknown"
		framework := "unknown"
		if res.IRSpec != nil {
			if res.IRSpec.App.Stack.Backend.Language != "" {
				language = res.IRSpec.App.Stack.Backend.Language
			}
			if res.IRSpec.App.Stack.Backend.Framework != "" {
				framework = res.IRSpec.App.Stack.Backend.Framework
			}
		}

		// Call the package functionality directly
		err := runPackageGenerated(res.OutputPath, packageName, language, framework)
		if err != nil {
			fmt.Printf("%s Failed to create package: %v\n", color.RedString("✗"), err)
		} else {
			fmt.Printf("%s Package created successfully!\n", green("✓"))
			fmt.Printf("  Package: %s/%s.qlcapsule\n", cyan(res.OutputPath+"/packages"), packageName)
		}
	}

	if flagVerbose && res.IRSpec != nil {
		fmt.Printf("\nGenerated %s application:\n", res.IRSpec.App.Type)
		fmt.Printf("  Language: %s\n", res.IRSpec.App.Stack.Backend.Language)
		fmt.Printf("  Framework: %s\n", res.IRSpec.App.Stack.Backend.Framework)
		fmt.Printf("  Files: %d\n", len(res.GeneratedCode))
		fmt.Printf("  Duration: %v\n", res.Duration)
	}

	if len(res.Warnings) > 0 {
		fmt.Println(color.YellowString("\nWarnings:"))
		for _, w := range res.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	if len(res.Errors) > 0 {
		fmt.Println(color.RedString("\nErrors:"))
		for _, e := range res.Errors {
			fmt.Printf("  - %s\n", e)
		}
	}

	return nil
}

// showOverlaySuggestions analyzes the brief and shows suggested overlays
func showOverlaySuggestions(brief string) error {
	// Import here to avoid import cycles
	compiler := ir.NewCompiler()
	suggestions := compiler.SuggestOverlays(brief)

	if len(suggestions.Suggestions) == 0 {
		fmt.Println("No specific overlays detected for this brief.")
		fmt.Println("You can manually specify overlays using --overlay flag.")
		return nil
	}

	fmt.Printf("🔍 Overlay Analysis for: %s\n\n", color.CyanString(brief))

	// High-confidence suggestions (auto-apply)
	if len(suggestions.AutoApply) > 0 {
		fmt.Println(color.GreenString("✅ Recommended Overlays (high confidence):"))
		for _, overlay := range suggestions.AutoApply {
			fmt.Printf("  - %s\n", color.New(color.Bold).Sprint(overlay))
		}
		fmt.Println()
	}

	// All suggestions with details
	fmt.Println(color.BlueString("📋 Detected Overlays:"))
	for _, suggestion := range suggestions.Suggestions {
		confidenceColor := color.GreenString
		if suggestion.Confidence < 0.7 {
			confidenceColor = color.YellowString
		}
		if suggestion.Confidence < 0.5 {
			confidenceColor = color.RedString
		}

		fmt.Printf("  %s %s (%s)\n",
			getOverlayTypeIcon(suggestion.Type),
			color.New(color.Bold).Sprint(suggestion.Name),
			suggestion.Type)
		fmt.Printf("    Confidence: %s\n", confidenceColor("%.0f%%", suggestion.Confidence*100))
		fmt.Printf("    Reason: %s\n", suggestion.Reason)
		if len(suggestion.Keywords) > 0 {
			fmt.Printf("    Keywords: %s\n", color.New(color.Faint).Sprint(strings.Join(suggestion.Keywords, ", ")))
		}
		fmt.Println()
	}

	// Warnings
	if len(suggestions.Warnings) > 0 {
		fmt.Println(color.YellowString("⚠️  Warnings:"))
		for _, warning := range suggestions.Warnings {
			fmt.Printf("  - %s\n", warning)
		}
		fmt.Println()
	}

	// Usage examples
	fmt.Println(color.CyanString("💡 Usage Examples:"))

	if len(suggestions.AutoApply) > 0 {
		overlayList := strings.Join(suggestions.AutoApply, ",")
		fmt.Printf("  # Apply recommended overlays:\n")
		fmt.Printf("  qlf generate \"%s\" --overlay %s\n\n", brief, overlayList)
	}

	if len(suggestions.Suggestions) > 1 {
		allOverlays := make([]string, len(suggestions.Suggestions))
		for i, s := range suggestions.Suggestions {
			allOverlays[i] = s.Name
		}
		overlayList := strings.Join(allOverlays, ",")
		fmt.Printf("  # Apply all detected overlays:\n")
		fmt.Printf("  qlf generate \"%s\" --overlay %s\n\n", brief, overlayList)
	}

	fmt.Printf("  # Generate without overlays:\n")
	fmt.Printf("  qlf generate \"%s\"\n", brief)

	return nil
}

// getOverlayTypeIcon returns an icon for the overlay type
func getOverlayTypeIcon(overlayType string) string {
	switch overlayType {
	case "domain":
		return "🏢"
	case "compliance":
		return "⚖️"
	default:
		return "📦"
	}
}

// runPackageGenerated creates a .qlcapsule package from the generated code
func runPackageGenerated(sourcePath, packageName, language, framework string) error {
	// Get the path to the qlf binary
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	qlfBinary := filepath.Join(filepath.Dir(execPath), "qlf")

	// Convert to absolute path
	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Prepare the package command arguments
	args := []string{
		"package",
		"--source", absSourcePath,
		"--name", packageName,
		"--version", "1.0.0",
		"--description", "Generated application package",
		"--language", language,
		"--framework", framework,
		"--output-dir", filepath.Join(absSourcePath, "packages"),
	}

	// Execute the package command from current working directory
	cmd := exec.Command(qlfBinary, args...)

	// Capture output for better error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("package command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}