package specter_test

import (
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func referencesOf(t *testing.T, data []byte) []string {
	t.Helper()
	var cfg struct {
		References []string `yaml:"references"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("result is not valid YAML: %v\n%s", err, data)
	}
	return cfg.References
}

func TestAddReference(t *testing.T) {
	tests := []struct {
		name string
		old  string
		want string
	}{
		{"flow list", "schema: spec-driven\nreferences: [a, b]\n", "schema: spec-driven\nreferences: [a, b, c]\n"},
		{"empty flow list", "references: []\n", "references: [c]\n"},
		{"block list", "references:\n  - a\n  - b\nother: 1\n", "references:\n  - a\n  - b\n  - c\nother: 1\n"},
		{"comments are kept", "# peers\nreferences: [a, b] # keep me\n# tail\n", "# peers\nreferences: [a, b, c] # keep me\n# tail\n"},
		{"id already present", "references: [a, c]\n", "references: [a, c]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := specter.AddReference([]byte(tt.old), "c")

			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestAddReference_MissingKeyIsAppended(t *testing.T) {
	old := "schema: spec-driven\n# context goes here\n"

	got, err := specter.AddReference([]byte(old), "c")

	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), old) {
		t.Errorf("got\n%s\nwant the original bytes kept as the prefix", got)
	}
	if refs := referencesOf(t, got); !reflect.DeepEqual(refs, []string{"c"}) {
		t.Errorf("references = %v, want [c]", refs)
	}
}

func TestSetReferences(t *testing.T) {
	tests := []struct {
		name string
		old  string
		want string
	}{
		{"flow list", "schema: spec-driven\nreferences: [z]\n", "schema: spec-driven\nreferences: [a, b]\n"},
		{"block list", "references:\n  - z\nother: 1\n", "references:\n  - a\n  - b\nother: 1\n"},
		{"comments are kept", "# peers\nreferences: [z] # keep me\n", "# peers\nreferences: [a, b] # keep me\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := specter.SetReferences([]byte(tt.old), []string{"a", "b"})

			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestSetReferences_MissingKeyIsAppended(t *testing.T) {
	old := "schema: spec-driven\n"

	got, err := specter.SetReferences([]byte(old), []string{"a", "b"})

	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(got), old) {
		t.Errorf("got\n%s\nwant the original bytes kept as the prefix", got)
	}
	if refs := referencesOf(t, got); !reflect.DeepEqual(refs, []string{"a", "b"}) {
		t.Errorf("references = %v, want [a b]", refs)
	}
}
