package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfg *Config

func Execute() {
	root := &cobra.Command{
		Use:   "qlf",
		Short: "QuantumLayer Factory CLI",
		Long:  "Transform natural language briefs into deployable applications using the QuantumLayer Factory.",
	}

	root.PersistentFlags().StringP("config", "c", "", "config file (default: $HOME/.qlf.yaml)")
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		path, _ := cmd.Flags().GetString("config")
		var err error
		cfg, err = LoadConfig(path)
		return err
	}

	root.AddCommand(NewGenerateCmd())
	root.AddCommand(NewStatusCmd())
	root.AddCommand(NewOverlaysCmd())
	root.AddCommand(NewPreviewCmd())
	root.AddCommand(NewPackageCmd())
	root.AddCommand(NewDeployCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}