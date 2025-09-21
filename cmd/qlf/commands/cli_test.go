package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootExecutes(t *testing.T) {
	root := &cobra.Command{Use: "qlf"}
	root.AddCommand(&cobra.Command{
		Use: "noop",
		Run: func(*cobra.Command, []string) {},
	})
	root.SetArgs([]string{"noop"})

	var out bytes.Buffer
	root.SetOut(&out)

	err := root.Execute()
	require.NoError(t, err)
}

func TestConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/path")
	require.NoError(t, err)

	assert.Equal(t, "localhost:7233", cfg.Temporal.Address)
	assert.Equal(t, "default", cfg.Temporal.Namespace)
	assert.Equal(t, "factory-task-queue", cfg.Temporal.TaskQueue)
	assert.Equal(t, "./generated", cfg.OutputDir)
	assert.False(t, cfg.Verbose)
}

func TestGenerateCommandFlags(t *testing.T) {
	cmd := NewGenerateCmd()

	// Test flag parsing
	cmd.SetArgs([]string{"--dry-run", "--verbose", "--output", "/tmp/test", "test brief"})
	err := cmd.ParseFlags([]string{"--dry-run", "--verbose", "--output", "/tmp/test"})
	require.NoError(t, err)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")
	output, _ := cmd.Flags().GetString("output")

	assert.True(t, dryRun)
	assert.True(t, verbose)
	assert.Equal(t, "/tmp/test", output)
}

func TestStatusCommandFlags(t *testing.T) {
	cmd := NewStatusCmd()

	cmd.SetArgs([]string{"--id", "test-workflow-123", "--run", "test-run-456"})
	err := cmd.ParseFlags([]string{"--id", "test-workflow-123", "--run", "test-run-456"})
	require.NoError(t, err)

	id, _ := cmd.Flags().GetString("id")
	run, _ := cmd.Flags().GetString("run")

	assert.Equal(t, "test-workflow-123", id)
	assert.Equal(t, "test-run-456", run)
}

func TestConfigEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	t.Setenv("QLF_TEMPORAL_ADDR", "remote:7233")
	t.Setenv("QLF_OUTPUT_DIR", "/custom/output")

	cfg, err := LoadConfig("/nonexistent/path")
	require.NoError(t, err)

	assert.Equal(t, "remote:7233", cfg.Temporal.Address)
	assert.Equal(t, "/custom/output", cfg.OutputDir)
}