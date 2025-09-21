package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/soc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("QuantumLayer Factory CLI")
		fmt.Println("Usage: qlf <command> [args...]")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("  parse     - Parse SOC input from stdin")
		fmt.Println("  generate  - Generate code from brief (coming soon)")
		fmt.Println("  serve     - Start factory server (coming soon)")
		fmt.Println("  version   - Show version")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "parse":
		handleParse()
	case "version":
		fmt.Println("QuantumLayer Factory v0.1.0")
		fmt.Println("SOC Parser: Ready")
	case "generate":
		fmt.Println("🚧 Generate command coming soon!")
		fmt.Println("This will transform natural language briefs into deployable applications.")
	case "serve":
		fmt.Println("🚧 Server mode coming soon!")
		fmt.Println("This will start the factory API gateway.")
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}
}

func handleParse() {
	// Read from stdin
	input := ""
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input += scanner.Text() + "\n"
	}

	if input == "" {
		fmt.Println("Error: No input provided. Please pipe SOC content to stdin.")
		os.Exit(1)
	}

	// Create parser with some basic allowed paths
	parser := soc.NewParser([]string{"backend/", "frontend/", "api/", "src/"})

	// Parse the input
	patch, err := parser.Parse(input)
	if err != nil {
		fmt.Printf("❌ Parse failed: %v\n", err)
		if patch != nil {
			fmt.Printf("Errors found: %v\n", patch.Errors)
		}
		os.Exit(1)
	}

	fmt.Println("✅ Valid SOC patch detected!")
	fmt.Printf("Files: %v\n", patch.Files)
	fmt.Printf("Content length: %d characters\n", len(patch.Content))
	fmt.Println("Diff preview:")
	fmt.Println("---")
	lines := splitLines(patch.Content)
	for i, line := range lines {
		if i >= 10 {
			fmt.Printf("... (%d more lines)\n", len(lines)-10)
			break
		}
		fmt.Println(line)
	}
}

func splitLines(content string) []string {
	if content == "" {
		return []string{}
	}

	var lines []string
	current := ""

	for _, char := range content {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}