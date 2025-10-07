package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list <app>",
	Short: "List keys for an app",
	Long: `List all environment variable keys for the specified app.

This shows only the key names (without the app prefix) that are stored
in Bitwarden for the given app.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]

		keys, err := bwsClient.GetAppKeys(app)
		if err != nil {
			return err
		}

		for _, key := range keys {
			fmt.Println(key)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}