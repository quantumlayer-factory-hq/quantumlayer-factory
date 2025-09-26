package soc

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// ABNF Grammar for Strict Output Contract:
//
// response      = header newline filelist newline patchblock newline trailer
// header        = "### FACTORY/1 PATCH"
// filelist      = 1*( "- file: " filepath newline )
// patchblock    = "```diff" newline 1*udiff newline "```"
// trailer       = "### END"
// filepath      = 1*( ALPHA / DIGIT / "/" / "." / "_" / "-" )

type Patch struct {
	Files   []string `json:"files"`
	Content string   `json:"content"`
	Valid   bool     `json:"valid"`
	Errors  []string `json:"errors"`
}

type Parser struct {
	allowedPaths []string
}

func NewParser(allowedPaths []string) *Parser {
	return &Parser{
		allowedPaths: allowedPaths,
	}
}

var (
	headerRegex      = regexp.MustCompile("^### FACTORY/1 PATCH\\s*$")
	trailerRegex     = regexp.MustCompile("^### END\\s*$")
	fileRegex        = regexp.MustCompile("^- file: ([a-zA-Z0-9/_.-]+)\\s*$")
	diffStartRegex   = regexp.MustCompile("^```diff\\s*$")
	diffEndRegex     = regexp.MustCompile("^```\\s*$")
	diffHeaderRegex  = regexp.MustCompile("^(---|\\+\\+\\+)\\s+[ab]/.*")
	proseRegex       = regexp.MustCompile("(?i)^\\s*(here's|let me|i'll|sure[,.]|of course|i can|based on [^\\+\\-]*|this will|please[,.]|sorry[,.])")
)

func (p *Parser) Parse(input string) (*Patch, error) {
	patch := &Patch{
		Files:  []string{},
		Errors: []string{},
	}

	lines := strings.Split(strings.TrimSpace(input), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	// Check for prose contamination first
	for i, line := range lines {
		if proseRegex.MatchString(line) {
			patch.Errors = append(patch.Errors, fmt.Sprintf("prose detected at line %d: %s", i+1, line))
		}
	}

	// Parse structure
	state := "expecting_header"
	diffContent := strings.Builder{}
	inDiff := false

	for i, line := range lines {
		lineNum := i + 1
		line = strings.TrimSpace(line)

		switch state {
		case "expecting_header":
			if headerRegex.MatchString(line) {
				state = "expecting_files"
			} else if line == "" {
				continue // skip empty lines
			} else {
				patch.Errors = append(patch.Errors, fmt.Sprintf("invalid header at line %d, expected '### FACTORY/1 PATCH'", lineNum))
				return patch, fmt.Errorf("invalid header")
			}

		case "expecting_files":
			if fileRegex.MatchString(line) {
				matches := fileRegex.FindStringSubmatch(line)
				filepath := matches[1]

				// Validate file path locality
				if !p.isPathAllowed(filepath) {
					patch.Errors = append(patch.Errors, fmt.Sprintf("file path not allowed: %s", filepath))
				}

				patch.Files = append(patch.Files, filepath)
			} else if diffStartRegex.MatchString(line) {
				state = "in_diff"
				inDiff = true
			} else if diffHeaderRegex.MatchString(line) {
				// Handle raw unified diff format but extract all files first
				// Parse the entire content to find all file references
				allFiles := p.extractAllFilesFromContent(input)
				for _, file := range allFiles {
					if !contains(patch.Files, file) {
						patch.Files = append(patch.Files, file)
					}
				}
				state = "in_raw_diff"
				diffContent.WriteString(line + "\n")
			} else if line == "" {
				continue // skip empty lines
			} else {
				patch.Errors = append(patch.Errors, fmt.Sprintf("expected file list or diff start at line %d", lineNum))
			}

		case "in_diff":
			if diffEndRegex.MatchString(line) {
				state = "expecting_trailer"
				inDiff = false
			} else if trailerRegex.MatchString(line) {
				// Handle case where ### END appears directly in diff block
				state = "done"
				inDiff = false
			} else {
				diffContent.WriteString(line + "\n")
			}

		case "in_raw_diff":
			if trailerRegex.MatchString(line) {
				state = "done"
			} else {
				diffContent.WriteString(line + "\n")
			}

		case "expecting_trailer":
			if trailerRegex.MatchString(line) {
				state = "done"
			} else if line == "" {
				continue // skip empty lines
			} else {
				patch.Errors = append(patch.Errors, fmt.Sprintf("expected trailer '### END' at line %d", lineNum))
			}
		}
	}

	// Validate final state
	if state != "done" {
		patch.Errors = append(patch.Errors, fmt.Sprintf("incomplete patch, ended in state: %s", state))
	}

	if inDiff {
		patch.Errors = append(patch.Errors, "unclosed diff block")
	}

	if len(patch.Files) == 0 {
		patch.Errors = append(patch.Errors, "no files specified")
	}

	// Validate diff content
	content := strings.TrimSpace(diffContent.String())
	if content == "" {
		patch.Errors = append(patch.Errors, "empty diff content")
	} else {
		patch.Content = content
		if err := p.validateDiff(content); err != nil {
			patch.Errors = append(patch.Errors, fmt.Sprintf("invalid diff: %v", err))
		}
	}

	patch.Valid = len(patch.Errors) == 0

	if !patch.Valid {
		return patch, fmt.Errorf("patch validation failed: %s", strings.Join(patch.Errors, "; "))
	}

	return patch, nil
}

func (p *Parser) isPathAllowed(path string) bool {
	if len(p.allowedPaths) == 0 {
		return true // no restrictions
	}

	for _, allowed := range p.allowedPaths {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}
	return false
}

func (p *Parser) validateDiff(content string) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	hasValidLine := false
	hasFileHeader := false

	diffHeaderRegex := regexp.MustCompile("^(---|\\+\\+\\+|@@|\\+|-| ).*")

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Check for file headers to identify diff sections
		if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "+++ b/") {
			hasFileHeader = true
			hasValidLine = true
			continue
		}

		// Allow lines in unified diff format OR raw code lines (for flexibility)
		if diffHeaderRegex.MatchString(line) {
			hasValidLine = true
		} else if hasFileHeader {
			// Allow raw code lines after file headers (LLM may not use + prefix consistently)
			hasValidLine = true
		} else {
			return fmt.Errorf("invalid diff line: %s", line)
		}
	}

	if !hasValidLine {
		return fmt.Errorf("no valid diff lines found")
	}

	return nil
}

// IsRefusal checks if the input contains common LLM refusal patterns
func IsRefusal(input string) bool {
	refusalPatterns := []string{
		"I cannot",
		"I'm unable to",
		"I can't",
		"I'm not able to",
		"I don't have the ability",
		"I'm not programmed to",
		"I'm sorry, but I can't",
		"I cannot assist with",
		"I'm not allowed to",
		"I cannot provide",
		"I'm not capable of",
		"I cannot help with",
		"I won't be able to",
		"I'm unable to assist",
		"I cannot complete",
		"I'm not permitted to",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range refusalPatterns {
		if strings.Contains(lowerInput, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// extractAllFilesFromContent scans the entire content to find all file references
func (p *Parser) extractAllFilesFromContent(content string) []string {
	var files []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for file references in both file list and diff headers
		if fileRegex.MatchString(line) {
			matches := fileRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				files = append(files, matches[1])
			}
		} else if strings.HasPrefix(line, "--- a/") {
			filepath := strings.TrimPrefix(line, "--- a/")
			files = append(files, filepath)
		}
	}

	return files
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidateInput performs basic validation before parsing
func ValidateInput(input string) error {
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("empty input")
	}

	if IsRefusal(input) {
		return fmt.Errorf("refusal detected in input")
	}

	if !strings.Contains(input, "### FACTORY/1 PATCH") {
		return fmt.Errorf("missing required header")
	}

	if !strings.Contains(input, "### END") {
		return fmt.Errorf("missing required trailer")
	}

	return nil
}