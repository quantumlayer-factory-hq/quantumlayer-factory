package ir

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompiler_Basic(t *testing.T) {
	compiler := NewCompiler()

	result, err := compiler.Compile("Create a user management system")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if result.Spec == nil {
		t.Fatal("Expected spec to be generated")
	}

	// Check basic fields
	if result.Spec.Brief != "Create a user management system" {
		t.Errorf("Expected brief to be preserved, got: %s", result.Spec.Brief)
	}

	if result.Spec.Version != "1.0" {
		t.Errorf("Expected version 1.0, got: %s", result.Spec.Version)
	}

	// Should detect user entity
	found := false
	for _, entity := range result.Spec.Data.Entities {
		if entity.Name == "User" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected User entity to be detected")
	}
}

func TestCompiler_EcommerceBrief(t *testing.T) {
	compiler := NewCompiler()

	brief := "Create an ecommerce platform with user authentication, product catalog, shopping cart, and payment processing"

	result, err := compiler.Compile(brief)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	spec := result.Spec

	// Check domain detection
	if spec.App.Domain != "ecommerce" {
		t.Errorf("Expected ecommerce domain, got: %s", spec.App.Domain)
	}

	// Check feature detection
	expectedFeatures := []string{"User Authentication", "CRUD Operations", "Payment Processing"}
	for _, expected := range expectedFeatures {
		found := false
		for _, feature := range spec.App.Features {
			if feature.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected feature '%s' not found", expected)
		}
	}

	// Check entities
	expectedEntities := []string{"User", "Product", "Order"}
	for _, expected := range expectedEntities {
		found := false
		for _, entity := range spec.Data.Entities {
			if entity.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected entity '%s' not found", expected)
		}
	}

	// Check API endpoints generation
	if len(spec.API.Endpoints) == 0 {
		t.Error("Expected API endpoints to be generated")
	}

	// Should have CRUD endpoints for each entity
	expectedEndpointCount := len(expectedEntities) * 5 // GET list, GET by ID, POST, PUT, DELETE
	if len(spec.API.Endpoints) < expectedEndpointCount {
		t.Errorf("Expected at least %d endpoints, got %d", expectedEndpointCount, len(spec.API.Endpoints))
	}
}

func TestCompiler_TechStackDetection(t *testing.T) {
	testCases := []struct {
		brief    string
		expected map[string]string
	}{
		{
			brief: "Create a FastAPI application with PostgreSQL database",
			expected: map[string]string{
				"backend_language":  "python",
				"backend_framework": "fastapi",
				"database":          "postgresql",
			},
		},
		{
			brief: "Build a Go microservice with Gin framework and Redis cache",
			expected: map[string]string{
				"backend_language":  "go",
				"backend_framework": "gin",
				"database":          "postgresql", // default
				"cache":             "redis",
			},
		},
		{
			brief: "Create a React TypeScript application with Node.js Express backend",
			expected: map[string]string{
				"backend_language":   "nodejs",
				"backend_framework":  "express",
				"frontend_language":  "typescript",
				"frontend_framework": "react",
			},
		},
	}

	compiler := NewCompiler()

	for _, tc := range testCases {
		t.Run(tc.brief, func(t *testing.T) {
			result, err := compiler.Compile(tc.brief)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			spec := result.Spec

			if expected, ok := tc.expected["backend_language"]; ok {
				if spec.App.Stack.Backend.Language != expected {
					t.Errorf("Expected backend language %s, got %s", expected, spec.App.Stack.Backend.Language)
				}
			}

			if expected, ok := tc.expected["backend_framework"]; ok {
				if spec.App.Stack.Backend.Framework != expected {
					t.Errorf("Expected backend framework %s, got %s", expected, spec.App.Stack.Backend.Framework)
				}
			}

			if expected, ok := tc.expected["database"]; ok {
				if spec.App.Stack.Database.Type != expected {
					t.Errorf("Expected database %s, got %s", expected, spec.App.Stack.Database.Type)
				}
			}

			if expected, ok := tc.expected["frontend_language"]; ok {
				if spec.App.Stack.Frontend.Language != expected {
					t.Errorf("Expected frontend language %s, got %s", expected, spec.App.Stack.Frontend.Language)
				}
			}

			if expected, ok := tc.expected["frontend_framework"]; ok {
				if spec.App.Stack.Frontend.Framework != expected {
					t.Errorf("Expected frontend framework %s, got %s", expected, spec.App.Stack.Frontend.Framework)
				}
			}

			if expected, ok := tc.expected["cache"]; ok {
				if spec.App.Stack.Cache.Type != expected {
					t.Errorf("Expected cache %s, got %s", expected, spec.App.Stack.Cache.Type)
				}
			}
		})
	}
}

func TestCompiler_SecurityAndCompliance(t *testing.T) {
	testCases := []struct {
		brief               string
		expectedCompliance  []string
		expectedAuth        []string
		expectedAudit       bool
	}{
		{
			brief:              "Create a healthcare application with HIPAA compliance",
			expectedCompliance: []string{"hipaa"},
			expectedAuth:       []string{"jwt"},
			expectedAudit:      true,
		},
		{
			brief:              "Build a fintech app with PCI compliance and OAuth2 authentication",
			expectedCompliance: []string{"pci"},
			expectedAuth:       []string{"jwt", "oauth2"},
			expectedAudit:      true,
		},
		{
			brief:              "Create a GDPR-compliant user management system",
			expectedCompliance: []string{"gdpr"},
			expectedAuth:       []string{"jwt"},
			expectedAudit:      true,
		},
	}

	compiler := NewCompiler()

	for _, tc := range testCases {
		t.Run(tc.brief, func(t *testing.T) {
			result, err := compiler.Compile(tc.brief)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			spec := result.Spec

			// Check compliance
			for _, expected := range tc.expectedCompliance {
				found := false
				for _, compliance := range spec.NonFunctionals.Security.Compliance {
					if compliance == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected compliance %s not found", expected)
				}
			}

			// Check authentication
			for _, expected := range tc.expectedAuth {
				found := false
				for _, auth := range spec.NonFunctionals.Security.Authentication {
					if auth == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected authentication %s not found", expected)
				}
			}

			// Check audit
			if spec.NonFunctionals.Security.Audit != tc.expectedAudit {
				t.Errorf("Expected audit %v, got %v", tc.expectedAudit, spec.NonFunctionals.Security.Audit)
			}
		})
	}
}

func TestCompiler_AppTypeDetection(t *testing.T) {
	testCases := []struct {
		brief       string
		expectedType string
	}{
		{"Create a REST API for user management", "api"},
		{"Build a web application with dashboard", "web"},
		{"Create a single page application", "web"},
		{"Build a mobile app for iOS and Android", "mobile"},
		{"Create a command line tool", "cli"},
		{"Build a desktop application", "desktop"},
		{"Create a microservice for payments", "api"},
	}

	compiler := NewCompiler()

	for _, tc := range testCases {
		t.Run(tc.brief, func(t *testing.T) {
			result, err := compiler.Compile(tc.brief)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			if result.Spec.App.Type != tc.expectedType {
				t.Errorf("Expected app type %s, got %s", tc.expectedType, result.Spec.App.Type)
			}
		})
	}
}

func TestCompiler_ScaleRequirements(t *testing.T) {
	testCases := []struct {
		brief         string
		expectedScale string
	}{
		{"Create a high scale enterprise application", "high"},
		{"Build a startup prototype application", "small"},
		{"Create an application for millions of users", "high"},
		{"Build a simple CRUD application", "normal"},
	}

	compiler := NewCompiler()

	for _, tc := range testCases {
		t.Run(tc.brief, func(t *testing.T) {
			result, err := compiler.Compile(tc.brief)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			scale := result.Spec.App.Scale

			switch tc.expectedScale {
			case "high":
				if scale.Users.Peak < 100000 {
					t.Errorf("Expected high scale users, got %d", scale.Users.Peak)
				}
				if scale.Uptime != "99.99%" {
					t.Errorf("Expected high uptime, got %s", scale.Uptime)
				}
			case "small":
				if scale.Users.Peak > 1000 {
					t.Errorf("Expected small scale users, got %d", scale.Users.Peak)
				}
			}
		})
	}
}

func TestCompiler_QuestionGeneration(t *testing.T) {
	compiler := NewCompiler()

	// Vague brief should generate questions
	result, err := compiler.Compile("Create an application")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Should have questions for domain clarification
	foundDomainQuestion := false
	for _, question := range result.Questions {
		if question.ID == "domain-001" {
			foundDomainQuestion = true
			break
		}
	}

	if !foundDomainQuestion {
		t.Error("Expected domain question for vague brief")
	}

	// Specific brief should have fewer questions
	result2, err := compiler.Compile("Create a FastAPI ecommerce application with PostgreSQL")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if len(result2.Questions) >= len(result.Questions) {
		t.Error("Expected fewer questions for specific brief")
	}
}

func TestCompiler_ConfidenceCalculation(t *testing.T) {
	compiler := NewCompiler()

	testCases := []struct {
		brief              string
		minConfidence      float64
		maxConfidence      float64
	}{
		{"Create an app", 0.0, 0.4}, // Very vague
		{"Create a user management system with REST API", 0.4, 0.7}, // Moderate detail
		{"Create a FastAPI ecommerce platform with user authentication, product catalog, PostgreSQL database, and payment processing", 0.7, 1.0}, // Very detailed
	}

	for _, tc := range testCases {
		t.Run(tc.brief, func(t *testing.T) {
			result, err := compiler.Compile(tc.brief)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			confidence := result.Confidence
			if confidence < tc.minConfidence || confidence > tc.maxConfidence {
				t.Errorf("Expected confidence between %f and %f, got %f", tc.minConfidence, tc.maxConfidence, confidence)
			}
		})
	}
}

func TestCompiler_EntityFieldGeneration(t *testing.T) {
	compiler := NewCompiler()

	result, err := compiler.Compile("Create a user management system with user authentication")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Find User entity
	var userEntity *Entity
	for _, entity := range result.Spec.Data.Entities {
		if entity.Name == "User" {
			userEntity = &entity
			break
		}
	}

	if userEntity == nil {
		t.Fatal("User entity not found")
	}

	// Check essential fields
	expectedFields := []string{"id", "email", "name", "password_hash", "created_at", "updated_at"}
	for _, expectedField := range expectedFields {
		found := false
		for _, field := range userEntity.Fields {
			if field.Name == expectedField {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field %s not found in User entity", expectedField)
		}
	}

	// Check constraints
	foundPrimaryKey := false
	for _, constraint := range userEntity.Constraints {
		if constraint.Type == "primary_key" && len(constraint.Fields) > 0 && constraint.Fields[0] == "id" {
			foundPrimaryKey = true
			break
		}
	}
	if !foundPrimaryKey {
		t.Error("Expected primary key constraint on id field")
	}
}

func TestCompiler_APIEndpointGeneration(t *testing.T) {
	compiler := NewCompiler()

	result, err := compiler.Compile("Create a REST API for user and product management")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if result.Spec.API.Type != "rest" {
		t.Errorf("Expected REST API, got %s", result.Spec.API.Type)
	}

	// Should have CRUD endpoints for User and Product
	expectedPaths := []string{
		"/user",      // GET list
		"/user/{id}", // GET, PUT, DELETE
		"/product",   // GET list, POST
		"/product/{id}", // GET, PUT, DELETE
	}

	for _, expectedPath := range expectedPaths {
		found := false
		for _, endpoint := range result.Spec.API.Endpoints {
			if endpoint.Path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected endpoint path %s not found", expectedPath)
		}
	}

	// Check HTTP methods
	expectedMethods := []string{"GET", "POST", "PUT", "DELETE"}
	foundMethods := make(map[string]bool)
	for _, endpoint := range result.Spec.API.Endpoints {
		foundMethods[endpoint.Method] = true
	}

	for _, method := range expectedMethods {
		if !foundMethods[method] {
			t.Errorf("Expected HTTP method %s not found", method)
		}
	}
}

func TestCompiler_EmptyBrief(t *testing.T) {
	compiler := NewCompiler()

	_, err := compiler.Compile("")
	if err == nil {
		t.Error("Expected error for empty brief")
	}

	_, err = compiler.Compile("   ")
	if err == nil {
		t.Error("Expected error for whitespace-only brief")
	}
}

func TestCompiler_JSONSerialization(t *testing.T) {
	compiler := NewCompiler()

	result, err := compiler.Compile("Create a user management API")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Test JSON serialization
	jsonData, err := json.Marshal(result.Spec)
	if err != nil {
		t.Fatalf("JSON serialization failed: %v", err)
	}

	// Test JSON deserialization
	var spec IRSpec
	err = json.Unmarshal(jsonData, &spec)
	if err != nil {
		t.Fatalf("JSON deserialization failed: %v", err)
	}

	// Verify key fields are preserved
	if spec.Brief != result.Spec.Brief {
		t.Error("Brief not preserved in JSON round-trip")
	}

	if spec.App.Type != result.Spec.App.Type {
		t.Error("App type not preserved in JSON round-trip")
	}
}

func TestCompiler_RelationshipDetection(t *testing.T) {
	compiler := NewCompiler()

	result, err := compiler.Compile("Create an ecommerce system with users placing orders")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// Should detect User and Order entities
	hasUser := false
	hasOrder := false
	for _, entity := range result.Spec.Data.Entities {
		if entity.Name == "User" {
			hasUser = true
		}
		if entity.Name == "Order" {
			hasOrder = true
		}
	}

	if !hasUser || !hasOrder {
		t.Error("Expected User and Order entities to be detected")
	}

	// Should detect relationship between User and Order
	foundRelationship := false
	for _, rel := range result.Spec.Data.Relationships {
		if rel.From == "User" && rel.To == "Order" && rel.Type == "one_to_many" {
			foundRelationship = true
			break
		}
	}

	if !foundRelationship {
		t.Error("Expected User -> Order relationship to be detected")
	}
}

func TestCompiler_WarningGeneration(t *testing.T) {
	compiler := NewCompiler()

	// Very vague brief should generate warnings
	result, err := compiler.Compile("Create an app")
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for vague brief")
	}

	// Check for specific warnings
	expectedWarnings := []string{
		"Low confidence",
		"No data entities",
		"No specific features",
		"Generic domain",
	}

	for _, expectedWarning := range expectedWarnings {
		found := false
		for _, warning := range result.Warnings {
			if strings.Contains(warning, expectedWarning) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected warning containing '%s' not found", expectedWarning)
		}
	}
}