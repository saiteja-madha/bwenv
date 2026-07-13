package bws

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Secret represents a Bitwarden secret
type Secret struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Client wraps BWS operations
type Client struct {
	ProjectID string
	Verbose   bool
}

// NewClient creates a new BWS client
func NewClient(projectID string) *Client {
	return &Client{ProjectID: projectID}
}

// CheckDependencies verifies required tools are available
func CheckDependencies() error {
	if err := checkCommand("bws"); err != nil {
		return fmt.Errorf("missing dependency: bws")
	}
	if err := checkCommand("jq"); err != nil {
		return fmt.Errorf("missing dependency: jq")
	}
	token := os.Getenv("BWS_ACCESS_TOKEN")
	if token == "" {
		return fmt.Errorf("BWS_ACCESS_TOKEN is not set")
	}
	if parts := strings.SplitN(token, ".", 2); len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("BWS_ACCESS_TOKEN is invalid: expected format <client_id>.<client_secret>")
	}
	return nil
}

func (c *Client) logCmd(cmd *exec.Cmd, maskArgs ...int) {
	if !c.Verbose {
		return
	}
	args := make([]string, len(cmd.Args))
	copy(args, cmd.Args)
	for _, i := range maskArgs {
		if i < len(args) {
			args[i] = "***"
		}
	}
	fmt.Fprintf(os.Stderr, "[bws] %s\n", strings.Join(args, " "))
}

// ListSecrets fetches all secrets from the project
func (c *Client) ListSecrets() ([]Secret, error) {
	cmd := exec.Command("bws", "secret", "list", c.ProjectID, "-o", "json")
	c.logCmd(cmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	var secrets []Secret
	if err := json.Unmarshal(output, &secrets); err != nil {
		return nil, fmt.Errorf("failed to parse secrets: %w", err)
	}

	return secrets, nil
}

// GetSecretID finds the ID of a secret by name
func (c *Client) GetSecretID(name string) (string, error) {
	secrets, err := c.ListSecrets()
	if err != nil {
		return "", err
	}

	for _, secret := range secrets {
		if secret.Key == name {
			return secret.ID, nil
		}
	}
	return "", nil
}

func validateValue(value string) error {
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("value contains null byte")
	}
	return nil
}

// CreateSecret creates a new secret
func (c *Client) CreateSecret(name, value string, dryRun bool) error {
	if err := validateValue(value); err != nil {
		return fmt.Errorf("invalid value for %s: %w", name, err)
	}
	if dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] create %s\n", name)
		return nil
	}

	cmd := exec.Command("bws", "secret", "create", name, value, c.ProjectID)
	c.logCmd(cmd, 4)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create secret %s: %w", name, err)
	}
	fmt.Fprintf(os.Stderr, "created: %s\n", name)
	return nil
}

// UpdateSecret updates an existing secret
func (c *Client) UpdateSecret(id, name, value string, dryRun bool) error {
	if err := validateValue(value); err != nil {
		return fmt.Errorf("invalid value for %s: %w", name, err)
	}
	if dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] update %s\n", name)
		return nil
	}

	cmd := exec.Command("bws", "secret", "edit", "--key", name, fmt.Sprintf("--value=%s", value), fmt.Sprintf("--project-id=%s", c.ProjectID), id)
	c.logCmd(cmd, 5)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update secret %s: %w", name, err)
	}
	fmt.Fprintf(os.Stderr, "updated: %s\n", name)
	return nil
}

// UpsertSecret creates or updates a secret
func (c *Client) UpsertSecret(name, value string, dryRun bool) error {
	id, err := c.GetSecretID(name)
	if err != nil {
		return err
	}

	if id != "" {
		return c.UpdateSecret(id, name, value, dryRun)
	}
	return c.CreateSecret(name, value, dryRun)
}

// GetEnvLines returns environment variable lines for an app
func (c *Client) GetEnvLines(app string, includeShared bool) ([]string, error) {
	secrets, err := c.ListSecrets()
	if err != nil {
		return nil, err
	}

	return FilterEnvLines(secrets, app, includeShared), nil
}

// FilterEnvLines filters secrets into KEY=VALUE lines for the given app
func FilterEnvLines(secrets []Secret, app string, includeShared bool) []string {
	var lines []string
	prefix := app + "__"

	// App-specific secrets
	for _, secret := range secrets {
		if secret.Key != "" && strings.HasPrefix(secret.Key, prefix) {
			key := strings.TrimPrefix(secret.Key, prefix)
			lines = append(lines, fmt.Sprintf("%s=%s", key, secret.Value))
		}
	}

	// Shared secrets if requested
	if includeShared {
		sharedPrefix := "shared__"
		for _, secret := range secrets {
			if secret.Key != "" && strings.HasPrefix(secret.Key, sharedPrefix) {
				key := strings.TrimPrefix(secret.Key, sharedPrefix)
				lines = append(lines, fmt.Sprintf("%s=%s", key, secret.Value))
			}
		}
	}

	return lines
}

// GetAppKeys returns all keys for an app (without prefix)
func (c *Client) GetAppKeys(app string) ([]string, error) {
	secrets, err := c.ListSecrets()
	if err != nil {
		return nil, err
	}

	return FilterAppKeys(secrets, app), nil
}

// FilterAppKeys filters secrets into key names for the given app (without prefix)
func FilterAppKeys(secrets []Secret, app string) []string {
	var keys []string
	prefix := app + "__"

	for _, secret := range secrets {
		if secret.Key != "" && strings.HasPrefix(secret.Key, prefix) {
			key := strings.TrimPrefix(secret.Key, prefix)
			keys = append(keys, key)
		}
	}

	return keys
}

func checkCommand(name string) error {
	_, err := exec.LookPath(name)
	return err
}
