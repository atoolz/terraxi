package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/atoolz/terraxi/internal/discovery"
	"github.com/atoolz/terraxi/internal/drift"
	"github.com/atoolz/terraxi/pkg/types"
)

type driftOpts struct {
	region      string
	profile     string
	state       string
	report      string
	format      string
	concurrency int
}

func newDriftCmd() *cobra.Command {
	opts := &driftOpts{}

	cmd := &cobra.Command{
		Use:   "drift [provider]",
		Short: "Detect infrastructure drift (compare cloud state vs Terraform files)",
		Long: `Compare discovered cloud resources against Terraform state
and report what is unmanaged, modified, or deleted.

Examples:
  terraxi drift aws --region us-east-1 --state ./terraform.tfstate
  terraxi drift aws --region us-east-1 --state ./terraform.tfstate --format json
  terraxi drift aws --region us-east-1 --state ./terraform.tfstate --report drift.html`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDrift(cmd.Context(), args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.region, "region", "", "Cloud region to scan (required)")
	cmd.Flags().StringVar(&opts.profile, "profile", "", "AWS profile to use")
	cmd.Flags().StringVar(&opts.state, "state", "", "Path to terraform.tfstate file (required)")
	cmd.Flags().StringVar(&opts.report, "report", "", "Write HTML drift report to this file")
	cmd.Flags().StringVar(&opts.format, "format", "table", "Output format: table or json")
	cmd.Flags().IntVar(&opts.concurrency, "concurrency", 10, "Max concurrent API calls")

	_ = cmd.MarkFlagRequired("region")
	_ = cmd.MarkFlagRequired("state")

	return cmd
}

func runDrift(ctx context.Context, providerName string, opts *driftOpts) error {
	// Parse state file
	stateResources, err := drift.ReadState(opts.state)
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}
	slog.Info("State loaded", "resources", len(stateResources), "path", opts.state)

	// Run discovery
	provider, err := getProvider(providerName)
	if err != nil {
		return err
	}

	cfg := discovery.ProviderConfig{Region: opts.region, Profile: opts.profile}
	if err := provider.Configure(ctx, cfg); err != nil {
		return fmt.Errorf("failed to configure %s provider: %w", providerName, err)
	}

	eng := discovery.NewEngine(provider, opts.concurrency)
	slog.Info("Starting discovery for drift analysis", "region", opts.region)

	result, err := eng.Run(ctx, types.Filter{})
	if err != nil {
		return fmt.Errorf("discovery failed: %w", err)
	}

	// Analyze drift
	report := drift.Analyze(result.Resources, stateResources)

	// Output
	if opts.format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Table output
	_, _ = fmt.Fprintf(os.Stdout, "\nDrift Report\n")
	_, _ = fmt.Fprintf(os.Stdout, "============\n\n")
	_, _ = fmt.Fprintf(os.Stdout, "Summary: %s\n\n", report.Summary())

	if len(report.Unmanaged) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Unmanaged Resources (%d):\n", len(report.Unmanaged))
		_, _ = fmt.Fprintf(os.Stdout, "  TYPE                          ID                      NAME\n")
		_, _ = fmt.Fprintf(os.Stdout, "  ----                          --                      ----\n")
		for _, r := range report.Unmanaged {
			name := r.Name
			if name == "" {
				name = "-"
			}
			_, _ = fmt.Fprintf(os.Stdout, "  %-30s %-24s %s\n", r.Type, r.ID, name)
		}
		_, _ = fmt.Fprintln(os.Stdout)
	}

	if len(report.Deleted) > 0 {
		_, _ = fmt.Fprintf(os.Stdout, "Deleted Resources (%d):\n", len(report.Deleted))
		_, _ = fmt.Fprintf(os.Stdout, "  TYPE                          ADDRESS                 ID\n")
		_, _ = fmt.Fprintf(os.Stdout, "  ----                          -------                 --\n")
		for _, r := range report.Deleted {
			_, _ = fmt.Fprintf(os.Stdout, "  %-30s %-24s %s\n", r.Type, r.Address, r.ID)
		}
		_, _ = fmt.Fprintln(os.Stdout)
	}

	if !report.HasDrift() {
		_, _ = fmt.Fprintf(os.Stdout, "No drift detected. All resources are managed.\n")
	}

	// Write HTML report if requested
	if opts.report != "" {
		htmlBytes, err := drift.RenderHTML(report)
		if err != nil {
			return fmt.Errorf("failed to render HTML report: %w", err)
		}
		if err := os.WriteFile(opts.report, htmlBytes, 0644); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		slog.Info("HTML report written", "path", opts.report)
	}

	// Exit with error if drift detected (CI-friendly)
	if report.HasDrift() {
		return fmt.Errorf("drift detected: %s", report.Summary())
	}

	return nil
}
