package ir

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Compiler transforms natural language briefs into structured IR specifications
type Compiler struct {
	patterns        map[string]*regexp.Regexp
	defaults        CompilerDefaults
	overlayDetector *OverlayDetector
}

// CompilerDefaults provides sensible defaults for common patterns
type CompilerDefaults struct {
	Backend  BackendStack
	Frontend FrontendStack
	Database DatabaseStack
	Cache    CacheStack
}

// CompilationResult contains the IR spec and any issues found
type CompilationResult struct {
	Spec               *IRSpec                 `json:"spec"`
	Questions          []BlockingQuestion      `json:"questions"`
	Warnings           []string                `json:"warnings"`
	Confidence         float64                 `json:"confidence"`
	OverlayDetection   *OverlayDetectionResult `json:"overlay_detection,omitempty"`
	SuggestedOverlays  []string                `json:"suggested_overlays,omitempty"`
	RequiredOverlays   []string                `json:"required_overlays,omitempty"`
}

// NewCompiler creates a new IR compiler with default patterns
func NewCompiler() *Compiler {
	return &Compiler{
		patterns:        initializePatterns(),
		defaults:        getDefaults(),
		overlayDetector: NewOverlayDetector(),
	}
}

// Compile transforms a natural language brief into an IR specification
func (c *Compiler) Compile(brief string) (*CompilationResult, error) {
	if strings.TrimSpace(brief) == "" {
		return nil, fmt.Errorf("brief cannot be empty")
	}

	// Normalize the brief
	normalizedBrief := c.normalizeBrief(brief)

	// Initialize the IR spec
	spec := &IRSpec{
		Version:        "1.0",
		Brief:          brief,
		App:            c.extractAppSpec(normalizedBrief),
		NonFunctionals: c.extractNonFunctionals(normalizedBrief),
		API:            c.extractAPISpec(normalizedBrief),
		Data:           c.extractDataSpec(normalizedBrief),
		UI:             c.extractUISpec(normalizedBrief),
		Ops:            c.extractOpsSpec(normalizedBrief),
		Acceptance:     c.extractAcceptanceCriteria(normalizedBrief),
		Metadata: SpecMetadata{
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Version:     "1.0",
			Source:      "ai",
			Tags:        c.extractTags(normalizedBrief),
		},
	}

	// Generate questions for ambiguities
	questions := c.generateQuestions(spec, normalizedBrief)

	// Calculate confidence and completeness
	confidence := c.calculateConfidence(spec, normalizedBrief)
	completeness := c.calculateCompleteness(spec)

	spec.Metadata.Confidence = confidence
	spec.Metadata.Completeness = completeness
	spec.Questions = questions

	// Generate warnings
	warnings := c.generateWarnings(spec)

	// Detect overlays from brief
	overlayDetection := c.overlayDetector.DetectOverlays(brief)

	// Extract suggested and required overlays
	var suggestedOverlays []string
	var requiredOverlays []string

	for _, suggestion := range overlayDetection.Suggestions {
		if suggestion.Confidence >= 0.8 {
			requiredOverlays = append(requiredOverlays, suggestion.Name)
		} else if suggestion.Confidence >= 0.5 {
			suggestedOverlays = append(suggestedOverlays, suggestion.Name)
		}
	}

	// Add overlay warnings to main warnings
	warnings = append(warnings, overlayDetection.Warnings...)

	return &CompilationResult{
		Spec:              spec,
		Questions:         questions,
		Warnings:          warnings,
		Confidence:        confidence,
		OverlayDetection:  overlayDetection,
		SuggestedOverlays: suggestedOverlays,
		RequiredOverlays:  requiredOverlays,
	}, nil
}

// CompileWithOverlays compiles a brief with explicitly specified overlays
func (c *Compiler) CompileWithOverlays(brief string, overlays []string) (*CompilationResult, error) {
	// First get the base compilation result
	result, err := c.Compile(brief)
	if err != nil {
		return nil, err
	}

	// Validate overlay compatibility
	compatibilityWarnings := c.overlayDetector.ValidateOverlayCompatibility(overlays, result.Spec)
	result.Warnings = append(result.Warnings, compatibilityWarnings...)

	// Add the explicitly specified overlays
	result.RequiredOverlays = append(result.RequiredOverlays, overlays...)

	// Deduplicate required overlays
	result.RequiredOverlays = c.overlayDetector.deduplicateStrings(result.RequiredOverlays)

	// Remove duplicates from suggested overlays that are already in required
	filteredSuggested := []string{}
	for _, suggested := range result.SuggestedOverlays {
		found := false
		for _, required := range result.RequiredOverlays {
			if suggested == required {
				found = true
				break
			}
		}
		if !found {
			filteredSuggested = append(filteredSuggested, suggested)
		}
	}
	result.SuggestedOverlays = filteredSuggested

	return result, nil
}

// SuggestOverlays analyzes a brief and returns overlay suggestions without full compilation
func (c *Compiler) SuggestOverlays(brief string) *OverlayDetectionResult {
	return c.overlayDetector.DetectOverlays(brief)
}

// normalizeBrief cleans and standardizes the input text
func (c *Compiler) normalizeBrief(brief string) string {
	// Convert to lowercase for pattern matching
	normalized := strings.ToLower(brief)

	// Remove extra whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Trim
	normalized = strings.TrimSpace(normalized)

	return normalized
}

// extractAppSpec extracts application-level specifications
func (c *Compiler) extractAppSpec(brief string) AppSpec {
	app := AppSpec{
		Name:        c.extractAppName(brief),
		Description: c.extractDescription(brief),
		Type:        c.extractAppType(brief),
		Domain:      c.extractDomain(brief),
		Stack:       c.extractTechStack(brief),
		Features:    c.extractFeatures(brief),
		Scale:       c.extractScaleRequirements(brief),
	}

	return app
}

func (c *Compiler) extractAppName(brief string) string {
	// Look for explicit names
	patterns := []string{
		`(?:app|application|system|service|platform)\s+(?:called|named)\s+["']?([^"'\s]+)["']?`,
		`["']([^"']+)["']\s+(?:app|application|system)`,
		`create\s+(?:a|an)?\s*["']?([^"'\s]+)["']?\s+(?:app|application|system)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(brief); len(matches) > 1 {
			return strings.Title(strings.ReplaceAll(matches[1], "_", " "))
		}
	}

	// Generate name from domain/type
	domain := c.extractDomain(brief)
	appType := c.extractAppType(brief)

	if domain != "" && appType != "" {
		return fmt.Sprintf("%s %s", strings.Title(domain), strings.Title(appType))
	}

	return "Generated Application"
}

func (c *Compiler) extractDescription(brief string) string {
	// Use the first sentence or up to 200 chars
	sentences := regexp.MustCompile(`[.!?]+`).Split(brief, -1)
	if len(sentences) > 0 && len(sentences[0]) > 0 {
		desc := strings.TrimSpace(sentences[0])
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		return desc
	}
	return brief
}

func (c *Compiler) extractAppType(brief string) string {
	patterns := map[string]string{
		`\b(?:rest\s+)?api\b`:                    "api",
		`\bweb\s+(?:app|application)\b`:          "web",
		`\bspa\b|single\s+page\s+application`:   "web",
		`\bmobile\s+app`:                        "mobile",
		`\bcli\b|command\s+line`:                "cli",
		`\bdesktop\s+app`:                       "desktop",
		`\bmicroservice`:                        "api",
		`\bwebsite\b`:                           "web",
		`\bdashboard\b`:                         "web",
	}

	for pattern, appType := range patterns {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			return appType
		}
	}

	// Default based on features mentioned
	if c.containsPattern(brief, `\b(?:endpoint|route|controller)\b`) {
		return "api"
	}
	if c.containsPattern(brief, `\b(?:page|component|ui|interface)\b`) {
		return "web"
	}

	return "web" // default
}

func (c *Compiler) extractDomain(brief string) string {
	domains := map[string]string{
		`\b(?:ecommerce|e-commerce|shop|store|cart|product|order)\b`:     "ecommerce",
		`\b(?:fintech|banking|payment|invoice|billing|finance)\b`:        "fintech",
		`\b(?:healthcare|medical|patient|doctor|hospital)\b`:             "healthcare",
		`\b(?:education|learning|course|student|teacher)\b`:              "education",
		`\b(?:social|media|post|comment|follow|friend)\b`:                "social",
		`\b(?:blog|news|article|content|cms)\b`:                          "content",
		`\b(?:saas|crm|erp|hr|human\s+resource)\b`:                       "business",
		`\b(?:iot|sensor|device|monitoring)\b`:                           "iot",
		`\b(?:game|gaming|player|score)\b`:                               "gaming",
		`\b(?:real\s+estate|property|listing)\b`:                         "realestate",
		`\b(?:travel|booking|hotel|flight|reservation)\b`:                "travel",
		`\b(?:food|restaurant|menu|delivery)\b`:                          "food",
	}

	for pattern, domain := range domains {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			return domain
		}
	}

	return "general"
}

func (c *Compiler) extractTechStack(brief string) TechStack {
	stack := TechStack{
		Backend:  c.extractBackendStack(brief),
		Frontend: c.extractFrontendStack(brief),
		Database: c.extractDatabaseStack(brief),
		Cache:    c.extractCacheStack(brief),
	}

	return stack
}

func (c *Compiler) extractBackendStack(brief string) BackendStack {
	// Language detection
	languages := map[string]string{
		`\b(?:python|py|fastapi|django|flask)\b`:     "python",
		`\b(?:golang|go|gin|echo|fiber)\b`:           "go",
		`\b(?:node|nodejs|express|nest|koa)\b`:       "nodejs",
		`\b(?:java|spring|springboot)\b`:             "java",
		`\b(?:ruby|rails|sinatra)\b`:                 "ruby",
		`\b(?:php|laravel|symfony)\b`:                "php",
		`\b(?:rust|actix|warp)\b`:                    "rust",
		`\b(?:dotnet|csharp|c#|asp\.net)\b`:          "csharp",
	}

	// Framework detection
	frameworks := map[string]map[string]string{
		"python": {
			`\bfastapi\b`:  "fastapi",
			`\bdjango\b`:   "django",
			`\bflask\b`:    "flask",
		},
		"go": {
			`\bgin\b`:      "gin",
			`\becho\b`:     "echo",
			`\bfiber\b`:    "fiber",
		},
		"nodejs": {
			`\bexpress\b`:  "express",
			`\bnest\b`:     "nestjs",
			`\bkoa\b`:      "koa",
		},
		"java": {
			`\bspring\b`:   "spring",
			`\bspringboot\b`: "springboot",
		},
	}

	language := c.defaults.Backend.Language
	framework := c.defaults.Backend.Framework

	// Detect language
	for pattern, lang := range languages {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			language = lang
			break
		}
	}

	// Detect framework for the detected language
	if langFrameworks, exists := frameworks[language]; exists {
		for pattern, fw := range langFrameworks {
			if matched, _ := regexp.MatchString(pattern, brief); matched {
				framework = fw
				break
			}
		}
	}

	return BackendStack{
		Language:  language,
		Framework: framework,
		Libraries: c.extractLibraries(brief, language),
		Runtime:   c.extractRuntime(brief),
	}
}

func (c *Compiler) extractFrontendStack(brief string) FrontendStack {
	// Only include frontend if it's a web app
	appType := c.extractAppType(brief)
	if appType != "web" {
		return FrontendStack{}
	}

	languages := map[string]string{
		`\btypescript\b|ts\b`:                      "typescript",
		`\bjavascript\b|js\b`:                      "javascript",
	}

	frameworks := map[string]string{
		`\breact\b|reactjs\b`:                      "react",
		`\bvue\b|vuejs\b`:                          "vue",
		`\bangular\b`:                              "angular",
		`\bsvelte\b`:                               "svelte",
		`\bnext\b|nextjs\b`:                        "next",
		`\bnuxt\b`:                                 "nuxt",
	}

	language := "typescript" // default
	framework := "react"     // default

	for pattern, lang := range languages {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			language = lang
			break
		}
	}

	for pattern, fw := range frameworks {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			framework = fw
			break
		}
	}

	return FrontendStack{
		Language:  language,
		Framework: framework,
		Libraries: c.extractFrontendLibraries(brief, framework),
		Build:     c.extractBuildTool(brief, framework),
	}
}

func (c *Compiler) extractDatabaseStack(brief string) DatabaseStack {
	// Primary database patterns (not cache)
	databases := map[string]string{
		`\bpostgresql\b|postgres\b`:               "postgresql",
		`\bmysql\b`:                               "mysql",
		`\bmongodb\b|mongo\b`:                     "mongodb",
		`\bsqlite\b`:                              "sqlite",
		`\bcassandra\b`:                           "cassandra",
	}

	dbType := c.defaults.Database.Type
	for pattern, db := range databases {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			dbType = db
			break
		}
	}

	// Don't treat Redis as primary database - it's for caching
	version := "latest"
	if dbType == "postgresql" {
		version = "15"
	} else if dbType == "mysql" {
		version = "8.0"
	} else if dbType == "mongodb" {
		version = "6.0"
	}

	return DatabaseStack{
		Type:     dbType,
		Version:  version,
		Features: []string{"migrations", "indexes"},
	}
}

func (c *Compiler) extractCacheStack(brief string) CacheStack {
	if c.containsPattern(brief, `\b(?:cache|caching|redis|memcache)\b`) {
		return CacheStack{
			Type:    "redis",
			Version: "7",
		}
	}
	return CacheStack{}
}

func (c *Compiler) extractFeatures(brief string) []Feature {
	features := []Feature{}

	// Common feature patterns
	featurePatterns := map[string]Feature{
		`\b(?:user|auth|login|signup|register)\b`: {
			Name:        "User Authentication",
			Description: "User registration, login, and authentication",
			Type:        "auth",
			Priority:    "high",
			Operations:  []string{"register", "login", "logout", "verify"},
		},
		`\b(?:crud|create|read|update|delete)\b`: {
			Name:        "CRUD Operations",
			Description: "Create, read, update, and delete operations",
			Type:        "crud",
			Priority:    "high",
			Operations:  []string{"create", "read", "update", "delete"},
		},
		`\b(?:payment|billing|invoice|checkout)\b`: {
			Name:        "Payment Processing",
			Description: "Handle payments and billing",
			Type:        "payment",
			Priority:    "high",
			Operations:  []string{"charge", "refund", "webhook"},
		},
		`\b(?:notification|email|sms|alert)\b`: {
			Name:        "Notifications",
			Description: "Send notifications to users",
			Type:        "notification",
			Priority:    "medium",
			Operations:  []string{"send", "template", "schedule"},
		},
		`\b(?:search|filter|query)\b`: {
			Name:        "Search & Filter",
			Description: "Search and filter functionality",
			Type:        "search",
			Priority:    "medium",
			Operations:  []string{"search", "filter", "sort"},
		},
		`\b(?:upload|file|document|image)\b`: {
			Name:        "File Upload",
			Description: "File upload and management",
			Type:        "file",
			Priority:    "medium",
			Operations:  []string{"upload", "download", "delete"},
		},
	}

	for pattern, feature := range featurePatterns {
		if matched, _ := regexp.MatchString(pattern, brief); matched {
			features = append(features, feature)
		}
	}

	return features
}

func (c *Compiler) extractScaleRequirements(brief string) ScaleRequirements {
	scale := ScaleRequirements{
		Users:       ScaleMetric{Initial: 100, Peak: 1000, Growth: 50},
		Requests:    ScaleMetric{Initial: 1000, Peak: 10000, Growth: 100},
		Storage:     ScaleMetric{Initial: 1, Peak: 100, Growth: 200}, // GB
		Latency:     "500ms",
		Uptime:      "99.9%",
		Concurrency: 100,
	}

	// Look for scale indicators
	if c.containsPattern(brief, `\b(?:high\s+scale|enterprise|millions?\s+of\s+users|millions?\s+users)\b`) {
		scale.Users.Peak = 1000000
		scale.Requests.Peak = 100000
		scale.Uptime = "99.99%"
		scale.Latency = "100ms"
	} else if c.containsPattern(brief, `\b(?:startup|small|prototype)\b`) {
		scale.Users.Peak = 100
		scale.Requests.Peak = 1000
	}

	return scale
}

func (c *Compiler) extractNonFunctionals(brief string) NonFunctionalSpec {
	return NonFunctionalSpec{
		Security:    c.extractSecuritySpec(brief),
		Performance: c.extractPerformanceSpec(brief),
		Compliance:  c.extractComplianceSpec(brief),
		Monitoring:  c.extractMonitoringSpec(brief),
	}
}

func (c *Compiler) extractSecuritySpec(brief string) SecuritySpec {
	auth := []string{"jwt"}
	if c.containsPattern(brief, `\boauth2?\b`) {
		auth = append(auth, "oauth2")
	}
	if c.containsPattern(brief, `\bbasic\s+auth\b`) {
		auth = append(auth, "basic")
	}

	authz := []string{"rbac"}
	if c.containsPattern(brief, `\bacl\b|access\s+control\s+list\b`) {
		authz = append(authz, "acl")
	}

	encryption := []string{"tls", "bcrypt"}
	if c.containsPattern(brief, `\baes\b`) {
		encryption = append(encryption, "aes256")
	}

	var compliance []string
	if c.containsPattern(brief, `\bgdpr\b`) {
		compliance = append(compliance, "gdpr")
	}
	if c.containsPattern(brief, `\bhipaa\b`) {
		compliance = append(compliance, "hipaa")
	}
	if c.containsPattern(brief, `\bpci\b`) {
		compliance = append(compliance, "pci")
	}

	// Audit is required for compliance or if explicitly mentioned
	audit := c.containsPattern(brief, `\baudit\b`) || len(compliance) > 0

	return SecuritySpec{
		Authentication: auth,
		Authorization:  authz,
		Encryption:     encryption,
		Audit:         audit,
		Compliance:    compliance,
	}
}

func (c *Compiler) extractPerformanceSpec(brief string) PerformanceSpec {
	responseTime := "500ms"
	if c.containsPattern(brief, `\bfast|quick|real.?time\b`) {
		responseTime = "100ms"
	}

	return PerformanceSpec{
		ResponseTime: responseTime,
		Throughput:   "1000rps",
		Memory:       "512MB",
		CPU:          "2cores",
	}
}

func (c *Compiler) extractComplianceSpec(brief string) ComplianceSpec {
	var standards []string
	dataRetention := "7years"

	if c.containsPattern(brief, `\bgdpr\b`) {
		standards = append(standards, "gdpr")
		dataRetention = "2years"
	}
	if c.containsPattern(brief, `\bhipaa\b`) {
		standards = append(standards, "hipaa")
		dataRetention = "7years"
	}
	if c.containsPattern(brief, `\bpci\b`) {
		standards = append(standards, "pci-dss")
	}

	return ComplianceSpec{
		Standards:     standards,
		DataRetention: dataRetention,
		AuditLog:      len(standards) > 0,
	}
}

func (c *Compiler) extractMonitoringSpec(brief string) MonitoringSpec {
	metrics := []string{"cpu", "memory", "requests", "errors"}
	logging := "structured"
	tracing := c.containsPattern(brief, `\btracing|trace\b`)

	return MonitoringSpec{
		Metrics: metrics,
		Logging: logging,
		Tracing: tracing,
		Alerts:  []AlertRule{},
	}
}

func (c *Compiler) extractAPISpec(brief string) APISpec {
	apiType := "rest"
	if c.containsPattern(brief, `\bgraphql\b`) {
		apiType = "graphql"
	} else if c.containsPattern(brief, `\bgrpc\b`) {
		apiType = "grpc"
	}

	return APISpec{
		Type:    apiType,
		Version: "v1",
		BaseURL: "/api/v1",
		Auth: AuthSpec{
			Type:     "bearer",
			Required: true,
		},
		Endpoints: c.extractEndpoints(brief),
		Schemas:   []Schema{},
		Config: APIConfig{
			CORS:        true,
			Compression: true,
			Versioning:  "path",
			Pagination:  "offset",
		},
	}
}

func (c *Compiler) extractEndpoints(brief string) []Endpoint {
	endpoints := []Endpoint{}

	// Extract entities mentioned
	entities := c.extractEntities(brief)

	// Generate CRUD endpoints for each entity
	for _, entity := range entities {
		entityName := entity.Name
		path := fmt.Sprintf("/%s", strings.ToLower(entityName))

		// GET list
		endpoints = append(endpoints, Endpoint{
			Path:        path,
			Method:      "GET",
			Summary:     fmt.Sprintf("List %s", entityName),
			Description: fmt.Sprintf("Retrieve a list of %s", entityName),
			Responses: map[string]Response{
				"200": {Description: "Success", Schema: fmt.Sprintf("%sList", entityName)},
			},
			Auth: true,
		})

		// GET by ID
		endpoints = append(endpoints, Endpoint{
			Path:        fmt.Sprintf("%s/{id}", path),
			Method:      "GET",
			Summary:     fmt.Sprintf("Get %s", entityName),
			Description: fmt.Sprintf("Retrieve a specific %s by ID", entityName),
			Parameters: []Parameter{
				{Name: "id", In: "path", Type: "string", Required: true},
			},
			Responses: map[string]Response{
				"200": {Description: "Success", Schema: entityName},
				"404": {Description: "Not found"},
			},
			Auth: true,
		})

		// POST create
		endpoints = append(endpoints, Endpoint{
			Path:        path,
			Method:      "POST",
			Summary:     fmt.Sprintf("Create %s", entityName),
			Description: fmt.Sprintf("Create a new %s", entityName),
			RequestBody: &RequestBody{
				Required:    true,
				ContentType: "application/json",
				Schema:      fmt.Sprintf("Create%s", entityName),
			},
			Responses: map[string]Response{
				"201": {Description: "Created", Schema: entityName},
				"400": {Description: "Bad request"},
			},
			Auth: true,
		})

		// PUT update
		endpoints = append(endpoints, Endpoint{
			Path:        fmt.Sprintf("%s/{id}", path),
			Method:      "PUT",
			Summary:     fmt.Sprintf("Update %s", entityName),
			Description: fmt.Sprintf("Update an existing %s", entityName),
			Parameters: []Parameter{
				{Name: "id", In: "path", Type: "string", Required: true},
			},
			RequestBody: &RequestBody{
				Required:    true,
				ContentType: "application/json",
				Schema:      fmt.Sprintf("Update%s", entityName),
			},
			Responses: map[string]Response{
				"200": {Description: "Updated", Schema: entityName},
				"404": {Description: "Not found"},
			},
			Auth: true,
		})

		// DELETE
		endpoints = append(endpoints, Endpoint{
			Path:        fmt.Sprintf("%s/{id}", path),
			Method:      "DELETE",
			Summary:     fmt.Sprintf("Delete %s", entityName),
			Description: fmt.Sprintf("Delete a %s", entityName),
			Parameters: []Parameter{
				{Name: "id", In: "path", Type: "string", Required: true},
			},
			Responses: map[string]Response{
				"204": {Description: "Deleted"},
				"404": {Description: "Not found"},
			},
			Auth: true,
		})
	}

	return endpoints
}

func (c *Compiler) extractDataSpec(brief string) DataSpec {
	entities := c.extractEntities(brief)
	relationships := c.extractRelationships(brief, entities)

	return DataSpec{
		Entities:      entities,
		Relationships: relationships,
		Migrations:    []Migration{},
		Seeds:         []Seed{},
		Indexes:       []Index{},
		Config: DataConfig{
			Migrations:  true,
			Seeds:       true,
			Soft_Delete: false,
			Timestamps:  true,
		},
	}
}

func (c *Compiler) extractEntities(brief string) []Entity {
	entities := []Entity{}

	// Define entity patterns and names in order
	entityDefs := []struct {
		pattern string
		name    string
		fields  []Field
	}{
		{
			pattern: `(?i)\busers?\b`,
			name:    "User",
			fields: []Field{
				{Name: "id", Type: "uuid", Required: true, Unique: true},
				{Name: "email", Type: "string", Required: true, Unique: true},
				{Name: "name", Type: "string", Required: true},
				{Name: "password_hash", Type: "string", Required: true},
				{Name: "created_at", Type: "timestamp", Required: true},
				{Name: "updated_at", Type: "timestamp", Required: true},
			},
		},
		{
			pattern: `(?i)\bproducts?\b`,
			name:    "Product",
			fields: []Field{
				{Name: "id", Type: "uuid", Required: true, Unique: true},
				{Name: "name", Type: "string", Required: true},
				{Name: "description", Type: "text"},
				{Name: "price", Type: "decimal", Required: true},
				{Name: "created_at", Type: "timestamp", Required: true},
				{Name: "updated_at", Type: "timestamp", Required: true},
			},
		},
		{
			pattern: `(?i)\b(?:orders?|carts?|purchases?)\b`,
			name:    "Order",
			fields: []Field{
				{Name: "id", Type: "uuid", Required: true, Unique: true},
				{Name: "user_id", Type: "uuid", Required: true},
				{Name: "total", Type: "decimal", Required: true},
				{Name: "status", Type: "string", Required: true, Default: "pending"},
				{Name: "created_at", Type: "timestamp", Required: true},
				{Name: "updated_at", Type: "timestamp", Required: true},
			},
		},
		{
			pattern: `(?i)\binvoices?\b`,
			name:    "Invoice",
			fields: []Field{
				{Name: "id", Type: "uuid", Required: true, Unique: true},
				{Name: "number", Type: "string", Required: true, Unique: true},
				{Name: "customer_id", Type: "uuid", Required: true},
				{Name: "amount", Type: "decimal", Required: true},
				{Name: "status", Type: "string", Required: true, Default: "draft"},
				{Name: "due_date", Type: "date", Required: true},
				{Name: "created_at", Type: "timestamp", Required: true},
				{Name: "updated_at", Type: "timestamp", Required: true},
			},
		},
	}

	for _, entityDef := range entityDefs {
		if matched, _ := regexp.MatchString(entityDef.pattern, brief); matched {
			entities = append(entities, Entity{
				Name:        entityDef.name,
				Description: fmt.Sprintf("%s entity", entityDef.name),
				Fields:      entityDef.fields,
				Constraints: []Constraint{
					{Type: "primary_key", Fields: []string{"id"}},
				},
			})
		}
	}

	return entities
}

func (c *Compiler) extractRelationships(brief string, entities []Entity) []Relationship {
	relationships := []Relationship{}

	// Create a map of entity names for lookup
	entityMap := make(map[string]bool)
	for _, entity := range entities {
		entityMap[strings.ToLower(entity.Name)] = true
	}

	// Common relationship patterns
	if entityMap["user"] && entityMap["order"] {
		relationships = append(relationships, Relationship{
			From:       "User",
			To:         "Order",
			Type:       "one_to_many",
			ForeignKey: "user_id",
		})
	}

	if entityMap["user"] && entityMap["invoice"] {
		relationships = append(relationships, Relationship{
			From:       "User",
			To:         "Invoice",
			Type:       "one_to_many",
			ForeignKey: "customer_id",
		})
	}

	return relationships
}

func (c *Compiler) extractUISpec(brief string) UISpec {
	appType := c.extractAppType(brief)
	if appType != "web" {
		return UISpec{}
	}

	return UISpec{
		Type:       "spa",
		Pages:      c.extractPages(brief),
		Components: []Component{},
		Theme: Theme{
			Primary:   "#007bff",
			Secondary: "#6c757d",
			Style:     "bootstrap",
		},
		Config: UIConfig{
			Responsive: true,
			PWA:        false,
			I18n:       false,
		},
	}
}

func (c *Compiler) extractPages(brief string) []Page {
	pages := []Page{
		{Name: "Home", Path: "/", Title: "Home", Auth: false},
		{Name: "Login", Path: "/login", Title: "Login", Auth: false},
	}

	// Add entity-specific pages
	entities := c.extractEntities(brief)
	for _, entity := range entities {
		entityName := strings.ToLower(entity.Name)
		pages = append(pages, Page{
			Name:  fmt.Sprintf("%s List", entity.Name),
			Path:  fmt.Sprintf("/%s", entityName),
			Title: fmt.Sprintf("%s Management", entity.Name),
			Auth:  true,
		})
	}

	return pages
}

func (c *Compiler) extractOpsSpec(brief string) OpsSpec {
	return OpsSpec{
		Environment: []Environment{
			{Name: "development", Provider: "docker", Region: "local"},
			{Name: "staging", Provider: "k8s", Region: "us-east-1"},
			{Name: "production", Provider: "k8s", Region: "us-east-1"},
		},
		CI_CD: CICDSpec{
			Provider: "github",
			Triggers: []string{"push", "pr"},
			Stages:   []string{"test", "build", "deploy"},
		},
		Monitoring: OpsMonitoring{
			Provider:   "prometheus",
			Dashboards: []string{"overview", "performance", "errors"},
			Alerts:     []string{"high_error_rate", "high_latency"},
		},
		Backup: BackupSpec{
			Frequency: "daily",
			Retention: "30d",
			Provider:  "s3",
		},
		Scaling: ScalingSpec{
			Type:     "horizontal",
			Min:      2,
			Max:      10,
			Triggers: []string{"cpu", "memory"},
		},
		Config: OpsConfig{
			SSL:          true,
			CDN:          true,
			LoadBalancer: true,
		},
	}
}

func (c *Compiler) extractAcceptanceCriteria(brief string) []AcceptanceCriteria {
	criteria := []AcceptanceCriteria{
		{
			ID:          "functional-001",
			Description: "All API endpoints return proper HTTP status codes",
			Type:        "functional",
			Priority:    "must",
			Testable:    true,
			Automated:   true,
		},
		{
			ID:          "security-001",
			Description: "All endpoints require proper authentication",
			Type:        "security",
			Priority:    "must",
			Testable:    true,
			Automated:   true,
		},
		{
			ID:          "performance-001",
			Description: "API response time is under 500ms for 95% of requests",
			Type:        "performance",
			Priority:    "should",
			Testable:    true,
			Automated:   true,
		},
	}

	return criteria
}

// Helper methods

func (c *Compiler) containsPattern(text, pattern string) bool {
	matched, _ := regexp.MatchString(pattern, text)
	return matched
}

func (c *Compiler) extractTags(brief string) []string {
	tags := []string{}

	if c.containsPattern(brief, `\bapi\b`) {
		tags = append(tags, "api")
	}
	if c.containsPattern(brief, `\bweb\b`) {
		tags = append(tags, "web")
	}
	if c.containsPattern(brief, `\bcrud\b`) {
		tags = append(tags, "crud")
	}
	if c.containsPattern(brief, `\bauth\b`) {
		tags = append(tags, "auth")
	}

	return tags
}

func (c *Compiler) extractLibraries(brief, language string) []string {
	libraries := []string{}

	libraryMap := map[string]map[string]string{
		"python": {
			`\bpydantic\b`: "pydantic",
			`\bsqlalchemy\b`: "sqlalchemy",
			`\balembic\b`: "alembic",
			`\bcelery\b`: "celery",
		},
		"go": {
			`\bgorm\b`: "gorm",
			`\bmux\b`: "gorilla/mux",
			`\bviper\b`: "viper",
		},
		"nodejs": {
			`\bmongose\b`: "mongoose",
			`\bsequelize\b`: "sequelize",
			`\bpassport\b`: "passport",
		},
	}

	if langLibs, exists := libraryMap[language]; exists {
		for pattern, lib := range langLibs {
			if c.containsPattern(brief, pattern) {
				libraries = append(libraries, lib)
			}
		}
	}

	return libraries
}

func (c *Compiler) extractFrontendLibraries(brief, framework string) []string {
	libraries := []string{}

	if framework == "react" {
		libraries = append(libraries, "@tanstack/react-query", "react-router-dom")
		if c.containsPattern(brief, `\bui\b|design\b`) {
			libraries = append(libraries, "@mui/material")
		}
	}

	return libraries
}

func (c *Compiler) extractBuildTool(brief, framework string) string {
	if framework == "next" {
		return "next"
	}
	if c.containsPattern(brief, `\bvite\b`) {
		return "vite"
	}
	if c.containsPattern(brief, `\bwebpack\b`) {
		return "webpack"
	}
	return "vite" // default
}

func (c *Compiler) extractRuntime(brief string) string {
	if c.containsPattern(brief, `\bserverless\b|lambda\b`) {
		return "serverless"
	}
	return "docker" // default
}

func (c *Compiler) generateQuestions(spec *IRSpec, brief string) []BlockingQuestion {
	questions := []BlockingQuestion{}

	// Generate questions based on missing or ambiguous information
	if spec.App.Domain == "general" {
		questions = append(questions, BlockingQuestion{
			ID:       "domain-001",
			Question: "What is the primary domain or industry for this application?",
			Context:  "This helps determine appropriate features and compliance requirements",
			Type:     "business",
			Options:  []string{"ecommerce", "fintech", "healthcare", "education", "social", "business"},
			Required: false,
		})
	}

	if len(spec.NonFunctionals.Security.Compliance) == 0 {
		questions = append(questions, BlockingQuestion{
			ID:       "compliance-001",
			Question: "Are there any specific compliance requirements (GDPR, HIPAA, PCI, etc.)?",
			Context:  "This affects data handling, security measures, and audit requirements",
			Type:     "security",
			Options:  []string{"none", "gdpr", "hipaa", "pci-dss", "sox"},
			Required: false,
		})
	}

	return questions
}

func (c *Compiler) calculateConfidence(spec *IRSpec, brief string) float64 {
	confidence := 0.5 // base confidence

	// Increase confidence based on specificity
	if len(spec.App.Features) > 2 {
		confidence += 0.1
	}
	if spec.App.Domain != "general" {
		confidence += 0.1
	}
	if len(spec.Data.Entities) > 0 {
		confidence += 0.15
	}
	if len(spec.API.Endpoints) > 5 {
		confidence += 0.1
	}

	// Decrease confidence for very short or vague briefs
	if len(brief) < 50 {
		confidence -= 0.2
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

func (c *Compiler) calculateCompleteness(spec *IRSpec) float64 {
	completeness := 0.0
	total := 10.0

	// Check various aspects
	if spec.App.Name != "" && spec.App.Name != "Generated Application" {
		completeness += 1.0
	}
	if len(spec.App.Features) > 0 {
		completeness += 1.0
	}
	if len(spec.Data.Entities) > 0 {
		completeness += 1.5
	}
	if len(spec.API.Endpoints) > 0 {
		completeness += 1.5
	}
	if spec.App.Stack.Backend.Language != "" {
		completeness += 1.0
	}
	if spec.App.Stack.Database.Type != "" {
		completeness += 1.0
	}
	if len(spec.NonFunctionals.Security.Authentication) > 0 {
		completeness += 1.0
	}
	if len(spec.Ops.Environment) > 0 {
		completeness += 1.0
	}
	if len(spec.Acceptance) > 0 {
		completeness += 1.0
	}

	return completeness / total
}

func (c *Compiler) generateWarnings(spec *IRSpec) []string {
	warnings := []string{}

	if spec.Metadata.Confidence < 0.6 {
		warnings = append(warnings, "Low confidence in specification accuracy - consider providing more details")
	}

	if len(spec.Data.Entities) == 0 {
		warnings = append(warnings, "No data entities identified - may need to define data model manually")
	}

	// Check if only generic features were detected
	hasSpecificFeatures := false
	for _, feature := range spec.App.Features {
		if feature.Type != "crud" {
			hasSpecificFeatures = true
			break
		}
	}

	if !hasSpecificFeatures {
		warnings = append(warnings, "No specific features identified - using default CRUD operations")
	}

	if spec.App.Domain == "general" {
		warnings = append(warnings, "Generic domain detected - consider specifying industry for better defaults")
	}

	return warnings
}

// Initialize patterns and defaults

func initializePatterns() map[string]*regexp.Regexp {
	patterns := make(map[string]*regexp.Regexp)

	// Add common patterns here if needed
	patterns["email"] = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	patterns["url"] = regexp.MustCompile(`https?://[^\s]+`)

	return patterns
}

func getDefaults() CompilerDefaults {
	return CompilerDefaults{
		Backend: BackendStack{
			Language:  "python",
			Framework: "fastapi",
			Runtime:   "docker",
		},
		Frontend: FrontendStack{
			Language:  "typescript",
			Framework: "react",
			Build:     "vite",
		},
		Database: DatabaseStack{
			Type:     "postgresql",
			Version:  "15",
			Features: []string{"migrations", "indexes"},
		},
		Cache: CacheStack{
			Type:    "redis",
			Version: "7",
		},
	}
}