package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	flagID  string
	flagRun string
)

func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show workflow status",
		RunE:  runStatus,
	}
	cmd.Flags().StringVar(&flagID, "id", "", "WorkflowID")
	cmd.Flags().StringVar(&flagRun, "run", "", "RunID (optional)")
	return cmd
}

func runStatus(cmd *cobra.Command, _ []string) error {
	if flagID == "" {
		return fmt.Errorf("--id is required")
	}

	c, err := NewTemporalClient(cfg.Temporal.Address, cfg.Temporal.Namespace)
	if err != nil {
		return err
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	desc, err := c.DescribeWorkflowExecution(ctx, flagID, flagRun)
	if err != nil {
		return fmt.Errorf("failed to describe workflow: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("%s Workflow Status\n", green("📊"))
	fmt.Printf("  ID:     %s\n", cyan(flagID))
	if flagRun != "" {
		fmt.Printf("  Run:    %s\n", cyan(flagRun))
	}

	status := desc.WorkflowExecutionInfo.Status.String()
	switch status {
	case "WORKFLOW_EXECUTION_STATUS_RUNNING":
		fmt.Printf("  Status: %s\n", yellow(status))
	case "WORKFLOW_EXECUTION_STATUS_COMPLETED":
		fmt.Printf("  Status: %s\n", green(status))
	case "WORKFLOW_EXECUTION_STATUS_FAILED":
		fmt.Printf("  Status: %s\n", red(status))
	default:
		fmt.Printf("  Status: %s\n", status)
	}

	if startTime := desc.WorkflowExecutionInfo.StartTime; startTime != nil {
		fmt.Printf("  Start:  %s\n", startTime.AsTime().Format(time.RFC3339))
	}

	if closeTime := desc.WorkflowExecutionInfo.CloseTime; closeTime != nil {
		fmt.Printf("  End:    %s\n", closeTime.AsTime().Format(time.RFC3339))
		duration := closeTime.AsTime().Sub(desc.WorkflowExecutionInfo.StartTime.AsTime())
		fmt.Printf("  Duration: %v\n", duration)
	} else if desc.WorkflowExecutionInfo.StartTime != nil {
		elapsed := time.Since(desc.WorkflowExecutionInfo.StartTime.AsTime())
		fmt.Printf("  Elapsed: %v\n", elapsed)
	}

	return nil
}