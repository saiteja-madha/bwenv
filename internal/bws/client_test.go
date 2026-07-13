package bws

import (
	"testing"
)

func TestValidateValue(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty string", "", false},
		{"normal string", "hello", false},
		{"with spaces", "hello world", false},
		{"special chars", `p@ssw0rd!#$%&'()*+,-./:;<=>?@[]^_{|}~`, false},
		{"unicode", "héllo 日本語", false},
		{"multiline", "line1\nline2\nline3", false},
		{"with equals", "foo=bar=baz", false},
		{"null byte", "secret\x00hidden", true},
		{"null byte at start", "\x00secret", true},
		{"null byte at end", "secret\x00", true},
		{"only null byte", "\x00", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateValue() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFilterEnvLines(t *testing.T) {
	secrets := []Secret{
		{ID: "1", Key: "myapp__DATABASE_URL", Value: "postgres://localhost"},
		{ID: "2", Key: "myapp__API_KEY", Value: "secret123"},
		{ID: "3", Key: "shared__LOG_LEVEL", Value: "debug"},
		{ID: "4", Key: "other__TOKEN", Value: "should-not-appear"},
		{ID: "5", Key: "", Value: "empty-key"},
	}

	t.Run("app specific", func(t *testing.T) {
		lines := FilterEnvLines(secrets, "myapp", false)
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
		}
		expected := map[string]string{
			"DATABASE_URL": "postgres://localhost",
			"API_KEY":      "secret123",
		}
		for _, line := range lines {
			parts := splitLine(line)
			if parts == nil {
				t.Fatalf("malformed line: %s", line)
			}
			if expected[parts.key] != parts.value {
				t.Errorf("for key %s: expected %q, got %q", parts.key, expected[parts.key], parts.value)
			}
		}
	})

	t.Run("with shared", func(t *testing.T) {
		lines := FilterEnvLines(secrets, "myapp", true)
		if len(lines) != 3 {
			t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
		}
	})

	t.Run("no secrets", func(t *testing.T) {
		lines := FilterEnvLines(nil, "nonexistent", false)
		if len(lines) != 0 {
			t.Fatalf("expected 0 lines, got %d", len(lines))
		}
	})
}

func TestFilterAppKeys(t *testing.T) {
	secrets := []Secret{
		{ID: "1", Key: "myapp__DATABASE_URL", Value: "postgres://localhost"},
		{ID: "2", Key: "myapp__API_KEY", Value: "secret123"},
		{ID: "3", Key: "other__TOKEN", Value: "should-not-appear"},
	}

	t.Run("app keys", func(t *testing.T) {
		keys := FilterAppKeys(secrets, "myapp")
		if len(keys) != 2 {
			t.Fatalf("expected 2 keys, got %d: %v", len(keys), keys)
		}
		expected := map[string]bool{"DATABASE_URL": true, "API_KEY": true}
		for _, k := range keys {
			if !expected[k] {
				t.Errorf("unexpected key: %s", k)
			}
		}
	})

	t.Run("no keys", func(t *testing.T) {
		keys := FilterAppKeys(nil, "nonexistent")
		if len(keys) != 0 {
			t.Fatalf("expected 0 keys, got %d", len(keys))
		}
	})
}

type lineParts struct {
	key   string
	value string
}

func splitLine(line string) *lineParts {
	for i := 0; i < len(line); i++ {
		if line[i] == '=' {
			return &lineParts{key: line[:i], value: line[i+1:]}
		}
	}
	return nil
}
