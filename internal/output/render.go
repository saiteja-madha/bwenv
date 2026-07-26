// Package output renders normalized bwenv records for users and scripts.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"bwenv/internal/environment"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// RenderEntries writes normalized environment entries in a bws-compatible format.
func RenderEntries(w io.Writer, entries []environment.Entry, format string, single bool, color string) error {
	var buffer bytes.Buffer
	if err := renderPlain(&buffer, entries, format, single); err != nil {
		return err
	}
	return Write(w, buffer.Bytes(), color)
}

func renderPlain(w io.Writer, entries []environment.Entry, format string, single bool) error {
	var value any = entries
	if single {
		if len(entries) != 1 {
			return fmt.Errorf("expected one entry, got %d", len(entries))
		}
		value = entries[0]
	}
	switch strings.ToLower(format) {
	case "json":
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	case "yaml":
		data, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	case "env":
		values := make(map[string]string, len(entries))
		for _, entry := range entries {
			values[entry.Key] = entry.Value
		}
		data, err := godotenv.Marshal(values)
		if err != nil {
			return fmt.Errorf("encode dotenv output: %w", err)
		}
		if data == "" {
			return nil
		}
		_, err = fmt.Fprintln(w, data)
		return err
	case "table":
		tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
		if _, err := fmt.Fprintln(tw, "ID\tKEY\tVALUE\tSOURCE\tCREATION DATE"); err != nil {
			return err
		}
		for _, entry := range entries {
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", entry.ID, entry.Key, entry.Value, entry.Source, entry.CreationDate); err != nil {
				return err
			}
		}
		return tw.Flush()
	case "tsv":
		if _, err := fmt.Fprintln(w, "ID\tKey\tValue\tSource\tCreation Date"); err != nil {
			return err
		}
		for _, entry := range entries {
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", entry.ID, entry.Key, entry.Value, entry.Source, entry.CreationDate); err != nil {
				return err
			}
		}
		return nil
	case "none":
		return nil
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

// Write applies the official yes/no/auto color policy to rendered output.
func Write(w io.Writer, data []byte, color string) error {
	if len(data) == 0 {
		return nil
	}
	if colorEnabled(w, color) {
		if _, err := io.WriteString(w, "\x1b[36m"); err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
		_, err := io.WriteString(w, "\x1b[0m")
		return err
	}
	_, err := w.Write(data)
	return err
}

func colorEnabled(w io.Writer, mode string) bool {
	switch mode {
	case "yes":
		return true
	case "no":
		return false
	}
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" {
		return false
	}
	if force := os.Getenv("CLICOLOR_FORCE"); force != "" && force != "0" {
		return true
	}
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
