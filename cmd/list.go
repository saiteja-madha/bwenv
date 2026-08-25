package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"bwenv/internal/bws"
	"bwenv/internal/environment"
	outputrenderer "bwenv/internal/output"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type appRecord struct {
	App  string `json:"app" yaml:"app"`
	Keys int    `json:"keys" yaml:"keys"`
}

func newListCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var includeShared bool
	var all bool
	command := &cobra.Command{
		Use:   "list [<app>|--all]",
		Short: "List one application's variables, or every app namespace",
		Args:  cobra.ArbitraryArgs,
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
				return renderApps(deps.stdout, collectApps(secrets, includeShared), cfg.output, cfg.color)
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
	command.Flags().BoolVar(&all, "all", false, "List every app namespace found in the project")
	return command
}

func collectApps(secrets []bws.Secret, includeShared bool) []appRecord {
	counts := make(map[string]int)
	for _, secret := range secrets {
		app, key, ok := strings.Cut(secret.Key, environment.Separator)
		if !ok || app == "" || key == "" {
			continue
		}
		if app == environment.SharedNamespace && !includeShared {
			continue
		}
		if err := environment.ValidateApp(app); err != nil {
			continue
		}
		if err := environment.ValidateKey(key); err != nil {
			continue
		}
		counts[app]++
	}
	apps := make([]string, 0, len(counts))
	for app := range counts {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	records := make([]appRecord, 0, len(apps))
	for _, app := range apps {
		records = append(records, appRecord{App: app, Keys: counts[app]})
	}
	return records
}

func renderApps(w io.Writer, records []appRecord, format, color string) error {
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.MarshalIndent(records, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	case "yaml":
		data, err = yaml.Marshal(records)
	case "env":
		names := make([]string, len(records))
		for i, record := range records {
			names[i] = record.App
		}
		data = []byte("APPS=" + strconv.Quote(strings.Join(names, ",")) + "\n")
	case "table", "tsv":
		data, err = renderAppRows(records, format == "table")
	case "none":
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
	if err != nil {
		return err
	}
	return outputrenderer.Write(w, data, color)
}

func renderAppRows(records []appRecord, table bool) ([]byte, error) {
	var buffer strings.Builder
	tw := tabwriter.NewWriter(&buffer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "APP\tKEYS"); err != nil {
		return nil, err
	}
	for _, record := range records {
		if _, err := fmt.Fprintf(tw, "%s\t%d\n", record.App, record.Keys); err != nil {
			return nil, err
		}
	}
	if err := tw.Flush(); err != nil {
		return nil, err
	}
	output := buffer.String()
	if !table {
		lines := strings.SplitN(output, "\n", 2)
		output = lines[1]
	}
	return []byte(output), nil
}
