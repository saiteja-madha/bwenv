package environment

import (
	"strings"
	"testing"

	"bwenv/internal/bws"
)

func fixture() []bws.Secret {
	return []bws.Secret{
		{ID: "app-db", Key: "photos__DATABASE_URL", Value: "app-db"},
		{ID: "app-port", Key: "photos__PORT", Value: "8080"},
		{ID: "shared-db", Key: "shared__DATABASE_URL", Value: "shared-db"},
		{ID: "shared-log", Key: "shared__LOG_LEVEL", Value: "info"},
		{ID: "other", Key: "wiki__TOKEN", Value: "ignored"},
	}
}

func TestFullKeyValidation(t *testing.T) {
	valid := []struct{ app, key string }{{"photos", "API_KEY"}, {"home-assistant", "PORT"}, {"app.v2", "_TOKEN"}}
	for _, tt := range valid {
		if _, err := FullKey(tt.app, tt.key); err != nil {
			t.Errorf("FullKey(%q, %q): %v", tt.app, tt.key, err)
		}
	}
	invalid := []struct{ app, key string }{{"", "KEY"}, {"bad__app", "KEY"}, {"bad app", "KEY"}, {"app", ""}, {"app", "1KEY"}, {"app", "BAD-KEY"}}
	for _, tt := range invalid {
		if _, err := FullKey(tt.app, tt.key); err == nil {
			t.Errorf("FullKey(%q, %q) unexpectedly succeeded", tt.app, tt.key)
		}
	}
}

func TestMergeSharedWithAppPrecedence(t *testing.T) {
	entries, err := Merge(fixture(), "photos", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries: %#v", len(entries), entries)
	}
	if entries[0].Key != "DATABASE_URL" || entries[0].Value != "app-db" || entries[0].Source != "app" {
		t.Fatalf("app value did not override shared: %#v", entries[0])
	}
	if entries[1].Key != "LOG_LEVEL" || entries[1].Source != "shared" || entries[2].Key != "PORT" {
		t.Fatalf("unexpected sorted merge: %#v", entries)
	}
}

func TestMergeSharedAppDoesNotDuplicate(t *testing.T) {
	entries, err := Merge(fixture(), "shared", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("shared app duplicated entries: %#v", entries)
	}
}

func TestGetFallsBackToShared(t *testing.T) {
	entry, err := Get(fixture(), "photos", "LOG_LEVEL", true)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Source != "shared" || entry.Value != "info" {
		t.Fatalf("unexpected shared fallback: %#v", entry)
	}
}

func TestDuplicateSourceIsRejected(t *testing.T) {
	secrets := append(fixture(), bws.Secret{ID: "duplicate", Key: "photos__PORT", Value: "9090"})
	_, err := Merge(secrets, "photos", false)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestMalformedStoredKeyIsRejected(t *testing.T) {
	_, err := Merge([]bws.Secret{{ID: "bad", Key: "photos__BAD-KEY"}}, "photos", false)
	if err == nil || !strings.Contains(err.Error(), "invalid environment key") {
		t.Fatalf("expected invalid key error, got %v", err)
	}
}
