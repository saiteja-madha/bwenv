package cmd

import (
	"fmt"

	"bwenv/internal/environment"
	outputrenderer "bwenv/internal/output"

	"github.com/spf13/cobra"
)

func newListCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var includeShared bool
	var all bool
	command := &cobra.Command{
		Use:   "list [<app>|--all]",
		Short: "List one application's variables, or every app namespace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case all && len(args) > 0:
				return fmt.Errorf("an app name cannot be combined with --all")
			case len(args) == 0 && !all:
				return fmt.Errorf("provide an app name or use --all")
			}
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			if all {
				if _, err := fmt.Fprintln(deps.stderr, "Warning: list --all is deprecated; use bwenv apps (includes shared by default). Its meaning may change in the next major release."); err != nil {
					return err
				}
				return outputrenderer.RenderApps(deps.stdout, environment.CollectApps(secrets, includeShared), cfg.output, cfg.color)
			}
			entries, err := environment.Merge(secrets, args[0], includeShared)
			if err != nil {
				return err
			}
			return outputrenderer.RenderEntries(deps.stdout, entries, cfg.output, false, cfg.color)
		},
	}
	needsSecrets(command)
	command.Flags().BoolVar(&includeShared, "include-shared", false, "Merge shared variables; app values win")
	command.Flags().BoolVar(&all, "all", false, "Deprecated: use bwenv apps; lists namespaces with legacy shared filtering")
	return command
}
