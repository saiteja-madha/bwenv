package cmd

import (
	"fmt"

	"bwenv/internal/environment"
	outputrenderer "bwenv/internal/output"

	"github.com/spf13/cobra"
)

func newCreateCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var note string
	var dryRun bool
	command := &cobra.Command{
		Use:   "create <app> <key> <value>",
		Short: "Create an environment variable",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			fullKey, err := environment.FullKey(args[0], args[1])
			if err != nil {
				return err
			}
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			for _, secret := range secrets {
				if secret.Key == fullKey {
					return fmt.Errorf("environment key %s/%s already exists as secret %s; use edit", args[0], args[1], secret.ID)
				}
			}
			if dryRun {
				_, err = fmt.Fprintf(deps.stdout, "create %s\n", fullKey)
				return err
			}
			secret, err := deps.client.CreateSecret(cmd.Context(), fullKey, args[2], cfg.projectID, note)
			if err != nil {
				return err
			}
			entry := normalizedEntry(secret, args[0], "app")
			return outputrenderer.RenderEntries(deps.stdout, []environment.Entry{entry}, cfg.output, true, cfg.color)
		},
	}
	needsSecrets(command)
	command.Flags().StringVar(&note, "note", "", "Optional Bitwarden secret note")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the operation without writing")
	return command
}
