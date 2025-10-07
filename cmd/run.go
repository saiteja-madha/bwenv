package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <app> <command> [args...]",
	Short: "Run command with app environment",
	Long: `Run a command with environment variables loaded from Bitwarden.

The specified app's environment variables are loaded and made available
to the command. Use --include-shared to also include shared secrets.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := args[0]
		command := args[1:]

		lines, err := bwsClient.GetEnvLines(app, includeShared)
		if err != nil {
			return err
		}

		// Parse environment variables
		env := os.Environ()
		for _, line := range lines {
			if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
				env = append(env, line)
			}
		}

		suffix := ""
		if includeShared {
			suffix = " + shared"
		}
		fmt.Fprintf(os.Stderr, "Running: %s  (env: %s%s)\n", strings.Join(command, " "), app, suffix)

		// Execute command
		binary, err := exec.LookPath(command[0])
		if err != nil {
			return fmt.Errorf("command not found: %s", command[0])
		}

		return syscall.Exec(binary, command, env)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
