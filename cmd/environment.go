package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"bwenv/internal/bws"
	"bwenv/internal/environment"

	"github.com/spf13/cobra"
)

func newEnvironmentCommands(cfg *config, deps *runtimeDeps) []*cobra.Command {
	return []*cobra.Command{
		newCreateCommand(cfg, deps),
		newImportCommand(cfg, deps),
		newListCommand(cfg, deps),
		newGetCommand(cfg, deps),
		newEditCommand(cfg, deps),
		newDeleteCommand(cfg, deps),
		newExportCommand(cfg, deps),
	}
}

func loadEntries(cmd *cobra.Command, cfg *config, deps *runtimeDeps, app string, includeShared bool) ([]environment.Entry, error) {
	secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
	if err != nil {
		return nil, err
	}
	return environment.Merge(secrets, app, includeShared)
}

func normalizedEntry(secret bws.Secret, app, source string) environment.Entry {
	secret.Key = strings.TrimPrefix(secret.Key, app+"__")
	return environment.Entry{Secret: secret, Source: source}
}

func indexAppSecrets(secrets []bws.Secret, app string) (map[string]bws.Secret, error) {
	prefix := app + "__"
	index := make(map[string]bws.Secret)
	for _, secret := range secrets {
		if !strings.HasPrefix(secret.Key, prefix) {
			continue
		}
		if existing, ok := index[secret.Key]; ok {
			return nil, fmt.Errorf("duplicate Bitwarden secret key %q has IDs %s and %s", secret.Key, existing.ID, secret.ID)
		}
		index[secret.Key] = secret
	}
	return index, nil
}

func readInput(stdin io.Reader, path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dotenv file %s: %w", path, err)
	}
	return data, nil
}
