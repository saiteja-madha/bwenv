package cmd

import (
	"bytes"
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"

	"bwenv/internal/bws"
	"bwenv/internal/environment"
)

type fakeClient struct {
	secrets      []bws.Secret
	listCalls    int
	created      []bws.Secret
	editedIDs    []string
	editRequests []bws.EditRequest
	deletedIDs   []string
}

func (f *fakeClient) ListSecrets(context.Context, string) ([]bws.Secret, error) {
	f.listCalls++
	return append([]bws.Secret(nil), f.secrets...), nil
}

func (f *fakeClient) CreateSecret(_ context.Context, key, value, projectID, note string) (bws.Secret, error) {
	secret := bws.Secret{ID: "created-id", Key: key, Value: value, ProjectID: projectID, Note: note}
	f.created = append(f.created, secret)
	return secret, nil
}

func (f *fakeClient) EditSecret(_ context.Context, id string, request bws.EditRequest) (bws.Secret, error) {
	f.editedIDs = append(f.editedIDs, id)
	f.editRequests = append(f.editRequests, request)
	for _, secret := range f.secrets {
		if secret.ID == id {
			if request.Key != nil {
				secret.Key = *request.Key
			}
			if request.Value != nil {
				secret.Value = *request.Value
			}
			if request.Note != nil {
				secret.Note = *request.Note
			}
			return secret, nil
		}
	}
	return bws.Secret{}, nil
}

func (f *fakeClient) DeleteSecrets(_ context.Context, ids []string) (string, error) {
	f.deletedIDs = append(f.deletedIDs, ids...)
	return "deleted successfully\n", nil
}

func executeForTest(t *testing.T, client *fakeClient, stdin string, args ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	deps := &runtimeDeps{
		client: client,
		stdin:  strings.NewReader(stdin),
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(key string) string {
			if key == "BWS_PROJECT_ID" {
				return "project-id"
			}
			return ""
		},
	}
	root := newRootCommand(deps)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func TestRootUsesFlatCommands(t *testing.T) {
	stdout, _, err := executeForTest(t, &fakeClient{}, "", "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"create", "import", "list", "get", "edit", "delete", "export", "run", "completion", "version"} {
		if !strings.Contains(stdout, command) {
			t.Errorf("root help missing %q:\n%s", command, stdout)
		}
	}
	if strings.Contains(stdout, "env create") {
		t.Fatalf("root help still contains nested env command:\n%s", stdout)
	}
}

func TestVersionNeedsNoProjectOrBWS(t *testing.T) {
	stdout, _, err := executeForTest(t, nil, "", "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "bwenv "+Version) {
		t.Fatalf("unexpected version output: %s", stdout)
	}
}

func TestCompletionNeedsNoProjectOrBWS(t *testing.T) {
	stdout, _, err := executeForTest(t, nil, "", "completion", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "bash completion") {
		t.Fatalf("unexpected completion output: %s", stdout)
	}
}

func TestCreateIsNotUpsert(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{{ID: "existing", Key: "photos__TOKEN", Value: "old"}}}
	_, _, err := executeForTest(t, client, "", "create", "photos", "TOKEN", "new")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing-key error, got %v", err)
	}
	if len(client.created) != 0 || len(client.editedIDs) != 0 {
		t.Fatal("create mutated an existing key")
	}
}

func TestImportListsOnceAndUpserts(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{{ID: "existing", Key: "photos__TOKEN", Value: "old"}}}
	stdout, _, err := executeForTest(t, client, "TOKEN=new\nPORT=8080\n", "import", "photos", "-")
	if err != nil {
		t.Fatal(err)
	}
	if client.listCalls != 1 || len(client.editedIDs) != 1 || len(client.created) != 1 {
		t.Fatalf("unexpected import calls: list=%d edit=%d create=%d", client.listCalls, len(client.editedIDs), len(client.created))
	}
	if !strings.Contains(stdout, `"created": [`) || !strings.Contains(stdout, `"updated": [`) {
		t.Fatalf("unexpected summary: %s", stdout)
	}
}

func TestImportDuplicateUsesFinalDefinition(t *testing.T) {
	client := &fakeClient{}
	_, _, err := executeForTest(t, client, "TOKEN=first\nTOKEN=last\n", "import", "photos", "-")
	if err != nil {
		t.Fatal(err)
	}
	if len(client.created) != 1 || client.created[0].Value != "last" {
		t.Fatalf("duplicate dotenv handling = %#v", client.created)
	}
}

func TestImportRejectsAllNullValuesBeforeMutation(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{{ID: "existing", Key: "photos__FIRST", Value: "old"}}}
	_, _, err := executeForTest(t, client, "FIRST=new\nSECOND=contains\x00null\n", "import", "photos", "-")
	if err == nil || !strings.Contains(err.Error(), "null byte") {
		t.Fatalf("expected null-byte error, got %v", err)
	}
	if client.listCalls != 0 || len(client.created) != 0 || len(client.editedIDs) != 0 {
		t.Fatalf("import performed work before complete validation: list=%d create=%d edit=%d", client.listCalls, len(client.created), len(client.editedIDs))
	}
}

func TestExportMergesSharedWithAppPrecedence(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{
		{ID: "1", Key: "shared__TOKEN", Value: "shared"},
		{ID: "2", Key: "shared__LOG", Value: "info"},
		{ID: "3", Key: "photos__TOKEN", Value: "app"},
	}}
	stdout, _, err := executeForTest(t, client, "", "export", "photos", "--include-shared")
	if err != nil {
		t.Fatal(err)
	}
	if stdout != "LOG=\"info\"\nTOKEN=\"app\"\n" {
		t.Fatalf("unexpected export: %q", stdout)
	}
}

func TestEditAllowsEmptyValue(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{{ID: "1", Key: "photos__TOKEN", Value: "old"}}}
	_, _, err := executeForTest(t, client, "", "edit", "photos", "TOKEN", "--value=")
	if err != nil {
		t.Fatal(err)
	}
	if len(client.editRequests) != 1 || client.editRequests[0].Value == nil || *client.editRequests[0].Value != "" {
		t.Fatalf("empty value was not preserved: %#v", client.editRequests)
	}
}

func TestDeleteResolvesAllKeysBeforeMutation(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{{ID: "1", Key: "photos__ONE"}}}
	_, _, err := executeForTest(t, client, "", "delete", "photos", "ONE", "MISSING")
	if err == nil {
		t.Fatal("expected missing-key error")
	}
	if len(client.deletedIDs) != 0 {
		t.Fatal("delete mutated before resolving every key")
	}
}

func TestRunStripsTokenAndAppliesSharedPrecedence(t *testing.T) {
	t.Setenv("BWS_ACCESS_TOKEN", "must-not-leak")
	client := &fakeClient{secrets: []bws.Secret{
		{ID: "1", Key: "shared__TOKEN", Value: "shared"},
		{ID: "2", Key: "photos__TOKEN", Value: "app"},
	}}
	var command string
	if runtime.GOOS == "windows" {
		command = `if (-not $env:BWS_ACCESS_TOKEN) { Write-Output $env:TOKEN }`
	} else {
		command = `test -z "$BWS_ACCESS_TOKEN" && printf '%s' "$TOKEN"`
	}
	stdout, _, err := executeForTest(t, client, "", "run", "photos", "--include-shared", "--", command)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimRight(stdout, "\r\n") != "app" {
		t.Fatalf("unexpected child output: %q", stdout)
	}
}

func TestRunPropagatesExitCode(t *testing.T) {
	client := &fakeClient{}
	_, _, err := executeForTest(t, client, "", "run", "photos", "--", "exit 23")
	var exitErr *exitCodeError
	if !errors.As(err, &exitErr) || exitErr.Code != 23 {
		t.Fatalf("expected exit code 23, got %v", err)
	}
}

func TestBuildEnvironmentUsesNormalizedUUIDs(t *testing.T) {
	entries := []environment.Entry{{Secret: bws.Secret{ID: "64246aa4-70b3-4332-8587-8b1284ce6d76", Key: "TOKEN", Value: "secret"}}}
	env := buildEnvironment(entries, true, true)
	want := "_64246aa4_70b3_4332_8587_8b1284ce6d76=secret"
	if !contains(env, want) {
		t.Fatalf("UUID environment missing %q: %#v", want, env)
	}
}

func TestBuildEnvironmentNeverAddsAccessToken(t *testing.T) {
	t.Setenv("bws_access_token", "inherited-token")
	entries := []environment.Entry{
		{Secret: bws.Secret{Key: "BWS_ACCESS_TOKEN", Value: "secret-token"}},
		{Secret: bws.Secret{Key: "SAFE", Value: "value"}},
	}
	env := buildEnvironment(entries, false, false)
	for _, pair := range env {
		key := strings.SplitN(pair, "=", 2)[0]
		if strings.EqualFold(key, "BWS_ACCESS_TOKEN") {
			t.Fatalf("child environment contains access token: %q", pair)
		}
	}
	if !contains(env, "SAFE=value") {
		t.Fatalf("child environment lost non-reserved secret: %#v", env)
	}
}

func TestRunWithoutCommandOnTerminalFailsBeforeListing(t *testing.T) {
	client := &fakeClient{}
	var stdout, stderr bytes.Buffer
	deps := &runtimeDeps{
		client: client,
		stdin:  strings.NewReader(""),
		stdout: &stdout,
		stderr: &stderr,
		getenv: func(key string) string {
			if key == "BWS_PROJECT_ID" {
				return "project-id"
			}
			return ""
		},
		stdinIsTerminal: func() bool { return true },
	}
	root := newRootCommand(deps)
	root.SetArgs([]string{"run", "photos"})
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "no command provided") {
		t.Fatalf("expected missing-command error, got %v", err)
	}
	if client.listCalls != 0 {
		t.Fatalf("run fetched secrets before validating the command: list calls=%d", client.listCalls)
	}
}
