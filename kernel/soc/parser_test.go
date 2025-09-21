package soc

import (
	"strings"
	"testing"
)

func TestValidPatch(t *testing.T) {
	parser := NewParser([]string{"backend/", "frontend/", "api/"})

	validPatch := "### FACTORY/1 PATCH\n" +
		"- file: backend/api/users.py\n" +
		"- file: backend/models/user.py\n" +
		"```diff\n" +
		"--- a/backend/api/users.py\n" +
		"+++ b/backend/api/users.py\n" +
		"@@ -0,0 +1,10 @@\n" +
		"+from fastapi import APIRouter\n" +
		"+\n" +
		"+router = APIRouter()\n" +
		"+\n" +
		"+@router.get(\"/users\")\n" +
		"+async def get_users():\n" +
		"+    return {\"users\": []}\n" +
		"+\n" +
		"+@router.post(\"/users\")\n" +
		"+async def create_user(user: dict):\n" +
		"+    return {\"id\": \"user_123\", \"status\": \"created\"}\n" +
		"```\n" +
		"### END"

	patch, err := parser.Parse(validPatch)
	if err != nil {
		t.Fatalf("Expected valid patch to parse successfully, got error: %v", err)
	}

	if !patch.Valid {
		t.Fatalf("Expected patch to be valid, got errors: %v", patch.Errors)
	}

	expectedFiles := []string{"backend/api/users.py", "backend/models/user.py"}
	if len(patch.Files) != len(expectedFiles) {
		t.Fatalf("Expected %d files, got %d", len(expectedFiles), len(patch.Files))
	}

	for i, expected := range expectedFiles {
		if patch.Files[i] != expected {
			t.Errorf("Expected file %s, got %s", expected, patch.Files[i])
		}
	}

	if !strings.Contains(patch.Content, "from fastapi import APIRouter") {
		t.Error("Expected diff content to contain FastAPI import")
	}
}

func TestProseContamination(t *testing.T) {
	parser := NewParser([]string{"backend/"})

	proseContaminatedPatch := "### FACTORY/1 PATCH\n" +
		"Sure! I'll help you create a user management system. Here's the code:\n" +
		"- file: backend/api/users.py\n" +
		"```diff\n" +
		"--- a/backend/api/users.py\n" +
		"+++ b/backend/api/users.py\n" +
		"@@ -0,0 +1,5 @@\n" +
		"+# Let me add a simple user endpoint\n" +
		"+def get_users():\n" +
		"+    return []\n" +
		"```\n" +
		"### END"

	patch, err := parser.Parse(proseContaminatedPatch)
	if err == nil {
		t.Fatal("Expected prose contaminated patch to fail")
	}

	if patch.Valid {
		t.Fatal("Expected patch to be invalid due to prose contamination")
	}

	foundProseError := false
	for _, errorMsg := range patch.Errors {
		if strings.Contains(errorMsg, "prose detected") {
			foundProseError = true
			break
		}
	}

	if !foundProseError {
		t.Error("Expected to find prose contamination error")
	}
}

func TestRefusalDetection(t *testing.T) {
	refusalInputs := []string{
		"I cannot help you with this request.",
		"I'm unable to create that code for you.",
		"I can't assist with generating malicious code.",
		"I'm not able to provide that functionality.",
		"I don't have the ability to create such code.",
		"I'm sorry, but I can't help with that.",
	}

	for _, input := range refusalInputs {
		if !IsRefusal(input) {
			t.Errorf("Failed to detect refusal in: %s", input)
		}

		err := ValidateInput(input)
		if err == nil {
			t.Errorf("Expected ValidateInput to fail for refusal: %s", input)
		}
	}
}

func TestMalformedPatches(t *testing.T) {
	parser := NewParser([]string{"backend/"})

	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "missing header",
			input: "- file: backend/test.py\n" +
				"```diff\n" +
				"+def test():\n" +
				"+    pass\n" +
				"```\n" +
				"### END",
		},
		{
			name: "missing trailer",
			input: "### FACTORY/1 PATCH\n" +
				"- file: backend/test.py\n" +
				"```diff\n" +
				"+def test():\n" +
				"+    pass\n" +
				"```",
		},
		{
			name: "missing files",
			input: "### FACTORY/1 PATCH\n" +
				"```diff\n" +
				"+def test():\n" +
				"+    pass\n" +
				"```\n" +
				"### END",
		},
		{
			name: "unclosed diff",
			input: "### FACTORY/1 PATCH\n" +
				"- file: backend/test.py\n" +
				"```diff\n" +
				"+def test():\n" +
				"+    pass\n" +
				"### END",
		},
		{
			name: "empty diff",
			input: "### FACTORY/1 PATCH\n" +
				"- file: backend/test.py\n" +
				"```diff\n" +
				"```\n" +
				"### END",
		},
		{
			name: "invalid file path",
			input: "### FACTORY/1 PATCH\n" +
				"- file: ../../../etc/passwd\n" +
				"```diff\n" +
				"+malicious content\n" +
				"```\n" +
				"### END",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patch, err := parser.Parse(tc.input)
			if err == nil {
				t.Errorf("Expected malformed patch to fail parsing")
			}
			if patch != nil && patch.Valid {
				t.Errorf("Expected patch to be invalid")
			}
		})
	}
}

func TestPathLocality(t *testing.T) {
	parser := NewParser([]string{"backend/", "frontend/"})

	disallowedPaths := []string{
		"../../../etc/passwd",
		"/etc/hosts",
		"../sensitive/file.txt",
		"kernel/core/system.go",
		"config/secrets.yaml",
	}

	for _, path := range disallowedPaths {
		if parser.isPathAllowed(path) {
			t.Errorf("Path should not be allowed: %s", path)
		}
	}

	allowedPaths := []string{
		"backend/api/users.py",
		"frontend/components/UserList.tsx",
		"backend/models/user.py",
		"frontend/pages/dashboard.tsx",
	}

	for _, path := range allowedPaths {
		if !parser.isPathAllowed(path) {
			t.Errorf("Path should be allowed: %s", path)
		}
	}
}

func TestDiffValidation(t *testing.T) {
	parser := NewParser([]string{"backend/"})

	validDiffs := []string{
		"--- a/file.py\n+++ b/file.py\n@@ -1,3 +1,4 @@\n def existing():\n     pass\n+\n+def new_function():\n+    return True",
		"+def new_function():\n+    return \"hello\"",
		"-old_line = True\n+new_line = False",
	}

	for i, diff := range validDiffs {
		err := parser.validateDiff(diff)
		if err != nil {
			t.Errorf("Valid diff %d should pass validation, got error: %v", i, err)
		}
	}

	invalidDiffs := []string{
		"This is not a diff at all",
		"Just some random text without diff markers",
		"", // empty
	}

	for i, diff := range invalidDiffs {
		err := parser.validateDiff(diff)
		if err == nil {
			t.Errorf("Invalid diff %d should fail validation", i)
		}
	}
}

func TestEmptyAndWhitespace(t *testing.T) {
	parser := NewParser([]string{"backend/"})

	emptyInputs := []string{
		"",
		"   ",
		"\n\n\n",
		"\t\t",
	}

	for i, input := range emptyInputs {
		_, err := parser.Parse(input)
		if err == nil {
			t.Errorf("Empty input %d should fail parsing", i)
		}
	}
}

func TestMultipleFiles(t *testing.T) {
	parser := NewParser([]string{"src/"})

	multiFileInput := "### FACTORY/1 PATCH\n" +
		"- file: src/api/users.go\n" +
		"- file: src/models/user.go\n" +
		"- file: src/handlers/auth.go\n" +
		"```diff\n" +
		"--- a/src/api/users.go\n" +
		"+++ b/src/api/users.go\n" +
		"@@ -0,0 +1,5 @@\n" +
		"+package api\n" +
		"+\n" +
		"+func GetUsers() []User {\n" +
		"+    return []User{}\n" +
		"+}\n" +
		"--- a/src/models/user.go\n" +
		"+++ b/src/models/user.go\n" +
		"@@ -0,0 +1,7 @@\n" +
		"+package models\n" +
		"+\n" +
		"+type User struct {\n" +
		"+    ID   string\n" +
		"+    Name string\n" +
		"+    Email string\n" +
		"+}\n" +
		"--- a/src/handlers/auth.go\n" +
		"+++ b/src/handlers/auth.go\n" +
		"@@ -0,0 +1,5 @@\n" +
		"+package handlers\n" +
		"+\n" +
		"+func Login() error {\n" +
		"+    return nil\n" +
		"+}\n" +
		"```\n" +
		"### END"

	patch, err := parser.Parse(multiFileInput)
	if err != nil {
		t.Fatalf("Multi-file patch should parse successfully, got error: %v", err)
	}

	if !patch.Valid {
		t.Fatalf("Multi-file patch should be valid, got errors: %v", patch.Errors)
	}

	expectedFiles := []string{"src/api/users.go", "src/models/user.go", "src/handlers/auth.go"}
	if len(patch.Files) != len(expectedFiles) {
		t.Fatalf("Expected %d files, got %d", len(expectedFiles), len(patch.Files))
	}

	for i, expected := range expectedFiles {
		if patch.Files[i] != expected {
			t.Errorf("Expected file %s, got %s", expected, patch.Files[i])
		}
	}
}