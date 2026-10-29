package environment

import (
	"bwenv/internal/bws"
	"sort"
	"strings"
)

// AppRecord summarizes a stored namespace without exposing secret values.
type AppRecord struct {
	App  string `json:"app" yaml:"app"`
	Keys int    `json:"keys" yaml:"keys"`
}

// CollectApps returns validated namespaces in sorted order.
func CollectApps(secrets []bws.Secret, includeShared bool) []AppRecord {
	counts := make(map[string]int)
	for _, secret := range secrets {
		app, key, ok := strings.Cut(secret.Key, Separator)
		if !ok || app == "" || key == "" {
			continue
		}
		if app == SharedNamespace && !includeShared {
			continue
		}
		if err := ValidateApp(app); err != nil {
			continue
		}
		if err := ValidateKey(key); err != nil {
			continue
		}
		counts[app]++
	}
	apps := make([]string, 0, len(counts))
	for app := range counts {
		apps = append(apps, app)
	}
	sort.Strings(apps)
	records := make([]AppRecord, 0, len(apps))
	for _, app := range apps {
		records = append(records, AppRecord{App: app, Keys: counts[app]})
	}
	return records
}
