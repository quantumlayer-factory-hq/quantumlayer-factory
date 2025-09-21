package commands

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/overlays"
)

func NewOverlaysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "overlays",
		Short: "Manage and inspect overlays",
		Long:  "Commands for listing, describing, and managing domain and compliance overlays.",
	}

	cmd.AddCommand(NewOverlaysListCmd())
	cmd.AddCommand(NewOverlaysDescribeCmd())

	return cmd
}

func NewOverlaysListCmd() *cobra.Command {
	var flagType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available overlays",
		Long:  "List all available domain and compliance overlays in the system.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOverlaysList(flagType)
		},
	}

	cmd.Flags().StringVar(&flagType, "type", "", "Filter by overlay type (domain, compliance)")

	return cmd
}

func NewOverlaysDescribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe <overlay-name>",
		Short: "Describe a specific overlay",
		Long:  "Show detailed information about a specific overlay including features, validation rules, and prompt enhancements.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOverlaysDescribe(args[0])
		},
	}

	return cmd
}

func runOverlaysList(filterType string) error {
	// Use default overlay paths
	config := overlays.ResolverConfig{
		OverlayPaths: []string{"./overlays/domains", "./overlays/compliance"},
	}
	resolver := overlays.NewFileSystemResolver(config)

	metadata, err := resolver.ListAvailable()
	if err != nil {
		return fmt.Errorf("failed to list overlays: %w", err)
	}

	if len(metadata) == 0 {
		fmt.Println("No overlays found.")
		fmt.Println("Make sure overlay files are present in ./overlays/")
		return nil
	}

	// Group by type
	domainOverlays := []overlays.OverlayMetadata{}
	complianceOverlays := []overlays.OverlayMetadata{}

	for _, overlay := range metadata {
		switch overlay.Type {
		case overlays.OverlayTypeDomain:
			domainOverlays = append(domainOverlays, overlay)
		case overlays.OverlayTypeCompliance:
			complianceOverlays = append(complianceOverlays, overlay)
		}
	}

	// Display results
	fmt.Printf("📦 Available Overlays (%d total)\n\n", len(metadata))

	if filterType == "" || filterType == "domain" {
		if len(domainOverlays) > 0 {
			fmt.Println(color.BlueString("🏢 Domain Overlays:"))
			for _, overlay := range domainOverlays {
				fmt.Printf("  %s %s\n",
					color.New(color.Bold).Sprint(overlay.Name),
					color.New(color.Faint).Sprint("v"+overlay.Version))
				fmt.Printf("    %s\n", overlay.Description)
				if len(overlay.Tags) > 0 {
					fmt.Printf("    Tags: %s\n", color.New(color.Faint).Sprint(strings.Join(overlay.Tags, ", ")))
				}
				fmt.Println()
			}
		}
	}

	if filterType == "" || filterType == "compliance" {
		if len(complianceOverlays) > 0 {
			fmt.Println(color.GreenString("⚖️  Compliance Overlays:"))
			for _, overlay := range complianceOverlays {
				fmt.Printf("  %s %s\n",
					color.New(color.Bold).Sprint(overlay.Name),
					color.New(color.Faint).Sprint("v"+overlay.Version))
				fmt.Printf("    %s\n", overlay.Description)
				if len(overlay.Tags) > 0 {
					fmt.Printf("    Tags: %s\n", color.New(color.Faint).Sprint(strings.Join(overlay.Tags, ", ")))
				}
				fmt.Println()
			}
		}
	}

	// Usage examples
	fmt.Println(color.CyanString("💡 Usage Examples:"))
	if len(domainOverlays) > 0 {
		exampleDomain := domainOverlays[0].Name
		fmt.Printf("  # Use domain overlay:\n")
		fmt.Printf("  qlf generate \"your brief\" --overlay %s\n\n", exampleDomain)
	}
	if len(complianceOverlays) > 0 {
		exampleCompliance := complianceOverlays[0].Name
		fmt.Printf("  # Use compliance overlay:\n")
		fmt.Printf("  qlf generate \"your brief\" --overlay %s\n\n", exampleCompliance)
	}
	if len(domainOverlays) > 0 && len(complianceOverlays) > 0 {
		fmt.Printf("  # Combine multiple overlays:\n")
		fmt.Printf("  qlf generate \"your brief\" --overlay %s,%s\n",
			domainOverlays[0].Name, complianceOverlays[0].Name)
	}

	return nil
}

func runOverlaysDescribe(overlayName string) error {
	// Use default overlay paths
	config := overlays.ResolverConfig{
		OverlayPaths: []string{"./overlays/domains", "./overlays/compliance"},
	}
	resolver := overlays.NewFileSystemResolver(config)

	overlay, err := resolver.LoadOverlay(overlayName)
	if err != nil {
		return fmt.Errorf("overlay '%s' not found: %w", overlayName, err)
	}

	metadata := overlay.GetMetadata()

	// Header
	fmt.Printf("📦 %s %s\n",
		color.New(color.Bold).Sprint(metadata.Name),
		color.New(color.Faint).Sprint("v"+metadata.Version))
	fmt.Printf("Type: %s %s\n", getOverlayTypeIcon(string(metadata.Type)), metadata.Type)
	fmt.Printf("Priority: %d\n", metadata.Priority)
	fmt.Printf("Description: %s\n", metadata.Description)

	if metadata.Author != "" {
		fmt.Printf("Author: %s\n", metadata.Author)
	}

	if len(metadata.Tags) > 0 {
		fmt.Printf("Tags: %s\n", strings.Join(metadata.Tags, ", "))
	}

	fmt.Printf("Created: %s\n", metadata.CreatedAt.Format("2006-01-02"))
	fmt.Printf("Updated: %s\n\n", metadata.UpdatedAt.Format("2006-01-02"))

	// Dependencies
	dependencies := overlay.GetDependencies()
	if len(dependencies) > 0 {
		fmt.Println(color.YellowString("🔗 Dependencies:"))
		for _, dep := range dependencies {
			fmt.Printf("  - %s\n", dep)
		}
		fmt.Println()
	}

	// Prompt enhancements
	enhancements := overlay.GetPromptEnhancements()
	if len(enhancements) > 0 {
		fmt.Println(color.BlueString("✨ Prompt Enhancements:"))
		for _, enhancement := range enhancements {
			fmt.Printf("  %s Agent (%s, priority %d)\n",
				enhancement.AgentType, enhancement.Section, enhancement.Priority)
			fmt.Printf("    Position: %s\n", enhancement.Position)
			if len(enhancement.Conditions) > 0 {
				fmt.Printf("    Conditions: %v\n", enhancement.Conditions)
			}
			// Show preview of content
			preview := enhancement.Content
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Printf("    Preview: %s\n", color.New(color.Faint).Sprint(preview))
			fmt.Println()
		}
	}

	// Validation rules
	rules := overlay.GetValidationRules()
	if len(rules) > 0 {
		fmt.Println(color.RedString("🔍 Validation Rules:"))
		for _, rule := range rules {
			severityColor := color.GreenString
			if rule.Severity == "warning" {
				severityColor = color.YellowString
			} else if rule.Severity == "error" {
				severityColor = color.RedString
			}

			fmt.Printf("  %s (%s)\n", rule.Name, severityColor(rule.Severity))
			fmt.Printf("    Type: %s\n", rule.Type)
			fmt.Printf("    Message: %s\n", rule.Message)
			if rule.Remediation != "" {
				fmt.Printf("    Fix: %s\n", color.New(color.Faint).Sprint(rule.Remediation))
			}
			fmt.Println()
		}
	}

	// Usage examples
	fmt.Println(color.CyanString("💡 Usage Examples:"))
	fmt.Printf("  # Use this overlay:\n")
	fmt.Printf("  qlf generate \"your brief\" --overlay %s\n\n", overlayName)

	if len(dependencies) > 0 {
		allOverlays := append(dependencies, overlayName)
		fmt.Printf("  # Use with dependencies:\n")
		fmt.Printf("  qlf generate \"your brief\" --overlay %s\n", strings.Join(allOverlays, ","))
	}

	return nil
}

