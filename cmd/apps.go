package cmd

import (
	"bwenv/internal/environment"
	outputrenderer "bwenv/internal/output"
	"github.com/spf13/cobra"
)

func newAppsCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	command := &cobra.Command{
		Use:   "apps",
		Short: "List application namespaces and key counts, including shared",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			return outputrenderer.RenderApps(deps.stdout, environment.CollectApps(secrets, true), cfg.output, cfg.color)
		},
	}
	needsSecrets(command)
	return command
}
