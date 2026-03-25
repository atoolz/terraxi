package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/atoolz/terraxi/internal/codegen"
	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/internal/drift"
	"github.com/atoolz/terraxi/internal/graph"
	"github.com/atoolz/terraxi/internal/output"
	awsprovider "github.com/atoolz/terraxi/internal/providers/aws"
	"github.com/atoolz/terraxi/pkg/types"
)

type discoverOpts struct {
	region      string
	regions     []string
	profile     string
	services    []string
	filter      string
	outputDir   string
	engine      string
	structure   string
	inventory   string
	state       string
	skipManaged bool
	dryRun      bool
	format      string
	concurrency int
}

func newDiscoverCmd() *cobra.Command {
	opts := &discoverOpts{}

	cmd := &cobra.Command{
		Use:   "discover [provider]",
		Short: "Discover cloud resources and generate Terraform/OpenTofu code",
		Long: `Scan a cloud account for existing resources and generate
production-quality Terraform or OpenTofu configuration files.

Examples:
  terraxi discover aws --region us-east-1
  terraxi discover aws --services ec2,s3,iam --region us-east-1
  terraxi discover aws --filter "tags.env=production" --region us-east-1
  terraxi discover aws --dry-run --region us-east-1
  terraxi discover aws --engine tofu --output ./imported
  terraxi discover aws --region us-east-1 --structure modules
  terraxi discover aws --regions us-east-1,eu-west-1`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "Cloud region to scan")
	cmd.Flags().StringSliceVar(&opts.regions, "regions", nil, "Multiple regions to scan (comma-separated, e.g., us-east-1,eu-west-1)")
	cmd.Flags().StringVar(&opts.profile, "profile", "", "AWS profile to use")
	cmd.Flags().StringSliceVar(&opts.services, "services", nil, "Services to scan (comma-separated, e.g., ec2,s3,iam)")
	cmd.Flags().StringVar(&opts.filter, "filter", "", "Filter expression (e.g., \"tags.env=production\")")
	cmd.Flags().StringVarP(&opts.outputDir, "output", "o", "./imported", "Output directory for generated files")
	cmd.Flags().StringVar(&opts.engine, "engine", "terraform", "IaC engine: terraform or tofu")
	cmd.Flags().StringVar(&opts.structure, "structure", "flat", "Output structure: flat (single dir) or modules (per-service subdirs)")
	cmd.Flags().StringVar(&opts.inventory, "inventory", "", "Write discovery results as JSON to this file")
	cmd.Flags().StringVar(&opts.state, "state", "", "Path to terraform.tfstate (for --skip-managed)")
	cmd.Flags().BoolVar(&opts.skipManaged, "skip-managed", false, "Skip resources already in the state file (requires --state)")
	cmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "Preview what would be discovered without generating code")
	cmd.Flags().StringVar(&opts.format, "format", "table", "Output format: table or json")
	cmd.Flags().IntVar(&opts.concurrency, "concurrency", 10, "Max concurrent API calls")

	// region is required unless --regions is provided (validated in runDiscover)

	return cmd
}

func runDiscover(ctx context.Context, providerName string, opts *discoverOpts) error {
	// Resolve regions: --regions takes priority, falls back to --region
	regions := opts.regions
	if len(regions) == 0 {
		if opts.region == "" {
			return fmt.Errorf("--region or --regions is required")
		}
		regions = []string{opts.region}
	}
	if opts.region != "" && len(opts.regions) > 0 {
		slog.Warn("--region is ignored when --regions is provided", "ignored", opts.region)
	}

	filter, err := buildFilter(opts)
	if err != nil {
		return fmt.Errorf("invalid filter: %w", err)
	}

	// Run discovery across all regions, merging results
	mergedResult := &types.DiscoveryResult{Provider: providerName}
	seenIAM := make(map[string]bool) // deduplicate global resources across regions

	for _, region := range regions {
		provider, err := getProvider(providerName)
		if err != nil {
			return err
		}

		cfg := discovery.ProviderConfig{
			Region:  region,
			Profile: opts.profile,
		}
		if err := provider.Configure(ctx, cfg); err != nil {
			return fmt.Errorf("failed to configure %s provider for %s: %w", providerName, region, err)
		}

		eng := discovery.NewEngine(provider, opts.concurrency)
		slog.Info("Starting discovery", "provider", providerName, "region", region)

		result, err := eng.Run(ctx, filter)
		if err != nil {
			return fmt.Errorf("discovery failed in %s: %w", region, err)
		}

		for _, r := range result.Resources {
			// IAM resources are global (Region=""), deduplicate across regions
			if r.Region == "" {
				key := r.Type + ":" + r.ID
				if seenIAM[key] {
					continue
				}
				seenIAM[key] = true
			}
			mergedResult.Resources = append(mergedResult.Resources, r)
		}
		mergedResult.Errors = append(mergedResult.Errors, result.Errors...)
	}

	mergedResult.Region = strings.Join(regions, ",")

	// Filter out already-managed resources if --skip-managed is set
	if opts.skipManaged {
		if opts.state == "" {
			return fmt.Errorf("--skip-managed requires --state")
		}
		stateResources, stateErr := drift.ReadState(opts.state)
		if stateErr != nil {
			return fmt.Errorf("failed to read state for --skip-managed: %w", stateErr)
		}
		stateIdx := drift.StateIndex(stateResources)
		var filtered []types.Resource
		for _, r := range mergedResult.Resources {
			key := r.Type + ":" + r.ID
			if _, managed := stateIdx[key]; !managed {
				filtered = append(filtered, r)
			}
		}
		skipped := len(mergedResult.Resources) - len(filtered)
		slog.Info("Skipped managed resources", "skipped", skipped, "remaining", len(filtered))
		mergedResult.Resources = filtered
	}

	result := mergedResult

	// Write JSON inventory if requested
	if opts.inventory != "" {
		invFile, invErr := os.Create(opts.inventory)
		if invErr != nil {
			return fmt.Errorf("failed to create inventory file: %w", invErr)
		}
		invWriter := output.NewWriter(invFile, output.FormatJSON)
		if writeErr := invWriter.WriteResult(result); writeErr != nil {
			_ = invFile.Close()
			return fmt.Errorf("failed to write inventory: %w", writeErr)
		}
		_ = invFile.Close()
		slog.Info("Inventory written", "path", opts.inventory)
	}

	if opts.dryRun {
		return writeDryRunOutput(os.Stdout, result, opts.format)
	}

	writer := output.NewWriter(os.Stdout, output.Format(opts.format))
	if err := writer.WriteResult(result); err != nil {
		return err
	}

	if len(result.Resources) == 0 {
		return nil
	}

	// Build dependency graph
	depGraph := graph.New()
	depGraph.AddAll(result.Resources)

	// Generate HCL
	iacEngine := types.EngineTerraform
	if opts.engine == "tofu" {
		iacEngine = types.EngineOpenTofu
	}

	structure := codegen.Structure(opts.structure)
	genCfg := discovery.ProviderConfig{Region: regions[0], Profile: opts.profile}
	gen := codegen.NewGenerator(iacEngine, opts.outputDir, genCfg, depGraph, structure)
	if err := gen.GenerateAll(ctx, depGraph.TopologicalSort()); err != nil {
		return fmt.Errorf("code generation failed: %w", err)
	}

	slog.Info("Code generation complete", "output", opts.outputDir, "resources", len(result.Resources), "structure", opts.structure)
	return nil
}

// writeDryRunOutput prints a summary grouped by service.
func writeDryRunOutput(w *os.File, result *types.DiscoveryResult, format string) error {
	if format == "json" {
		writer := output.NewWriter(w, output.FormatJSON)
		return writer.WriteResult(result)
	}

	// Group by service for summary
	byService := make(map[string]int)
	byServiceTypes := make(map[string]map[string]bool)
	for _, r := range result.Resources {
		service := codegen.ServiceFromResourceType(r.Type)
		byService[service]++
		if byServiceTypes[service] == nil {
			byServiceTypes[service] = make(map[string]bool)
		}
		byServiceTypes[service][r.Type] = true
	}

	_, _ = fmt.Fprintf(w, "\nDry Run Summary\n")
	_, _ = fmt.Fprintf(w, "===============\n\n")
	_, _ = fmt.Fprintf(w, "SERVICE          COUNT   RESOURCE TYPES\n")
	_, _ = fmt.Fprintf(w, "-------          -----   --------------\n")

	// Sort services for deterministic output
	services := make([]string, 0, len(byService))
	for s := range byService {
		services = append(services, s)
	}
	sort.Strings(services)

	for _, service := range services {
		count := byService[service]
		typeList := make([]string, 0, len(byServiceTypes[service]))
		for t := range byServiceTypes[service] {
			typeList = append(typeList, t)
		}
		sort.Strings(typeList)
		_, _ = fmt.Fprintf(w, "%-16s %5d   %s\n", service, count, strings.Join(typeList, ", "))
	}

	_, _ = fmt.Fprintf(w, "\nTotal: %d resources discovered\n", len(result.Resources))

	if len(result.Errors) > 0 {
		_, _ = fmt.Fprintf(w, "\nErrors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			_, _ = fmt.Fprintf(w, "  - %s\n", e)
		}
		return fmt.Errorf("discovery completed with %d errors", len(result.Errors))
	}

	return nil
}

func getProvider(name string) (discovery.Provider, error) {
	switch name {
	case "aws":
		return awsprovider.New(), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: aws)", name)
	}
}

func buildFilter(opts *discoverOpts) (types.Filter, error) {
	var filter types.Filter
	var err error

	if opts.filter != "" {
		filter, err = discovery.ParseFilter(opts.filter)
		if err != nil {
			return filter, err
		}
	}

	if len(opts.services) > 0 {
		for _, s := range opts.services {
			filter.Services = append(filter.Services, strings.TrimSpace(s))
		}
	}

	return filter, nil
}
