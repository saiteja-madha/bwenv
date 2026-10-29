package cmd

import (
	"bwenv/internal/bws"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type failingAppsClient struct{ fakeClient }

func (*failingAppsClient) ListSecrets(context.Context, string) ([]bws.Secret, error) {
	return nil, errors.New("remote unavailable")
}

func TestAppsDiscovery(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{
		{Key: "zoo__TOKEN", Value: "PRIVATE_VALUE", Note: "PRIVATE_NOTE"},
		{Key: "alpha__A"}, {Key: "alpha__B"}, {Key: "shared__TZ"},
		{Key: "unprefixed"}, {Key: "bad app__X"}, {Key: "alpha__bad-key"},
		{Key: "__EMPTY"}, {Key: "alpha__"},
	}}
	stdout, _, err := executeForTest(t, client, "", "apps", "--output", "json")
	if err != nil {
		t.Fatal(err)
	}
	var records []struct {
		App  string
		Keys int
	}
	if err := json.Unmarshal([]byte(stdout), &records); err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 || records[0].App != "alpha" || records[0].Keys != 2 || records[1].App != "shared" || records[2].App != "zoo" {
		t.Fatalf("unexpected summaries: %s", stdout)
	}
	if client.listCalls != 1 {
		t.Fatalf("fetches: %d", client.listCalls)
	}
	if strings.Contains(stdout, "PRIVATE") {
		t.Fatal("exposed secret content")
	}
}

func TestAppsFormatsAndEmptyProject(t *testing.T) {
	for _, format := range []string{"json", "yaml", "table", "tsv", "env", "none"} {
		t.Run(format, func(t *testing.T) {
			client := &fakeClient{}
			stdout, _, err := executeForTest(t, client, "", "apps", "--output", format)
			if err != nil {
				t.Fatal(err)
			}
			if format == "json" && strings.TrimSpace(stdout) != "[]" {
				t.Fatalf("empty result: %q", stdout)
			}
			if format == "none" && stdout != "" {
				t.Fatalf("unexpected output: %q", stdout)
			}
			client.secrets = []bws.Secret{{Key: "app__TOKEN", Value: "PRIVATE"}}
			stdout, _, err = executeForTest(t, client, "", "apps", "--output", format)
			if err != nil || strings.Contains(stdout, "PRIVATE") {
				t.Fatalf("output %q, error %v", stdout, err)
			}
		})
	}
}

func TestAppsValidationAndFailure(t *testing.T) {
	client := &fakeClient{}
	_, _, err := executeForTest(t, client, "", "apps", "extra")
	if err == nil || client.listCalls != 0 || exitCodeFor(err) != 1 {
		t.Fatal("expected validation before fetch")
	}
	stdout, _, err := executeForTest(t, &failingAppsClient{}, "", "apps")
	if err == nil || stdout != "" || exitCodeFor(err) != 1 {
		t.Fatalf("remote failure: %q, %v", stdout, err)
	}
}

func TestListAllDeprecationPreservesSummary(t *testing.T) {
	client := &fakeClient{secrets: []bws.Secret{{Key: "app__TOKEN", Value: "PRIVATE"}, {Key: "shared__TZ"}}}
	stdout, stderr, err := executeForTest(t, client, "", "list", "--all", "--output", "json")
	if err != nil || !strings.Contains(stderr, "deprecated") || !strings.Contains(stderr, "bwenv apps") {
		t.Fatalf("missing deprecation: %v %q", err, stderr)
	}
	if strings.Contains(stdout, "PRIVATE") || strings.Contains(stdout, "shared") {
		t.Fatalf("legacy output changed: %s", stdout)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatal("warning corrupted JSON")
	}
}
