package cmd

import (
	"bwenv/internal/environment"
	outputrenderer "bwenv/internal/output"

	"github.com/spf13/cobra"
)

func newGetCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var includeShared bool
	command := &cobra.Command{
		Use:   "get <app> <key>",
		Short: "Get one environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			entry, err := environment.Get(secrets, args[0], args[1], includeShared)
			if err != nil {
				return err
			}
			return outputrenderer.RenderEntries(deps.stdout, []environment.Entry{entry}, cfg.output, true, cfg.color)
		},
	}
	needsSecrets(command)
	command.Flags().BoolVar(&includeShared, "include-shared", false, "Fall back to the shared value")
	return command
}
