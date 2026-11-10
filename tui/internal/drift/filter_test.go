package drift_test

import (
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
)

func TestFilterExcludes(t *testing.T) {
	tests := []struct {
		name     string
		filter   drift.Filter
		excluded []string
		kept     []string
	}{
		{"install", drift.InstallFilter, []string{"__pycache__", "x.pyc"}, []string{"a.md", ".git", ".env"}},
		{"pull", drift.PullFilter, []string{
			".git", "node_modules", "auth.json", "credentials.json", ".env", ".env.local",
			"a.pem", "a.key", ".DS_Store", "__pycache__", "x.pyc",
		}, []string{"a.md", "environment.md"}},
		{"zero", drift.Filter{}, nil, []string{"__pycache__", ".git", "a.md"}},
	}
	for _, tt := range tests {
		f := tt.filter
		for _, base := range tt.excluded {
			t.Run(tt.name+" excludes "+base, func(t *testing.T) {
				if !f.Excludes(base) {
					t.Fatalf("Excludes(%q) = false, want true", base)
				}
			})
		}
		for _, base := range tt.kept {
			t.Run(tt.name+" keeps "+base, func(t *testing.T) {
				if f.Excludes(base) {
					t.Fatalf("Excludes(%q) = true, want false", base)
				}
			})
		}
	}
}
