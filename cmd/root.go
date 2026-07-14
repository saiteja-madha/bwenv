// Package cmd defines the bwenv command-line interface.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"bwenv/internal/bws"

	"github.com/spf13/cobra"
)

var (
	// Version is the release version populated with linker flags.
	Version = "dev"
	// Commit is the source revision populated with linker flags.
	Commit = "none"
	// Date is the build timestamp populated with linker flags.
	Date = "unknown"
)

type bwsClient interface {
	ListSecrets(context.Context, string) ([]bws.Secret, error)
	CreateSecret(context.Context, string, string, string, string) (bws.Secret, error)
	EditSecret(context.Context, string, bws.EditRequest) (bws.Secret, error)
	DeleteSecrets(context.Context, []string) (string, error)
}

type config struct {
	projectID   string
	output      string
	color       string
	accessToken string
	configFile  string
	profile     string
	serverURL   string
	verbose     bool
}

type runtimeDeps struct {
	client bwsClient
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
	getenv func(string) string
}

func defaultDeps() *runtimeDeps {
	return &runtimeDeps{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr, getenv: os.Getenv}
}

func newRootCommand(deps *runtimeDeps) *cobra.Command {
	if deps == nil {
		deps = defaultDeps()
	}
	cfg := &config{}
	root := &cobra.Command{
		Use:           "bwenv",
		Short:         "App-scoped environments backed by Bitwarden Secrets Manager",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       Version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if !contains([]string{"json", "yaml", "env", "table", "tsv", "none"}, cfg.output) {
				return fmt.Errorf("invalid output format %q", cfg.output)
			}
			if !contains([]string{"yes", "no", "auto"}, cfg.color) {
				return fmt.Errorf("invalid color mode %q", cfg.color)
			}
			if cmd.Annotations["needsProject"] == "true" {
				if cfg.projectID == "" {
					cfg.projectID = deps.getenv("BWS_PROJECT_ID")
				}
				if cfg.projectID == "" {
					return fmt.Errorf("no project specified: set BWS_PROJECT_ID or pass --project-id")
				}
			}
			if cmd.Annotations["needsBWS"] != "true" || deps.client != nil {
				return nil
			}
			client := bws.NewClient(bws.GlobalOptions{
				AccessToken: cfg.accessToken,
				ConfigFile:  cfg.configFile,
				Profile:     cfg.profile,
				ServerURL:   cfg.serverURL,
			}, cfg.verbose, deps.stderr)
			if err := bws.CheckDependency(client.Binary); err != nil {
				return err
			}
			deps.client = client
			return nil
		},
	}
	root.SetIn(deps.stdin)
	root.SetOut(deps.stdout)
	root.SetErr(deps.stderr)
	root.PersistentFlags().StringVar(&cfg.projectID, "project-id", "", "Bitwarden project UUID (or BWS_PROJECT_ID)")
	root.PersistentFlags().StringVarP(&cfg.output, "output", "o", "json", "Output format: json, yaml, env, table, tsv, none")
	root.PersistentFlags().StringVarP(&cfg.color, "color", "c", "auto", "Color mode: yes, no, auto")
	root.PersistentFlags().StringVarP(&cfg.accessToken, "access-token", "t", "", "BWS access token (or BWS_ACCESS_TOKEN)")
	root.PersistentFlags().StringVarP(&cfg.configFile, "config-file", "f", "", "BWS config file (or BWS_CONFIG_FILE)")
	root.PersistentFlags().StringVarP(&cfg.profile, "profile", "p", "", "BWS profile (or BWS_PROFILE)")
	root.PersistentFlags().StringVarP(&cfg.serverURL, "server-url", "u", "", "BWS server URL (or BWS_SERVER_URL)")
	root.PersistentFlags().BoolVar(&cfg.verbose, "verbose", false, "Log masked bws commands")
	root.InitDefaultVersionFlag()
	root.Flags().Lookup("version").Shorthand = "V"
	_ = root.RegisterFlagCompletionFunc("output", fixedCompletions("json", "yaml", "env", "table", "tsv", "none"))
	_ = root.RegisterFlagCompletionFunc("color", fixedCompletions("yes", "no", "auto"))

	root.AddCommand(newEnvironmentCommands(cfg, deps)...)
	root.AddCommand(newRunCommand(cfg, deps), newVersionCommand())
	return root
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func fixedCompletions(values ...string) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return values, cobra.ShellCompDirectiveNoFileComp
	}
}

// Execute runs the CLI and returns a process exit code.
func Execute() int {
	err := newRootCommand(nil).Execute()
	if err == nil {
		return 0
	}
	var exitErr *exitCodeError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	fmt.Fprintln(os.Stderr, "Error:", err)
	var commandErr *bws.CommandError
	if errors.As(err, &commandErr) && commandErr.Code > 0 {
		return commandErr.Code
	}
	return 1
}

func needsSecrets(command *cobra.Command) {
	command.Annotations = map[string]string{"needsBWS": "true", "needsProject": "true"}
}
