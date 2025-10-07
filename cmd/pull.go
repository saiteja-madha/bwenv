package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pullCmd represents the pull command
var pullCmd = &cobra.Command{
	Use:   "pull <app>",
	Short: "Pull environment variables for an app",
	Long: `Pull environment variables for the specified app from Bitwarden.

Output is in KEY=VALUE format suitable for sourcing or saving to a .env file.
Use --include-shared to also include shared secrets.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]

		lines, err := bwsClient.GetEnvLines(app, includeShared)
		if err != nil {
			return err
		}

		for _, line := range lines {
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pullCmd)
}
