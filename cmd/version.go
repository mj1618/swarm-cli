package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/matt/swarm-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	versionShort  bool
	versionFormat string
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version number, commit hash, build date, and runtime information for swarm-cli.`,
	Example: `  # Show full version information
  swarm version

  # Show only version number
  swarm version --short

  # Output as JSON
  swarm version --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		info := version.GetInfo()

		if versionShort {
			fmt.Println(info.Version)
			return nil
		}

		if versionFormat == "json" {
			output, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal version info: %w", err)
			}
			fmt.Println(string(output))
			return nil
		}

		fmt.Println(info.String())
		return nil
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&versionShort, "short", "s", false, "Print only the version number")
	versionCmd.Flags().StringVar(&versionFormat, "format", "", "Output format: json or text (default)")
}
