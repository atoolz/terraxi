package codegen

import (
	"sync"

	"github.com/ahlert/terraxi/internal/graph"
	"github.com/ahlert/terraxi/pkg/types"
)

// PostProcessor transforms raw generated HCL into production-quality code.
// This is Terraxi's core differentiator.
type PostProcessor struct {
	graph *graph.DependencyGraph
}

// NewPostProcessor creates a new post-processor with a dependency graph.
func NewPostProcessor(g *graph.DependencyGraph) *PostProcessor {
	return &PostProcessor{graph: g}
}

// Process takes raw generated HCL content and transforms it.
// Transformations applied:
// 1. Replace hardcoded resource IDs with references (e.g., aws_vpc.main.id)
// 2. Extract common values into variables (region, tags, etc.)
// 3. Collapse similar resources into for_each blocks
// 4. Organize into module structure (one dir per service)
// 5. Generate outputs.tf for cross-module references
func (pp *PostProcessor) Process(rawHCL []byte, resources []types.Resource) ([]byte, error) {
	// TODO: Implement using hclwrite
	// Phase 1: Parse raw HCL into hclwrite.File
	// Phase 2: Walk all attribute expressions, find hardcoded IDs that match discovered resources
	// Phase 3: Replace with references (e.g., "vpc-123abc" -> aws_vpc.main.id)
	// Phase 4: Extract repeated values into variables
	// Phase 5: Detect groups of similar resources and convert to for_each
	return rawHCL, nil
}

// OrganizeByService splits resources into service-based groups.
// Uses the service mapping registered via RegisterServiceMapping.
// Resource types not registered are grouped under the key "other".
func (pp *PostProcessor) OrganizeByService(resources []types.Resource) map[string][]types.Resource {
	byService := make(map[string][]types.Resource)
	for _, r := range resources {
		service := ServiceFromResourceType(r.Type)
		byService[service] = append(byService[service], r)
	}
	return byService
}

// serviceRegistry maps resource types to service names.
// Populated from the provider's ListResourceTypes at runtime.
var (
	serviceRegistry   = make(map[string]string)
	serviceRegistryMu sync.RWMutex
)

// RegisterServiceMapping registers which service a resource type belongs to.
func RegisterServiceMapping(resourceType, service string) {
	serviceRegistryMu.Lock()
	defer serviceRegistryMu.Unlock()
	serviceRegistry[resourceType] = service
}

// ServiceFromResourceType returns the service name for a resource type.
// Falls back to "other" if not registered.
func ServiceFromResourceType(resourceType string) string {
	serviceRegistryMu.RLock()
	defer serviceRegistryMu.RUnlock()
	if service, ok := serviceRegistry[resourceType]; ok {
		return service
	}
	return "other"
}
