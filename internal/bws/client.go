// Package bws contains the subprocess boundary to the official bws CLI.
package bws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// GlobalOptions mirrors the global options supported by the official bws CLI.
type GlobalOptions struct {
	AccessToken string
	ConfigFile  string
	Profile     string
	ServerURL   string
}

// Secret is the JSON shape returned by bws secret commands.
type Secret struct {
	Object         string `json:"object,omitempty" yaml:"object,omitempty"`
	ID             string `json:"id" yaml:"id"`
	OrganizationID string `json:"organizationId,omitempty" yaml:"organizationId,omitempty"`
	ProjectID      string `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	Key            string `json:"key" yaml:"key"`
	Value          string `json:"value" yaml:"value"`
	Note           string `json:"note,omitempty" yaml:"note,omitempty"`
	CreationDate   string `json:"creationDate,omitempty" yaml:"creationDate,omitempty"`
	RevisionDate   string `json:"revisionDate,omitempty" yaml:"revisionDate,omitempty"`
}

// EditRequest contains only fields explicitly changed by the user.
type EditRequest struct {
	Key   *string
	Value *string
	Note  *string
}

// CommandError preserves bws stderr and its process exit code.
type CommandError struct {
	Args   []string
	Stderr string
	Code   int
	Err    error
}

func (e *CommandError) Error() string {
	message := strings.TrimSpace(e.Stderr)
	if message != "" {
		return message
	}
	return fmt.Sprintf("bws command failed: %v", e.Err)
}

func (e *CommandError) Unwrap() error { return e.Err }

// Client shells out to the official bws binary.
type Client struct {
	Binary  string
	Options GlobalOptions
	Verbose bool
	Stderr  io.Writer
	command func(context.Context, string, ...string) *exec.Cmd
}

// NewClient constructs a client using the official bws binary on PATH.
func NewClient(options GlobalOptions, verbose bool, stderr io.Writer) *Client {
	return &Client{Binary: "bws", Options: options, Verbose: verbose, Stderr: stderr}
}

// CheckDependency verifies that the configured bws binary is executable.
func CheckDependency(binary string) error {
	if binary == "" {
		binary = "bws"
	}
	if _, err := exec.LookPath(binary); err != nil {
		return fmt.Errorf("missing dependency: bws: %w", err)
	}
	return nil
}

func (c *Client) globalArgs() []string {
	args := make([]string, 0, 8)
	if c.Options.AccessToken != "" {
		args = append(args, "--access-token", c.Options.AccessToken)
	}
	if c.Options.ConfigFile != "" {
		args = append(args, "--config-file", c.Options.ConfigFile)
	}
	if c.Options.Profile != "" {
		args = append(args, "--profile", c.Options.Profile)
	}
	if c.Options.ServerURL != "" {
		args = append(args, "--server-url", c.Options.ServerURL)
	}
	return args
}

func (c *Client) runJSON(ctx context.Context, sensitiveValues []string, args ...string) ([]byte, error) {
	fullArgs := append(c.globalArgs(), args...)
	fullArgs = append(fullArgs, "--output", "json", "--color", "no")
	return c.run(ctx, sensitiveValues, fullArgs...)
}

func (c *Client) run(ctx context.Context, sensitiveValues []string, args ...string) ([]byte, error) {
	binary := c.Binary
	if binary == "" {
		binary = "bws"
	}
	if c.Verbose && c.Stderr != nil {
		if _, err := fmt.Fprintf(c.Stderr, "[bws] %s\n", strings.Join(maskArgs(args, sensitiveValues), " ")); err != nil {
			return nil, fmt.Errorf("write verbose bws command: %w", err)
		}
	}
	command := c.command
	if command == nil {
		command = exec.CommandContext
	}
	cmd := command(ctx, binary, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}
	code := 1
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	}
	return nil, &CommandError{Args: args, Stderr: stderr.String(), Code: code, Err: err}
}

func maskArgs(args, sensitiveValues []string) []string {
	masked := append([]string(nil), args...)
	for i := range masked {
		if i > 0 && masked[i-1] == "--access-token" {
			masked[i] = "***"
			continue
		}
		for _, value := range sensitiveValues {
			if value != "" && masked[i] == value {
				masked[i] = "***"
			}
		}
		if strings.HasPrefix(masked[i], "--value=") {
			masked[i] = "--value=***"
		}
		if strings.HasPrefix(masked[i], "--note=") {
			masked[i] = "--note=***"
		}
	}
	return masked
}

// ListSecrets returns every secret in one project.
func (c *Client) ListSecrets(ctx context.Context, projectID string) ([]Secret, error) {
	output, err := c.runJSON(ctx, nil, "secret", "list", projectID)
	if err != nil {
		return nil, err
	}
	var secrets []Secret
	if err := json.Unmarshal(output, &secrets); err != nil {
		return nil, fmt.Errorf("parse bws secret list response: %w", err)
	}
	return secrets, nil
}

// CreateSecret creates one secret and returns the official response.
func (c *Client) CreateSecret(ctx context.Context, key, value, projectID, note string) (Secret, error) {
	if err := validateValue(value); err != nil {
		return Secret{}, err
	}
	args := append(c.globalArgs(), "--output", "json", "--color", "no", "secret", "create")
	if note != "" {
		args = append(args, "--note="+note)
	}
	// The separator prevents values beginning with '-' from being parsed as flags.
	args = append(args, "--", key, value, projectID)
	output, err := c.run(ctx, []string{value, note}, args...)
	if err != nil {
		return Secret{}, err
	}
	return decodeSecret(output, "create")
}

// EditSecret applies the explicitly supplied fields to one secret.
func (c *Client) EditSecret(ctx context.Context, id string, request EditRequest) (Secret, error) {
	args := []string{"secret", "edit", id}
	sensitive := make([]string, 0, 2)
	if request.Key != nil {
		args = append(args, "--key", *request.Key)
	}
	if request.Value != nil {
		if err := validateValue(*request.Value); err != nil {
			return Secret{}, err
		}
		args = append(args, "--value="+*request.Value)
		sensitive = append(sensitive, *request.Value)
	}
	if request.Note != nil {
		args = append(args, "--note="+*request.Note)
		sensitive = append(sensitive, *request.Note)
	}
	output, err := c.runJSON(ctx, sensitive, args...)
	if err != nil {
		return Secret{}, err
	}
	return decodeSecret(output, "edit")
}

// DeleteSecrets deletes one or more secrets by UUID.
func (c *Client) DeleteSecrets(ctx context.Context, ids []string) (string, error) {
	args := append([]string{"secret", "delete"}, ids...)
	output, err := c.run(ctx, nil, append(c.globalArgs(), args...)...)
	return string(output), err
}

// Version returns the installed official bws version.
func (c *Client) Version(ctx context.Context) (string, error) {
	output, err := c.run(ctx, nil, "--version")
	return strings.TrimSpace(string(output)), err
}

func decodeSecret(output []byte, operation string) (Secret, error) {
	var secret Secret
	if err := json.Unmarshal(output, &secret); err != nil {
		return Secret{}, fmt.Errorf("parse bws secret %s response: %w", operation, err)
	}
	return secret, nil
}

func validateValue(value string) error {
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("secret value contains a null byte")
	}
	return nil
}
