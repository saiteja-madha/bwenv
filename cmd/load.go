package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load <app> <file>",
	Short: "Load secrets from .env file",
	Long: `Load environment variables from a .env file into Bitwarden secrets.

The file should contain lines in the format:
  KEY=VALUE

Comments (lines starting with #) and empty lines are ignored.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app, file := args[0], args[1]

		f, err := os.Open(file)
		if err != nil {
			return fmt.Errorf("file not found: %s", file)
		}
		defer f.Close()

		envRegex := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
		scanner := bufio.NewScanner(f)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			if envRegex.MatchString(line) {
				parts := strings.SplitN(line, "=", 2)
				key, value := parts[0], parts[1]
				fullName := fmt.Sprintf("%s__%s", app, key)
				if err := bwsClient.UpsertSecret(fullName, value, dryRun); err != nil {
					return err
				}
			}
		}

		return scanner.Err()
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
}