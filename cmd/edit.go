package cmd

import (
	"fmt"

	"bwenv/internal/bws"
	"bwenv/internal/environment"
	outputrenderer "bwenv/internal/output"

	"github.com/spf13/cobra"
)

func newEditCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var newKey, value, note string
	var dryRun bool
	command := &cobra.Command{
		Use:   "edit <app> <key>",
		Short: "Edit an environment variable",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("key") && !cmd.Flags().Changed("value") && !cmd.Flags().Changed("note") {
				return fmt.Errorf("provide at least one of --key, --value or --note")
			}
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			secret, err := environment.Resolve(secrets, args[0], args[1])
			if err != nil {
				return err
			}
			request := bws.EditRequest{}
			if cmd.Flags().Changed("key") {
				fullKey, err := environment.FullKey(args[0], newKey)
				if err != nil {
					return err
				}
				for _, candidate := range secrets {
					if candidate.Key == fullKey && candidate.ID != secret.ID {
						return fmt.Errorf("environment key %s/%s already exists", args[0], newKey)
					}
				}
				request.Key = &fullKey
			}
			if cmd.Flags().Changed("value") {
				request.Value = &value
			}
			if cmd.Flags().Changed("note") {
				request.Note = &note
			}
			if dryRun {
				_, err = fmt.Fprintf(deps.stdout, "edit %s\n", secret.Key)
				return err
			}
			updated, err := deps.client.EditSecret(cmd.Context(), secret.ID, request)
			if err != nil {
				return err
			}
			entry := normalizedEntry(updated, args[0], "app")
			return outputrenderer.RenderEntries(deps.stdout, []environment.Entry{entry}, cfg.output, true, cfg.color)
		},
	}
	needsSecrets(command)
	command.Flags().StringVar(&newKey, "key", "", "Rename the environment key")
	command.Flags().StringVar(&value, "value", "", "Set the value; empty is valid")
	command.Flags().StringVar(&note, "note", "", "Set the note; empty clears it")
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the operation without writing")
	return command
}
