package output

import (
	"bwenv/internal/environment"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

// RenderApps writes namespace summaries.
func RenderApps(w io.Writer, records []environment.AppRecord, format, color string) error {
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
	return WriteFormatted(w, data, format, color)
}

func renderAppRows(records []environment.AppRecord, table bool) ([]byte, error) {
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
