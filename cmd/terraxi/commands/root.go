package commands

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	debug   bool
)

func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "terraxi",
		Short: "Discover cloud resources. Generate production-quality Terraform/OpenTofu code.",
		Long: `Terraxi auto-discovers your cloud infrastructure and generates
production-quality Terraform or OpenTofu code.

It delegates HCL generation to terraform/tofu import and then
post-processes the output into clean, modular code with proper
variables, references, and structure.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable info-level logging")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug-level logging")

	cobra.OnInitialize(initLogging)

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newDriftCmd())

	return cmd
}

func initLogging() {
	level := slog.LevelWarn
	if verbose {
		level = slog.LevelInfo
	}
	if debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}
