package bws

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestValidateValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty", "", false},
		{"leading dash", "--password", false},
		{"unicode", "héllo 日本語", false},
		{"multiline", "line1\nline2", false},
		{"equals", "foo=bar=baz", false},
		{"null", "secret\x00hidden", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateSecretSubprocessContract(t *testing.T) {
	argsFile := t.TempDir() + "/args.json"
	envFile := t.TempDir() + "/token"
	t.Setenv("BWENV_HELPER_PROCESS", "1")
	t.Setenv("BWENV_ARGS_FILE", argsFile)
	t.Setenv("BWENV_ENV_FILE", envFile)
	t.Setenv("BWS_ACCESS_TOKEN", "inherited-token")
	client := NewClient(GlobalOptions{
		AccessToken: "access-token",
		ConfigFile:  "/tmp/config",
		Profile:     "lab",
		ServerURL:   "https://vault.example.test",
	}, false, nil)
	client.command = helperCommand
	secret, err := client.CreateSecret(context.Background(), "photos__TOKEN", "--leading-dash", "project-id", "private note")
	if err != nil {
		t.Fatal(err)
	}
	if secret.Key != "photos__TOKEN" || secret.Value != "--leading-dash" {
		t.Fatalf("unexpected decoded secret: %#v", secret)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"--config-file", "/tmp/config",
		"--profile", "lab",
		"--server-url", "https://vault.example.test",
		"--output", "json", "--color", "no",
		"secret", "create", "--note=private note", "--",
		"photos__TOKEN", "--leading-dash", "project-id",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bws args = %#v, want %#v", got, want)
	}
	token, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(token) != "access-token" {
		t.Fatalf("BWS_ACCESS_TOKEN = %q", token)
	}
}

func helperCommand(ctx context.Context, _ string, args ...string) *exec.Cmd {
	helperArgs := append([]string{"-test.run=TestBWSHelperProcess", "--"}, args...)
	return exec.CommandContext(ctx, os.Args[0], helperArgs...)
}

func TestBWSHelperProcess(_ *testing.T) {
	if os.Getenv("BWENV_HELPER_PROCESS") != "1" {
		return
	}
	separator := 0
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i + 1
			break
		}
	}
	args := os.Args[separator:]
	data, err := json.Marshal(args)
	if err != nil {
		os.Exit(2)
	}
	if err := os.WriteFile(os.Getenv("BWENV_ARGS_FILE"), data, 0o600); err != nil {
		os.Exit(2)
	}
	if envFile := os.Getenv("BWENV_ENV_FILE"); envFile != "" {
		if err := os.WriteFile(envFile, []byte(os.Getenv("BWS_ACCESS_TOKEN")), 0o600); err != nil {
			os.Exit(2)
		}
	}
	if warning := os.Getenv("BWENV_HELPER_STDERR"); warning != "" {
		_, _ = os.Stderr.WriteString(warning)
	}
	if code := os.Getenv("BWENV_HELPER_EXIT"); code != "" {
		exitCode, err := strconv.Atoi(code)
		if err != nil {
			os.Exit(2)
		}
		os.Exit(exitCode)
	}
	if response := os.Getenv("BWENV_HELPER_STDOUT"); response != "" {
		_, _ = os.Stdout.WriteString(response)
		os.Exit(0)
	}
	response := Secret{ID: "created", Key: "photos__TOKEN", Value: "--leading-dash"}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestSuccessfulStderrIsForwarded(t *testing.T) {
	argsFile := t.TempDir() + "/args.json"
	t.Setenv("BWENV_HELPER_PROCESS", "1")
	t.Setenv("BWENV_ARGS_FILE", argsFile)
	t.Setenv("BWENV_HELPER_STDERR", "bws warning\n")
	var stderr strings.Builder
	client := NewClient(GlobalOptions{}, false, &stderr)
	client.command = helperCommand

	if _, err := client.CreateSecret(context.Background(), "photos__TOKEN", "value", "project-id", ""); err != nil {
		t.Fatal(err)
	}
	if stderr.String() != "bws warning\n" {
		t.Fatalf("forwarded stderr = %q", stderr.String())
	}
}

func TestGlobalArgs(t *testing.T) {
	client := NewClient(GlobalOptions{
		AccessToken: "token",
		ConfigFile:  "/tmp/bws-config",
		Profile:     "homelab",
		ServerURL:   "https://vault.example.test",
	}, false, nil)
	want := []string{
		"--config-file", "/tmp/bws-config",
		"--profile", "homelab",
		"--server-url", "https://vault.example.test",
	}
	if got := client.globalArgs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("globalArgs() = %#v, want %#v", got, want)
	}
}

func TestSecretSubprocessContracts(t *testing.T) {
	tests := []struct {
		name     string
		response string
		call     func(*Client) error
		want     []string
	}{
		{
			name:     "list",
			response: "[]\n",
			call: func(client *Client) error {
				_, err := client.ListSecrets(context.Background(), "project-id")
				return err
			},
			want: []string{"secret", "list", "project-id", "--output", "json", "--color", "no"},
		},
		{
			name:     "edit",
			response: `{"id":"id","key":"photos__NEW","value":"value"}` + "\n",
			call: func(client *Client) error {
				key, value, note := "photos__NEW", "value", "note"
				_, err := client.EditSecret(context.Background(), "id", EditRequest{
					Key: &key, Value: &value, Note: &note,
				})
				return err
			},
			want: []string{
				"secret", "edit", "id", "--key", "photos__NEW",
				"--value=value", "--note=note", "--output", "json", "--color", "no",
			},
		},
		{
			name:     "delete",
			response: "2 secrets deleted successfully.\n",
			call: func(client *Client) error {
				_, err := client.DeleteSecrets(context.Background(), []string{"id-1", "id-2"})
				return err
			},
			want: []string{"secret", "delete", "id-1", "id-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argsFile := t.TempDir() + "/args.json"
			t.Setenv("BWENV_HELPER_PROCESS", "1")
			t.Setenv("BWENV_ARGS_FILE", argsFile)
			t.Setenv("BWENV_HELPER_STDOUT", tt.response)
			client := NewClient(GlobalOptions{}, false, nil)
			client.command = helperCommand
			if err := tt.call(client); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("bws args = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCommandErrorDoesNotExposeSensitiveArguments(t *testing.T) {
	argsFile := t.TempDir() + "/args.json"
	t.Setenv("BWENV_HELPER_PROCESS", "1")
	t.Setenv("BWENV_ARGS_FILE", argsFile)
	t.Setenv("BWENV_HELPER_EXIT", "7")
	client := NewClient(GlobalOptions{}, false, nil)
	client.command = helperCommand

	_, err := client.CreateSecret(context.Background(), "photos__TOKEN", "private-value", "project-id", "private-note")
	if err == nil {
		t.Fatal("expected subprocess error")
	}
	for _, sensitive := range []string{"private-value", "private-note"} {
		if strings.Contains(err.Error(), sensitive) {
			t.Fatalf("error exposes %q: %v", sensitive, err)
		}
	}
}

func TestMaskArgs(t *testing.T) {
	args := []string{
		"--access-token", "machine.secret",
		"secret", "edit", "id",
		"--value=starts-with-a-dash",
		"--note=private note",
	}
	masked := maskArgs(args, []string{"private note"})
	joined := strings.Join(masked, " ")
	for _, secret := range []string{"machine.secret", "starts-with-a-dash", "private note"} {
		if strings.Contains(joined, secret) {
			t.Fatalf("masked command leaks %q: %s", secret, joined)
		}
	}
	if strings.Count(joined, "***") != 3 {
		t.Fatalf("expected three masked values: %s", joined)
	}
}
