package discovery

import (
	"fmt"
	"strings"

	"github.com/ahlert/terraxi/pkg/types"
)

// ParseFilter parses a filter string into a Filter struct.
// Format: "key=value AND key2=value2"
// Supported keys: type, service, tags.KEY
// Example: "tags.env=production AND service=ec2"
func ParseFilter(raw string) (types.Filter, error) {
	filter := types.Filter{
		Tags: make(map[string]string),
	}

	if raw == "" {
		return filter, nil
	}

	parts := strings.Split(raw, " AND ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return filter, fmt.Errorf("invalid filter expression: %q (expected key=value)", part)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch {
		case key == "type":
			filter.Types = append(filter.Types, value)
		case key == "service":
			filter.Services = append(filter.Services, value)
		case strings.HasPrefix(key, "tags."):
			tagKey := strings.TrimPrefix(key, "tags.")
			filter.Tags[tagKey] = value
		case key == "exclude":
			filter.Exclude = append(filter.Exclude, value)
		default:
			return filter, fmt.Errorf("unknown filter key: %q", key)
		}
	}

	return filter, nil
}

// MatchesTags returns true if the resource matches all tag filters.
func MatchesTags(resource types.Resource, tags map[string]string) bool {
	for k, v := range tags {
		if resource.Tags[k] != v {
			return false
		}
	}
	return true
}
