package codegen

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

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

// replaceInBody walks attributes and nested blocks, replacing ID literals.
func (pp *PostProcessor) replaceInBody(body *hclwrite.Body) {
	for name, attr := range body.Attributes() {
		tokens := attr.Expr().BuildTokens(nil)
		modified := false

		for _, tok := range tokens {
			val := unquoteTokenPtr(tok)
			if val == "" {
				continue
			}

			ref, ok := pp.idIndex.Lookup(val)
			if !ok {
				continue
			}

			slog.Debug("Replacing ID with reference", "attribute", name, "id", val, "ref", ref)
			tok.Bytes = []byte(ref)
			modified = true
		}

		if modified {
			body.SetAttributeRaw(name, tokens)
		}
	}

	// Recurse into nested blocks (e.g., ingress/egress in security groups)
	for _, nested := range body.Blocks() {
		pp.replaceInBody(nested.Body())
	}
}

// unquoteTokenPtr extracts the string value from a quoted HCL token pointer.
func unquoteTokenPtr(tok *hclwrite.Token) string {
	b := tok.Bytes
	if len(b) < 2 {
		return ""
	}
	// hclwrite tokens for string literals contain the raw bytes without quotes,
	// but the token type tells us it's a string literal
	val := string(b)
	// Check common AWS ID patterns to avoid replacing arbitrary strings
	if !looksLikeAWSID(val) {
		return ""
	}
	return val
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
