package cmd

import (
	"fmt"
	"os"

	"bwenv/internal/bws"

	"github.com/spf13/cobra"
)

var (
	projectID     string
	dryRun        bool
	includeShared bool
	bwsClient     *bws.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bwenv",
	Short: "Bitwarden Secrets Manager helper for local dev",
	Long: `bwenv - Bitwarden Secrets Manager helper for local dev

Prefix convention: <app>__KEY  (e.g., notes-api__DATABASE_URL)

Global env:
  BWS_ACCESS_TOKEN   (required) - Bitwarden Secrets Manager machine token  
  BWS_PROJECT_ID     (default project UUID; can be overridden by --project-id)

Requires: bws, jq`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check dependencies
		if err := bws.CheckDependencies(); err != nil {
			return err
		}

		// Set project ID from env if not provided
		if projectID == "" {
			projectID = os.Getenv("BWS_PROJECT_ID")
		}
		if projectID == "" {
			return fmt.Errorf("no project specified. Set BWS_PROJECT_ID or pass --project-id")
		}

		// Initialize BWS client
		bwsClient = bws.NewClient(projectID)
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&projectID, "project-id", "", "Project UUID (overrides BWS_PROJECT_ID)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't make changes")
	rootCmd.PersistentFlags().BoolVar(&includeShared, "include-shared", false, "Include shared secrets")
}