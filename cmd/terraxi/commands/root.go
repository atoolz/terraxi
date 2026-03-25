package commands

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	var verbose, debug bool

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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
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
		},
	}

	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable info-level logging")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug-level logging")

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newDriftCmd())

	return cmd
}
