package codegen

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"

	"github.com/atoolz/terraxi/internal/codegen/hclutil"
	"github.com/atoolz/terraxi/internal/graph"
	"github.com/atoolz/terraxi/pkg/types"
)

// PostProcessor transforms raw generated HCL into production-quality code.
// This is Terraxi's core differentiator.
type PostProcessor struct {
	graph   *graph.DependencyGraph
	idIndex *IDIndex
}

// NewPostProcessor creates a new post-processor with a dependency graph and ID index.
func NewPostProcessor(g *graph.DependencyGraph, idx *IDIndex) *PostProcessor {
	return &PostProcessor{graph: g, idIndex: idx}
}

// Process takes raw generated HCL content and applies transformations.
// Pass 1: Replace hardcoded resource IDs with Terraform references.
// Pass 2: Extract common values into variables.
func (pp *PostProcessor) Process(rawHCL []byte, resources []types.Resource) ([]byte, error) {
	if len(rawHCL) == 0 || pp.idIndex == nil {
		return rawHCL, nil
	}

	f, err := hclutil.ParseFile(rawHCL)
	if err != nil {
		slog.Warn("Failed to parse generated HCL for post-processing, returning raw", "error", err)
		return rawHCL, nil
	}

	// Pass 1: Replace hardcoded IDs with references
	pp.replaceIDs(f)

	// Pass 2: Collapse repeated resources into for_each
	CollapseForEach(f)

	return hclutil.FormatFile(f), nil
}

// replaceIDs walks every resource block and replaces string literals
// that match discovered resource IDs with Terraform reference expressions.
func (pp *PostProcessor) replaceIDs(f *hclwrite.File) {
	for _, block := range f.Body().Blocks() {
		if block.Type() != "resource" {
			continue
		}
		pp.replaceInBody(block.Body())
	}
}

// replaceInBody walks attributes and replaces string values that match
// discovered resource IDs with Terraform traversal references.
func (pp *PostProcessor) replaceInBody(body *hclwrite.Body) {
	for name, attr := range body.Attributes() {
		// Extract the raw string value from the attribute
		tokens := attr.Expr().BuildTokens(nil)
		val := extractStringValue(tokens)
		if val == "" {
			continue
		}

		// Check if this value matches a discovered resource ID
		ref, ok := pp.idIndex.Lookup(val)
		if !ok {
			continue
		}

		// Replace the entire attribute with a traversal expression
		slog.Debug("Replacing ID with reference", "attribute", name, "id", val, "ref", ref)
		body.SetAttributeRaw(name, rawRefTokens(ref))
	}

	// Recurse into nested blocks
	for _, nested := range body.Blocks() {
		pp.replaceInBody(nested.Body())
	}
}

// extractStringValue extracts a plain string value from HCL tokens.
// Returns empty string if the expression is not a simple quoted string.
func extractStringValue(tokens hclwrite.Tokens) string {
	// A simple quoted string in hclwrite tokens looks like:
	// [TokenOQuote][TokenQuotedLit "value"][TokenCQuote]
	// We want the middle token's bytes.
	var parts []string
	for _, tok := range tokens {
		b := string(tok.Bytes)
		// Skip quotes, newlines, and template markers
		if b == `"` || b == "" || b == "\n" {
			continue
		}
		parts = append(parts, b)
	}
	if len(parts) != 1 {
		return "" // Not a simple single-value string
	}
	val := parts[0]
	if !looksLikeAWSID(val) {
		return ""
	}
	return val
}

// rawRefTokens creates raw HCL tokens for a reference expression like "aws_vpc.main.id".
// This is used with SetAttributeRaw to replace a quoted string with an unquoted traversal.
func rawRefTokens(ref string) hclwrite.Tokens {
	return hclwrite.Tokens{
		{
			Type:         hclsyntax.TokenIdent,
			Bytes:        []byte(ref),
			SpacesBefore: 1,
		},
		{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		},
	}
}

// looksLikeAWSID returns true if the string matches common AWS resource ID patterns.
func looksLikeAWSID(s string) bool {
	prefixes := []string{
		"vpc-", "subnet-", "sg-", "igw-", "nat-", "rtb-", "eipalloc-",
		"i-", "vol-", "snap-", "ami-", "key-", "eni-",
		"arn:aws:", "arn:partition:",
	}
	for _, p := range prefixes {
		if len(s) > len(p) && s[:len(p)] == p {
			return true
		}
	}
	return false
}

// SplitByService writes HCL blocks into per-service subdirectories.
// Output structure:
//
//	outputDir/
//	  ec2/main.tf
//	  vpc/main.tf
//	  s3/main.tf
//	  ...
func (pp *PostProcessor) SplitByService(f *hclwrite.File, outputDir string) error {
	byService := make(map[string]*hclwrite.File)

	for _, block := range f.Body().Blocks() {
		if block.Type() != "resource" || len(block.Labels()) < 1 {
			continue
		}
		resourceType := block.Labels()[0]
		service := ServiceFromResourceType(resourceType)

		// Sanitize service name to prevent path traversal
		if !isValidServiceName(service) {
			slog.Warn("Skipping block with invalid service name", "service", service, "type", resourceType)
			continue
		}

		if _, exists := byService[service]; !exists {
			byService[service] = hclwrite.NewEmptyFile()
		}

		byService[service].Body().AppendNewline()
		byService[service].Body().AppendBlock(block)
	}

	for service, sf := range byService {
		dir := filepath.Join(outputDir, service)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating service directory %s: %w", service, err)
		}
		path := filepath.Join(dir, "main.tf")
		if err := os.WriteFile(path, hclutil.FormatFile(sf), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		slog.Debug("Wrote service module", "service", service, "path", path)
	}

	return nil
}

// ExtractVariables scans HCL for values that should become variables.
// Currently extracts: region, account ID (values appearing in 3+ resource blocks).
func (pp *PostProcessor) ExtractVariables(f *hclwrite.File, region string) []byte {
	var vars bytes.Buffer

	if region != "" {
		_, _ = fmt.Fprintf(&vars, `variable "region" {
  description = "AWS region"
  type        = string
  default     = %q
}
`, region)
	}

	return vars.Bytes()
}

// OrganizeByService splits resources into service-based groups.
func (pp *PostProcessor) OrganizeByService(resources []types.Resource) map[string][]types.Resource {
	byService := make(map[string][]types.Resource)
	for _, r := range resources {
		service := ServiceFromResourceType(r.Type)
		byService[service] = append(byService[service], r)
	}
	return byService
}

// isValidServiceName checks that a service name is safe for use as a directory name.
func isValidServiceName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		isLower := c >= 'a' && c <= 'z'
		isDigit := c >= '0' && c <= '9'
		if !isLower && !isDigit && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

// serviceRegistry maps resource types to service names.
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
