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

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type importSummary struct {
	App     string   `json:"app" yaml:"app"`
	Created []string `json:"created" yaml:"created"`
	Updated []string `json:"updated" yaml:"updated"`
	DryRun  bool     `json:"dryRun" yaml:"dryRun"`
}

func newImportCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var dryRun bool
	command := &cobra.Command{
		Use:   "import <app> <file|->",
		Short: "Import and upsert variables from a dotenv file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := args[0]
			if err := environment.ValidateApp(app); err != nil {
				return err
			}
			data, err := readInput(deps.stdin, args[1])
			if err != nil {
				return err
			}
			values, err := godotenv.Unmarshal(string(data))
			if err != nil {
				return fmt.Errorf("parse dotenv input: %w", err)
			}
			keys := make([]string, 0, len(values))
			for key := range values {
				if err := environment.ValidateKey(key); err != nil {
					return err
				}
				keys = append(keys, key)
			}
			sort.Strings(keys)
			secrets, err := deps.client.ListSecrets(cmd.Context(), cfg.projectID)
			if err != nil {
				return err
			}
			index, err := indexAppSecrets(secrets, app)
			if err != nil {
				return err
			}
			summary := importSummary{App: app, Created: []string{}, Updated: []string{}, DryRun: dryRun}
			for _, key := range keys {
				fullKey, _ := environment.FullKey(app, key)
				value := values[key]
				if existing, ok := index[fullKey]; ok {
					summary.Updated = append(summary.Updated, key)
					if !dryRun {
						request := bws.EditRequest{Value: &value}
						if _, err := deps.client.EditSecret(cmd.Context(), existing.ID, request); err != nil {
							return importFailure(key, summary, err)
						}
					}
				} else {
					summary.Created = append(summary.Created, key)
					if !dryRun {
						if _, err := deps.client.CreateSecret(cmd.Context(), fullKey, value, cfg.projectID, ""); err != nil {
							return importFailure(key, summary, err)
						}
					}
				}
			}
			return renderImportSummary(deps.stdout, summary, cfg.output, cfg.color)
		},
	}
	needsSecrets(command)
	command.Flags().BoolVar(&dryRun, "dry-run", false, "Show planned creates and updates without writing")
	return command
}

func importFailure(key string, summary importSummary, err error) error {
	completed := len(summary.Created) + len(summary.Updated) - 1
	return fmt.Errorf("import %s after %d completed operations: %w", key, completed, err)
}

func renderImportSummary(w io.Writer, summary importSummary, format, color string) error {
	var data []byte
	var err error
	switch format {
	case "json":
		data, err = json.MarshalIndent(summary, "", "  ")
		if err == nil {
			data = append(data, '\n')
		}
	case "yaml":
		data, err = yaml.Marshal(summary)
	case "env":
		data = []byte(fmt.Sprintf(
			"APP=%s\nCREATED=%s\nUPDATED=%s\nDRY_RUN=%s\n",
			strconv.Quote(summary.App),
			strconv.Quote(strings.Join(summary.Created, ",")),
			strconv.Quote(strings.Join(summary.Updated, ",")),
			strconv.Quote(strconv.FormatBool(summary.DryRun)),
		))
	case "table", "tsv":
		data, err = renderImportRows(summary, format == "table")
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

func renderImportRows(summary importSummary, table bool) ([]byte, error) {
	var buffer strings.Builder
	if !table {
		buffer.WriteString("Action\tKey\n")
		for _, key := range summary.Created {
			buffer.WriteString("created\t" + key + "\n")
		}
		for _, key := range summary.Updated {
			buffer.WriteString("updated\t")
			buffer.WriteString(key)
			buffer.WriteString("\n")
		}
		return []byte(buffer.String()), nil
	}
	tw := tabwriter.NewWriter(&buffer, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ACTION\tKEY"); err != nil {
		return nil, err
	}
	for _, key := range summary.Created {
		if _, err := fmt.Fprintf(tw, "created\t%s\n", key); err != nil {
			return nil, err
		}
	}
	for _, key := range summary.Updated {
		if _, err := fmt.Fprintf(tw, "updated\t%s\n", key); err != nil {
			return nil, err
		}
	}
	if err := tw.Flush(); err != nil {
		return nil, err
	}
	return []byte(buffer.String()), nil
}
