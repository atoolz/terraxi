package gcp

import (
	"context"
	"fmt"

	"github.com/atoolz/terraxi/internal/codegen"
	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/pkg/types"
)

// Provider discovers GCP resources.
type Provider struct {
	project string
	region  string
}

// New creates a new GCP provider.
func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string { return "gcp" }

func (p *Provider) Configure(_ context.Context, cfg discovery.ProviderConfig) error {
	p.region = cfg.Region
	p.project = cfg.Extra["project"]
	if p.region == "" {
		return fmt.Errorf("GCP region is required: set --region")
	}
	if p.project == "" {
		return fmt.Errorf("GCP project is required: set --project or GOOGLE_CLOUD_PROJECT")
	}

	for _, rt := range p.ListResourceTypes() {
		codegen.RegisterServiceMapping(rt.Type, rt.Service)
	}

	return nil
}

func (p *Provider) ListResourceTypes() []types.ResourceType {
	return []types.ResourceType{
		{Type: "google_compute_instance", Service: "compute", Description: "Compute Engine instances"},
		{Type: "google_compute_network", Service: "compute", Description: "VPC networks"},
		{Type: "google_compute_subnetwork", Service: "compute", Description: "Subnetworks"},
		{Type: "google_storage_bucket", Service: "storage", Description: "Cloud Storage buckets"},
		{Type: "google_project_iam_member", Service: "iam", Description: "IAM policy members"},
	}
}

func (p *Provider) Discover(_ context.Context, resourceType string, _ types.Filter) ([]types.Resource, error) {
	// GCP discoverers are stubs for now. Real implementation requires
	// google.golang.org/api and google.golang.org/genproto packages.
	return nil, fmt.Errorf("GCP resource type %s is not yet implemented (planned for v1.1.0)", resourceType)
}
