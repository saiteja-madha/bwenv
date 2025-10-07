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
	if os.Getenv("BWS_ACCESS_TOKEN") == "" {
		return fmt.Errorf("BWS_ACCESS_TOKEN is not set")
	}
	return nil
}

// ListSecrets fetches all secrets from the project
func (c *Client) ListSecrets() ([]Secret, error) {
	cmd := exec.Command("bws", "secret", "list", c.ProjectID, "-o", "json")
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

// CreateSecret creates a new secret
func (c *Client) CreateSecret(name, value string, dryRun bool) error {
	if dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] create %s\n", name)
		return nil
	}

	cmd := exec.Command("bws", "secret", "create", name, value, c.ProjectID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	fmt.Fprintf(os.Stderr, "created: %s\n", name)
	return nil
}

// UpdateSecret updates an existing secret
func (c *Client) UpdateSecret(id, name, value string, dryRun bool) error {
	if dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] update %s\n", name)
		return nil
	}

	cmd := exec.Command("bws", "secret", "edit", "--key", name, "--value", value, "--project-id", c.ProjectID, id)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
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

	return lines, nil
}

// GetAppKeys returns all keys for an app (without prefix)
func (c *Client) GetAppKeys(app string) ([]string, error) {
	secrets, err := c.ListSecrets()
	if err != nil {
		return nil, err
	}

	var keys []string
	prefix := app + "__"

	for _, secret := range secrets {
		if secret.Key != "" && strings.HasPrefix(secret.Key, prefix) {
			key := strings.TrimPrefix(secret.Key, prefix)
			keys = append(keys, key)
		}
	}

	return keys, nil
}

func checkCommand(name string) error {
	_, err := exec.LookPath(name)
	return err
}
