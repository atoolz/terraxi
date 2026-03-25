package drift

import (
	"encoding/json"
	"fmt"
	"os"
)

// StateResource represents a resource tracked in Terraform state.
type StateResource struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Address string `json:"address"` // e.g., "aws_vpc.main"
	ID      string `json:"id"`
}

// stateFile is the JSON structure of a terraform.tfstate file (schema v4).
type stateFile struct {
	Version   int          `json:"version"`
	Resources []stateEntry `json:"resources"`
}

type stateEntry struct {
	Mode      string          `json:"mode"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Instances []stateInstance `json:"instances"`
}

type stateInstance struct {
	IndexKey       interface{}       `json:"index_key"`
	Attributes     json.RawMessage   `json:"attributes"`
	AttributesFlat map[string]string `json:"attributes_flat"`
}

// ReadState parses a terraform.tfstate file and returns all managed resources.
func ReadState(path string) ([]StateResource, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	return ParseState(data)
}

// ParseState parses terraform state JSON bytes into StateResources.
func ParseState(data []byte) ([]StateResource, error) {
	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parsing state JSON: %w", err)
	}

	if sf.Version != 4 {
		return nil, fmt.Errorf("unsupported state version %d (expected 4)", sf.Version)
	}

	var resources []StateResource
	for _, entry := range sf.Resources {
		if entry.Mode != "managed" {
			continue // skip data sources
		}

		for _, inst := range entry.Instances {
			id := extractIDFromAttributes(inst.Attributes)
			addr := fmt.Sprintf("%s.%s", entry.Type, entry.Name)
			if inst.IndexKey != nil {
				switch k := inst.IndexKey.(type) {
				case string:
					addr = fmt.Sprintf("%s.%s[%q]", entry.Type, entry.Name, k)
				default:
					addr = fmt.Sprintf("%s.%s[%v]", entry.Type, entry.Name, inst.IndexKey)
				}
			}
			resources = append(resources, StateResource{
				Type:    entry.Type,
				Name:    entry.Name,
				Address: addr,
				ID:      id,
			})
		}
	}

	return resources, nil
}

// extractIDFromAttributes pulls the "id" field from resource attributes JSON.
func extractIDFromAttributes(attrs json.RawMessage) string {
	if len(attrs) == 0 {
		return ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal(attrs, &m); err != nil {
		return ""
	}
	if id, ok := m["id"].(string); ok {
		return id
	}
	return ""
}

// StateIndex builds a lookup map from resource ID to StateResource.
func StateIndex(resources []StateResource) map[string]StateResource {
	idx := make(map[string]StateResource, len(resources))
	for _, r := range resources {
		key := r.Type + ":" + r.ID
		idx[key] = r
	}
	return idx
}
