// Package environment maps Bitwarden secrets to app-scoped environment entries.
package environment

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"bwenv/internal/bws"
)

const separator = "__"

var (
	appPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	keyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// NotFoundError reports a missing app/key combination.
type NotFoundError struct {
	App string
	Key string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("environment key %s/%s not found", e.App, e.Key)
}

// Entry is a Bitwarden secret normalized for one application environment.
type Entry struct {
	bws.Secret `yaml:",inline"`
	Source     string `json:"source" yaml:"source"`
}

// ValidateApp checks an application namespace.
func ValidateApp(app string) error {
	if !appPattern.MatchString(app) || strings.Contains(app, separator) {
		return fmt.Errorf("invalid app %q: use letters, numbers, '.', '_' or '-' and do not use %q", app, separator)
	}
	return nil
}

// ValidateKey checks that a key is a portable environment variable name.
func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("invalid environment key %q: expected [A-Za-z_][A-Za-z0-9_]*", key)
	}
	return nil
}

// FullKey validates and joins an app and environment key.
func FullKey(app, key string) (string, error) {
	if err := ValidateApp(app); err != nil {
		return "", err
	}
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	return app + separator + key, nil
}

// Resolve returns exactly one app-specific secret.
func Resolve(secrets []bws.Secret, app, key string) (bws.Secret, error) {
	fullKey, err := FullKey(app, key)
	if err != nil {
		return bws.Secret{}, err
	}
	var matches []bws.Secret
	for _, secret := range secrets {
		if secret.Key == fullKey {
			matches = append(matches, secret)
		}
	}
	if len(matches) == 0 {
		return bws.Secret{}, &NotFoundError{App: app, Key: key}
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, match.ID)
		}
		return bws.Secret{}, fmt.Errorf("environment key %s/%s is ambiguous; matching secret IDs: %s", app, key, strings.Join(ids, ", "))
	}
	return matches[0], nil
}

// Get resolves app-specific first, then shared when requested.
func Get(secrets []bws.Secret, app, key string, includeShared bool) (Entry, error) {
	secret, err := Resolve(secrets, app, key)
	if err == nil {
		return normalize(secret, app, "app"), nil
	}
	var notFound *NotFoundError
	if !includeShared || app == "shared" || !errors.As(err, &notFound) {
		return Entry{}, err
	}
	secret, sharedErr := Resolve(secrets, "shared", key)
	if sharedErr != nil {
		if errors.As(sharedErr, &notFound) {
			return Entry{}, err
		}
		return Entry{}, sharedErr
	}
	return normalize(secret, "shared", "shared"), nil
}

// Merge builds the effective environment. Shared entries are loaded first and
// app entries override them. Duplicate keys within either source are errors.
func Merge(secrets []bws.Secret, app string, includeShared bool) ([]Entry, error) {
	if err := ValidateApp(app); err != nil {
		return nil, err
	}
	entries := make(map[string]Entry)
	if includeShared && app != "shared" {
		shared, err := collect(secrets, "shared", "shared")
		if err != nil {
			return nil, err
		}
		for key, entry := range shared {
			entries[key] = entry
		}
	}
	appEntries, err := collect(secrets, app, "app")
	if err != nil {
		return nil, err
	}
	for key, entry := range appEntries {
		entries[key] = entry
	}

	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]Entry, 0, len(keys))
	for _, key := range keys {
		result = append(result, entries[key])
	}
	return result, nil
}

func collect(secrets []bws.Secret, app, source string) (map[string]Entry, error) {
	prefix := app + separator
	entries := make(map[string]Entry)
	for _, secret := range secrets {
		if !strings.HasPrefix(secret.Key, prefix) {
			continue
		}
		key := strings.TrimPrefix(secret.Key, prefix)
		if err := ValidateKey(key); err != nil {
			return nil, fmt.Errorf("secret %s has %w", secret.ID, err)
		}
		if existing, ok := entries[key]; ok {
			return nil, fmt.Errorf("duplicate environment key %s/%s has secret IDs %s and %s", app, key, existing.ID, secret.ID)
		}
		entries[key] = normalize(secret, app, source)
	}
	return entries, nil
}

func normalize(secret bws.Secret, app, source string) Entry {
	secret.Key = strings.TrimPrefix(secret.Key, app+separator)
	return Entry{Secret: secret, Source: source}
}
