package types

// Resource represents a discovered cloud resource.
type Resource struct {
	Type         string            `json:"type"`
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	Tags         map[string]string `json:"tags,omitempty"`
	Attributes   map[string]any    `json:"attributes,omitempty"`
	Dependencies []ResourceRef     `json:"dependencies,omitempty"`
}

// ResourceRef is a reference to another resource by type and ID.
type ResourceRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ResourceType describes a kind of cloud resource that can be discovered.
type ResourceType struct {
	Type        string `json:"type"`
	Service     string `json:"service"`
	Description string `json:"description"`
}

// DiscoveryResult holds the output of a discovery run.
type DiscoveryResult struct {
	Provider  string     `json:"provider"`
	Region    string     `json:"region"`
	Resources []Resource `json:"resources"`
	Errors    []string   `json:"errors,omitempty"`
}

// Filter defines criteria for selecting which resources to discover.
type Filter struct {
	Services []string          `json:"services,omitempty"`
	Types    []string          `json:"types,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
	Exclude  []string          `json:"exclude,omitempty"`
}

// Engine is the IaC engine to target (terraform or tofu).
type Engine string

const (
	EngineTerraform Engine = "terraform"
	EngineOpenTofu  Engine = "tofu"
)
