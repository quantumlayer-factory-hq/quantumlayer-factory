package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.temporal.io/sdk/client"

	wf "github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/workflows"
)

var (
	flagDryRun  bool
	flagVerbose bool
	flagOutput  string
	flagAsync   bool
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

	if flagOutput == "" {
		flagOutput = cfg.OutputDir
	}
	flagVerbose = flagVerbose || cfg.Verbose

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