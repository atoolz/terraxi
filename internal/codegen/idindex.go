package codegen

import (
	"github.com/atoolz/terraxi/pkg/types"
)

// IDIndex maps discovered resource IDs to their Terraform addresses.
// Used by the post-processor to replace hardcoded IDs with references.
//
// Example: "vpc-abc123" -> "aws_vpc.main.id"
type IDIndex struct {
	// idToAddress maps resource ID -> terraform address (e.g., "aws_vpc.main")
	idToAddress map[string]string
}

// NewIDIndex builds an ID-to-address mapping from discovered resources.
// It uses the provided NameResolver to generate the same Terraform names
// that were used in the import blocks, guaranteeing consistency.
func NewIDIndex(resources []types.Resource, names *NameResolver) *IDIndex {
	idx := &IDIndex{
		idToAddress: make(map[string]string, len(resources)),
	}
	for _, r := range resources {
		tfName := names.Resolve(r)
		address := r.Type + "." + tfName
		idx.idToAddress[r.ID] = address
	}
	return idx
}

// Lookup returns the Terraform address for a resource ID.
// Returns (address, true) if found, ("", false) otherwise.
// The address includes ".id" suffix for use in reference expressions.
func (idx *IDIndex) Lookup(resourceID string) (string, bool) {
	addr, ok := idx.idToAddress[resourceID]
	if !ok {
		return "", false
	}
	return addr + ".id", true
}

// LookupAddress returns the Terraform address without the ".id" suffix.
func (idx *IDIndex) LookupAddress(resourceID string) (string, bool) {
	addr, ok := idx.idToAddress[resourceID]
	return addr, ok
}

// Len returns the number of indexed resources.
func (idx *IDIndex) Len() int {
	return len(idx.idToAddress)
}
