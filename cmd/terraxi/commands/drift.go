package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDriftCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "drift [provider]",
		Short: "Detect infrastructure drift (compare cloud state vs Terraform files)",
		Long: `Compare discovered cloud resources against existing .tf files
and report what is unmanaged, modified, or deleted.

Examples:
  terraxi drift aws --region us-east-1 --state ./terraform.tfstate
  terraxi drift aws --region us-east-1 --tf-dir ./infrastructure`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("drift detection is not yet implemented (planned for v0.2)")
		},
	}

	cmd.Flags().String("state", "", "Path to terraform.tfstate file")
	cmd.Flags().String("tf-dir", ".", "Directory containing .tf files")
	cmd.Flags().String("region", "", "Cloud region to scan")

	return cmd
}
