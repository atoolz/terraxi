package commands

import (
	"github.com/spf13/cobra"
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

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newDriftCmd())

	return cmd
}
