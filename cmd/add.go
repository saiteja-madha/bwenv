package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <app> <key> <value>",
	Short: "Add a secret to the project",
	Long: `Add a secret to the Bitwarden project.

Usage:
  bwenv add <app> KEY VALUE
  bwenv add <app> KEY=VALUE`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]

		var key, value string
		if len(args) == 2 && strings.Contains(args[1], "=") {
			parts := strings.SplitN(args[1], "=", 2)
			key, value = parts[0], parts[1]
		} else if len(args) == 3 {
			key, value = args[1], args[2]
		} else {
			return fmt.Errorf("provide KEY and VALUE")
		}

		if key == "" || value == "" {
			return fmt.Errorf("provide KEY and VALUE")
		}

		fullName := fmt.Sprintf("%s__%s", app, key)
		return bwsClient.UpsertSecret(fullName, value, dryRun)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}