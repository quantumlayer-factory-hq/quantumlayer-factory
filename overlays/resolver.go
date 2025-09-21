package overlays

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// FileSystemResolver implements the Resolver interface using local files
type FileSystemResolver struct {
	config    ResolverConfig
	registry  map[string]Overlay
	cache     map[string]*ResolverResult
	cacheMux  sync.RWMutex
	overlayMux sync.RWMutex
}

// NewFileSystemResolver creates a new file system-based overlay resolver
func NewFileSystemResolver(config ResolverConfig) *FileSystemResolver {
	if config.OverlayPaths == nil {
		config.OverlayPaths = []string{"./overlays"}
	}
	if config.ConflictResolution == "" {
		config.ConflictResolution = "priority"
	}
	if config.CacheTTL == 0 {
		config.CacheTTL = 300 // 5 minutes default
	}

	return &FileSystemResolver{
		config:   config,
		registry: make(map[string]Overlay),
		cache:    make(map[string]*ResolverResult),
	}
}

// LoadOverlay loads an overlay from the file system
func (r *FileSystemResolver) LoadOverlay(name string) (Overlay, error) {
	r.overlayMux.RLock()
	if overlay, exists := r.registry[name]; exists {
		r.overlayMux.RUnlock()
		return overlay, nil
	}
	r.overlayMux.RUnlock()

	// Try to load from file system
	for _, overlayPath := range r.config.OverlayPaths {
		overlayFile := filepath.Join(overlayPath, name+".yaml")
		if _, err := os.Stat(overlayFile); err == nil {
			overlay, err := r.loadOverlayFromFile(overlayFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load overlay %s: %w", name, err)
			}

			// Cache the loaded overlay
			r.overlayMux.Lock()
			r.registry[name] = overlay
			r.overlayMux.Unlock()

			return overlay, nil
		}
	}

	return nil, fmt.Errorf("overlay not found: %s", name)
}

// loadOverlayFromFile loads an overlay from a YAML file
func (r *FileSystemResolver) loadOverlayFromFile(filePath string) (Overlay, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var spec OverlaySpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse overlay YAML: %w", err)
	}

	return NewYAMLOverlay(spec), nil
}

// ResolveOverlays resolves a list of overlay names into a coherent set
func (r *FileSystemResolver) ResolveOverlays(overlayNames []string, baseIR *ir.IRSpec) (*ResolverResult, error) {
	// Check cache first
	if r.config.EnableCaching {
		cacheKey := r.buildCacheKey(overlayNames, baseIR)
		if result := r.getCachedResult(cacheKey); result != nil {
			return result, nil
		}
	}

	// Load all requested overlays
	overlays := make(map[string]Overlay)
	for _, name := range overlayNames {
		overlay, err := r.LoadOverlay(name)
		if err != nil {
			return nil, fmt.Errorf("failed to load overlay %s: %w", name, err)
		}
		overlays[name] = overlay
	}

	// Resolve dependencies
	resolvedNames, err := r.resolveDependencies(overlayNames, overlays)
	if err != nil {
		return nil, fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Sort by priority while preserving dependency order
	sortedOverlays := r.sortByPriorityWithDependencies(resolvedNames, overlays)

	// Apply overlays to IR
	resultIR := r.deepCopyIR(baseIR)
	conflicts := []ConflictInfo{}
	promptChanges := make(map[string][]PromptEnhancement)
	validationRules := []ValidationRule{}
	metadata := make(map[string]OverlayMetadata)

	for _, overlayName := range sortedOverlays {
		overlay := overlays[overlayName]

		// Apply IR modifications
		modifiedIR, err := overlay.ApplyToIR(resultIR)
		if err != nil {
			return nil, fmt.Errorf("failed to apply overlay %s: %w", overlayName, err)
		}
		resultIR = modifiedIR

		// Collect prompt enhancements
		for _, enhancement := range overlay.GetPromptEnhancements() {
			agentType := enhancement.AgentType
			if promptChanges[agentType] == nil {
				promptChanges[agentType] = []PromptEnhancement{}
			}
			promptChanges[agentType] = append(promptChanges[agentType], enhancement)
		}

		// Collect validation rules
		validationRules = append(validationRules, overlay.GetValidationRules()...)

		// Store metadata
		metadata[overlayName] = overlay.GetMetadata()
	}

	result := &ResolverResult{
		ResolvedOverlays: resolvedNames,
		AppliedOrder:     sortedOverlays,
		Conflicts:        conflicts,
		IRSpec:           resultIR,
		PromptChanges:    promptChanges,
		ValidationRules:  validationRules,
		Metadata:         metadata,
	}

	// Cache result if enabled
	if r.config.EnableCaching {
		cacheKey := r.buildCacheKey(overlayNames, baseIR)
		r.setCachedResult(cacheKey, result)
	}

	return result, nil
}

// resolveDependencies resolves overlay dependencies recursively
func (r *FileSystemResolver) resolveDependencies(overlayNames []string, overlays map[string]Overlay) ([]string, error) {
	resolved := make(map[string]bool)
	var result []string
	var visiting = make(map[string]bool)

	var resolve func(name string) error
	resolve = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("circular dependency detected: %s", name)
		}
		if resolved[name] {
			return nil
		}

		visiting[name] = true

		overlay, exists := overlays[name]
		if !exists {
			// Try to load dependency
			loadedOverlay, err := r.LoadOverlay(name)
			if err != nil {
				return fmt.Errorf("dependency not found: %s", name)
			}
			overlay = loadedOverlay
			overlays[name] = overlay
		}

		// Resolve dependencies first
		for _, dep := range overlay.GetDependencies() {
			if err := resolve(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		resolved[name] = true
		result = append(result, name)
		return nil
	}

	// Resolve all requested overlays
	for _, name := range overlayNames {
		if err := resolve(name); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// sortByPriority sorts overlays by priority (higher priority applied last)
func (r *FileSystemResolver) sortByPriority(overlayNames []string, overlays map[string]Overlay) []string {
	type overlayInfo struct {
		name     string
		priority Priority
	}

	infos := make([]overlayInfo, len(overlayNames))
	for i, name := range overlayNames {
		infos[i] = overlayInfo{
			name:     name,
			priority: overlays[name].GetMetadata().Priority,
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].priority < infos[j].priority
	})

	result := make([]string, len(infos))
	for i, info := range infos {
		result[i] = info.name
	}

	return result
}

// sortByPriorityWithDependencies sorts overlays by priority while preserving dependency order
func (r *FileSystemResolver) sortByPriorityWithDependencies(overlayNames []string, overlays map[string]Overlay) []string {
	// Since overlayNames comes from dependency resolution, it's already in correct dependency order
	// We need to check if there are any dependencies, and if not, sort by priority
	hasDependencies := false
	for _, name := range overlayNames {
		if len(overlays[name].GetDependencies()) > 0 {
			hasDependencies = true
			break
		}
	}

	// If no dependencies, sort by priority
	if !hasDependencies {
		return r.sortByPriority(overlayNames, overlays)
	}

	// If there are dependencies, preserve dependency order
	return overlayNames
}

// ValidateOverlays checks overlay compatibility and dependencies
func (r *FileSystemResolver) ValidateOverlays(overlayNames []string) error {
	overlays := make(map[string]Overlay)
	for _, name := range overlayNames {
		overlay, err := r.LoadOverlay(name)
		if err != nil {
			return fmt.Errorf("validation failed for %s: %w", name, err)
		}
		overlays[name] = overlay

		// Validate individual overlay
		if err := overlay.Validate(); err != nil {
			return fmt.Errorf("overlay %s is invalid: %w", name, err)
		}
	}

	// Check dependencies
	_, err := r.resolveDependencies(overlayNames, overlays)
	if err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	return nil
}

// ListAvailable returns available overlays from all overlay paths
func (r *FileSystemResolver) ListAvailable() ([]OverlayMetadata, error) {
	var metadata []OverlayMetadata

	for _, overlayPath := range r.config.OverlayPaths {
		files, err := filepath.Glob(filepath.Join(overlayPath, "*.yaml"))
		if err != nil {
			continue
		}

		for _, file := range files {
			overlay, err := r.loadOverlayFromFile(file)
			if err != nil {
				continue // Skip invalid overlays
			}
			metadata = append(metadata, overlay.GetMetadata())
		}
	}

	return metadata, nil
}

// GetConflicts identifies potential conflicts between overlays
func (r *FileSystemResolver) GetConflicts(overlayNames []string) ([]ConflictInfo, error) {
	// This is a simplified implementation
	// In a full implementation, this would analyze IR modifications and prompt changes
	// to detect actual conflicts
	return []ConflictInfo{}, nil
}

// Helper methods for caching
func (r *FileSystemResolver) buildCacheKey(overlayNames []string, baseIR *ir.IRSpec) string {
	// Create a simple cache key based on overlay names and IR hash
	sort.Strings(overlayNames)
	overlaysStr := strings.Join(overlayNames, ",")

	// Simple hash of IR (in a real implementation, use proper hashing)
	irData, _ := json.Marshal(baseIR)
	irHash := fmt.Sprintf("%x", len(irData))

	return fmt.Sprintf("%s:%s", overlaysStr, irHash)
}

func (r *FileSystemResolver) getCachedResult(key string) *ResolverResult {
	r.cacheMux.RLock()
	defer r.cacheMux.RUnlock()

	result, exists := r.cache[key]
	if !exists {
		return nil
	}

	// Check TTL (simplified - in real implementation, store timestamp)
	return result
}

func (r *FileSystemResolver) setCachedResult(key string, result *ResolverResult) {
	r.cacheMux.Lock()
	defer r.cacheMux.Unlock()
	r.cache[key] = result
}

// deepCopyIR creates a deep copy of an IR specification
func (r *FileSystemResolver) deepCopyIR(original *ir.IRSpec) *ir.IRSpec {
	// Simple implementation - in production, use a proper deep copy library
	data, _ := json.Marshal(original)
	var copy ir.IRSpec
	json.Unmarshal(data, &copy)
	return &copy
}

// YAMLOverlay implements the Overlay interface for YAML-defined overlays
type YAMLOverlay struct {
	spec OverlaySpec
}

// NewYAMLOverlay creates a new YAML-based overlay
func NewYAMLOverlay(spec OverlaySpec) *YAMLOverlay {
	return &YAMLOverlay{spec: spec}
}

// GetMetadata returns overlay metadata
func (o *YAMLOverlay) GetMetadata() OverlayMetadata {
	return o.spec.Metadata
}

// ApplyToIR applies overlay modifications to an IR specification
func (o *YAMLOverlay) ApplyToIR(spec *ir.IRSpec) (*ir.IRSpec, error) {
	// Apply each IR modification
	for _, mod := range o.spec.IRModifications {
		if err := o.applyIRModification(spec, mod); err != nil {
			return nil, fmt.Errorf("failed to apply modification at %s: %w", mod.Path, err)
		}
	}
	return spec, nil
}

// applyIRModification applies a single IR modification
func (o *YAMLOverlay) applyIRModification(spec *ir.IRSpec, mod IRModification) error {
	// Simplified implementation - in production, use JSONPath or similar
	switch mod.Path {
	case "app.features":
		if mod.Operation == "add" && mod.Value != nil {
			if features, ok := mod.Value.([]interface{}); ok {
				for _, feature := range features {
					if featureStr, ok := feature.(string); ok {
						// Create Feature struct from string
						newFeature := ir.Feature{
							Name:        featureStr,
							Description: "Added by overlay",
							Type:        "overlay",
							Priority:    "medium",
						}
						spec.App.Features = append(spec.App.Features, newFeature)
					}
				}
			}
		}
	case "app.stack.database.type":
		if mod.Operation == "replace" && mod.Value != nil {
			if dbStr, ok := mod.Value.(string); ok {
				spec.App.Stack.Database.Type = dbStr
			}
		}
	}
	return nil
}

// GetPromptEnhancements returns prompt modifications
func (o *YAMLOverlay) GetPromptEnhancements() []PromptEnhancement {
	return o.spec.PromptEnhancements
}

// GetValidationRules returns validation rules
func (o *YAMLOverlay) GetValidationRules() []ValidationRule {
	return o.spec.ValidationRules
}

// GetDependencies returns overlay dependencies
func (o *YAMLOverlay) GetDependencies() []string {
	return o.spec.Dependencies
}

// Validate checks if the overlay is valid
func (o *YAMLOverlay) Validate() error {
	if o.spec.Metadata.Name == "" {
		return fmt.Errorf("overlay name is required")
	}
	if o.spec.Metadata.Version == "" {
		return fmt.Errorf("overlay version is required")
	}
	if o.spec.Metadata.Type == "" {
		return fmt.Errorf("overlay type is required")
	}
	return nil
}