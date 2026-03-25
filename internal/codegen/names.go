package codegen

import (
	"fmt"

	"github.com/atoolz/terraxi/pkg/types"
)

// NameResolver generates unique, valid Terraform resource names from resources.
// Generator and PostProcessor each create a fresh NameResolver and call Resolve
// over the same ordered resource slice. Because Resolve is deterministic,
// both produce identical names without sharing state.
type NameResolver struct {
	counts map[string]int
}

// NewNameResolver creates a new name resolver.
func NewNameResolver() *NameResolver {
	return &NameResolver{counts: make(map[string]int)}
}

// Resolve returns a unique Terraform-valid resource name for the given resource.
// Calling Resolve on the same resource sequence always produces the same names.
func (nr *NameResolver) Resolve(r types.Resource) string {
	base := sanitizeName(r.Name)
	if base == "" || base == "resource" {
		base = sanitizeName(r.ID)
	}

	nr.counts[base]++
	count := nr.counts[base]
	if count == 1 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, count)
}

// Reset clears all state for a fresh resolution pass.
func (nr *NameResolver) Reset() {
	nr.counts = make(map[string]int)
}

// sanitizeName converts a resource ID/name into a valid Terraform resource name.
func sanitizeName(name string) string {
	var result []byte
	for _, c := range []byte(name) {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
			result = append(result, c)
		case c == '-', c == '.', c == '/', c == ':':
			result = append(result, '_')
		}
	}
	if len(result) > 0 && result[0] >= '0' && result[0] <= '9' {
		result = append([]byte{'r', '_'}, result...)
	}
	if len(result) == 0 {
		return "resource"
	}
	return string(result)
}
