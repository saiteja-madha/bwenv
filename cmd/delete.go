package cmd

import (
	"fmt"
	"io"
	"strings"

	"bwenv/internal/environment"

	"github.com/spf13/cobra"
)

func newDeleteCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var dryRun bool
	command := &cobra.Command{
		Use:   "delete <app> <key>...",
		Short: "Delete one or more environment variables",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			ids := make([]string, 0, len(args)-1)
			seen := make(map[string]struct{}, len(args)-1)
			for _, key := range args[1:] {
				if _, ok := seen[key]; ok {
					return fmt.Errorf("environment key %s/%s was requested more than once", args[0], key)
				}
				seen[key] = struct{}{}
				secret, err := environment.Resolve(secrets, args[0], key)
				if err != nil {
					return err
				}
				ids = append(ids, secret.ID)
			}
			if dryRun {
				_, err = fmt.Fprintf(deps.stdout, "delete %s/%s\n", args[0], strings.Join(args[1:], ","))
				return err
			}
			message, err := deps.client.DeleteSecrets(cmd.Context(), ids)
			if err != nil {
				return err
			}
			if cfg.output != "none" {
				_, err = io.WriteString(deps.stdout, message)
			}
			return err
		},
	}
	needsSecrets(command)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show the operation without writing")
	return command
}
