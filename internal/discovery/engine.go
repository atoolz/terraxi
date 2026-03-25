package discovery

import (
	"context"
	"fmt"
	"sync"

	"github.com/atoolz/terraxi/pkg/types"
)

// Engine orchestrates resource discovery across providers.
type Engine struct {
	provider    Provider
	concurrency int
}

// NewEngine creates a discovery engine for the given provider.
func NewEngine(provider Provider, concurrency int) *Engine {
	if concurrency <= 0 {
		concurrency = 10
	}
	return &Engine{
		provider:    provider,
		concurrency: concurrency,
	}
}

// Run discovers all resources matching the filter.
// It runs discovery for each resource type in parallel.
func (e *Engine) Run(ctx context.Context, filter types.Filter) (*types.DiscoveryResult, error) {
	allTypes := e.provider.ListResourceTypes()
	targetTypes := e.filterTypes(allTypes, filter)

	result := &types.DiscoveryResult{
		Provider: e.provider.Name(),
	}

	var (
		mu  sync.Mutex
		wg  sync.WaitGroup
		sem = make(chan struct{}, e.concurrency)
	)

	for _, rt := range targetTypes {
		wg.Add(1)
		go func(resourceType string) {
			defer wg.Done()

			// Context-aware semaphore: unblocks on cancellation
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", resourceType, ctx.Err()))
				mu.Unlock()
				return
			}

			resources, err := e.provider.Discover(ctx, resourceType, filter)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", resourceType, err))
				return
			}
			result.Resources = append(result.Resources, resources...)
		}(rt.Type)
	}

	wg.Wait()
	return result, nil
}

// filterTypes returns only the resource types that match the filter.
func (e *Engine) filterTypes(allTypes []types.ResourceType, filter types.Filter) []types.ResourceType {
	if len(filter.Services) == 0 && len(filter.Types) == 0 {
		return allTypes
	}

	serviceSet := make(map[string]bool, len(filter.Services))
	for _, s := range filter.Services {
		serviceSet[s] = true
	}

	typeSet := make(map[string]bool, len(filter.Types))
	for _, t := range filter.Types {
		typeSet[t] = true
	}

	excludeSet := make(map[string]bool, len(filter.Exclude))
	for _, ex := range filter.Exclude {
		excludeSet[ex] = true
	}

	var filtered []types.ResourceType
	for _, rt := range allTypes {
		if excludeSet[rt.Type] || excludeSet[rt.Service] {
			continue
		}
		serviceMatch := len(serviceSet) == 0 || serviceSet[rt.Service]
		typeMatch := len(typeSet) == 0 || typeSet[rt.Type]
		// OR logic: include if matches service filter OR type filter
		if !serviceMatch && !typeMatch {
			continue
		}
		filtered = append(filtered, rt)
	}
	return filtered
}
