package discovery

import (
	"context"

	"github.com/atoolz/terraxi/pkg/types"
)

// Provider is the interface that cloud providers must implement.
// Each provider knows how to discover resources in its cloud.
type Provider interface {
	// Name returns the provider identifier (e.g., "aws", "gcp", "azure").
	Name() string

	// Configure initializes the provider with credentials and options.
	Configure(ctx context.Context, cfg ProviderConfig) error

	// ListResourceTypes returns all resource types this provider can discover.
	ListResourceTypes() []types.ResourceType

	// Discover finds all resources of a given type, applying filters.
	Discover(ctx context.Context, resourceType string, filter types.Filter) ([]types.Resource, error)
}

// ProviderConfig holds configuration for a cloud provider.
type ProviderConfig struct {
	Region     string            `json:"region"`
	Profile    string            `json:"profile,omitempty"`
	AssumeRole string            `json:"assume_role,omitempty"`
	ExternalID string            `json:"external_id,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
}
