package cmd

import (
	outputrenderer "bwenv/internal/output"

	"github.com/spf13/cobra"
)

func newExportCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var includeShared bool
	command := &cobra.Command{
		Use:   "export <app>",
		Short: "Export an application's effective environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := loadEntries(cmd, cfg, deps, args[0], includeShared)
			if err != nil {
				return err
			}
			format := cfg.output
			if !cmd.Root().PersistentFlags().Changed("output") {
				format = "env"
			}
			return outputrenderer.RenderEntries(deps.stdout, entries, format, false, cfg.color)
		},
	}
	needsSecrets(command)
	command.Flags().BoolVar(&includeShared, "include-shared", false, "Merge shared variables; app values win")
	return command
}
