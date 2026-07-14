package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"bwenv/internal/environment"

	"github.com/spf13/cobra"
)

type exitCodeError struct{ Code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("command exited with status %d", e.Code) }

func newRunCommand(cfg *config, deps *runtimeDeps) *cobra.Command {
	var includeShared, noInherit, uuidsAsKeynames bool
	var shell string
	command := &cobra.Command{
		Use:   "run <app> [--] <command> [args...]",
		Short: "Run a command with an app-scoped environment",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("uuids-as-keynames") {
				value := deps.getenv("BWS_UUIDS_AS_KEYNAMES")
				if value != "" {
					parsed, err := strconv.ParseBool(value)
					if err != nil {
						return fmt.Errorf("invalid BWS_UUIDS_AS_KEYNAMES value %q: %w", value, err)
					}
					uuidsAsKeynames = parsed
				}
			}
			entries, err := loadEntries(cmd, cfg, deps, args[0], includeShared)
			if err != nil {
				return err
			}
			userCommand := strings.Join(args[1:], " ")
			if userCommand == "" {
				data, err := io.ReadAll(deps.stdin)
				if err != nil {
					return fmt.Errorf("read command from stdin: %w", err)
				}
				userCommand = string(data)
			}
			if strings.TrimSpace(userCommand) == "" {
				return fmt.Errorf("no command provided")
			}
			if shell == "" {
				if runtime.GOOS == "windows" {
					shell = "powershell"
				} else {
					shell = "sh"
				}
			}
			if _, err := exec.LookPath(shell); err != nil {
				return fmt.Errorf("shell %q not found: %w", shell, err)
			}
			env := buildEnvironment(entries, noInherit, uuidsAsKeynames)
			child := exec.CommandContext(cmd.Context(), shell, "-c", userCommand)
			child.Stdin = deps.stdin
			child.Stdout = deps.stdout
			child.Stderr = deps.stderr
			child.Env = env
			err = child.Run()
			if err == nil {
				return nil
			}
			if exitErr, ok := err.(*exec.ExitError); ok {
				code := exitErr.ExitCode()
				if code < 0 {
					code = 1
				}
				return &exitCodeError{Code: code}
			}
			return fmt.Errorf("execute command: %w", err)
		},
	}
	needsSecrets(command)
	command.Flags().BoolVar(&includeShared, "include-shared", false, "Merge shared variables; app values win")
	command.Flags().StringVar(&shell, "shell", "", "Shell used to execute the command")
	command.Flags().BoolVar(&noInherit, "no-inherit-env", false, "Do not inherit the current environment except essentials")
	command.Flags().BoolVar(&uuidsAsKeynames, "uuids-as-keynames", false, "Use POSIX-normalized secret UUIDs as variable names")
	return command
}

func buildEnvironment(entries []environment.Entry, noInherit, uuidsAsKeynames bool) []string {
	env := make(map[string]string)
	if !noInherit {
		for _, pair := range os.Environ() {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 && parts[0] != "BWS_ACCESS_TOKEN" {
				env[parts[0]] = parts[1]
			}
		}
	} else {
		if path, ok := os.LookupEnv("PATH"); ok {
			env["PATH"] = path
		} else if runtime.GOOS == "windows" {
			env["PATH"] = `C:\Windows;C:\Windows\System32`
		} else {
			env["PATH"] = "/bin:/usr/bin"
		}
		if runtime.GOOS == "windows" {
			for _, key := range []string{"SystemRoot", "ComSpec", "windir"} {
				if value, ok := os.LookupEnv(key); ok {
					env[key] = value
				}
			}
		}
	}
	for _, entry := range entries {
		key := entry.Key
		if uuidsAsKeynames {
			key = "_" + strings.ReplaceAll(entry.ID, "-", "_")
		}
		env[key] = entry.Value
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+env[key])
	}
	return result
}
