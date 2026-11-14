package check_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

var envRefPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-[^}]*)?\}`)

func TestEnvRefs(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		env         map[string]string
		want        []check.Status
		wantMessage string
	}{
		{"unset warns naming variable", "token ${EXAMPLE_TOKEN}", nil, []check.Status{check.StatusWarning}, "EXAMPLE_TOKEN"},
		{"set is ok", "token ${EXAMPLE_TOKEN}", map[string]string{"EXAMPLE_TOKEN": "v"}, []check.Status{check.StatusOK}, ""},
		{"unset with default is ok", "${EXAMPLE_TOKEN:-fallback}", nil, []check.Status{check.StatusOK}, ""},
		{"bare dollar is ignored", "$EXAMPLE_TOKEN", nil, nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := check.Env{LookupEnv: func(k string) (string, bool) { v, ok := tt.env[k]; return v, ok }}
			extract := func(check.Env) ([]check.TextSource, error) {
				return []check.TextSource{{Subject: "github", Text: tt.text}}, nil
			}

			got := runCheck(t, check.EnvRefs("mcp-env", envRefPattern, extract), env)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d results, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, r := range got {
				if r.CheckID != "mcp-env" || r.Subject != "github" {
					t.Errorf("result %+v: want CheckID mcp-env, Subject github", r)
				}
				if r.Status != tt.want[i] {
					t.Errorf("result %d status = %v, want %v", i, r.Status, tt.want[i])
				}
				if !strings.Contains(r.Message, tt.wantMessage) {
					t.Errorf("message %q lacks %q", r.Message, tt.wantMessage)
				}
			}
		})
	}
}

func TestEnvRefs_NeverLeaksValue(t *testing.T) {
	const secret = "s3cr3t-value-do-not-print"
	env := check.Env{LookupEnv: func(k string) (string, bool) {
		if k == "EXAMPLE_TOKEN" {
			return secret, true
		}
		return "", false
	}}
	extract := func(check.Env) ([]check.TextSource, error) {
		return []check.TextSource{{Subject: "github", Text: "${EXAMPLE_TOKEN} ${OTHER}"}}, nil
	}

	got := runCheck(t, check.EnvRefs("mcp-env", envRefPattern, extract), env)

	if len(got) == 0 {
		t.Fatal("no results")
	}
	dump, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dump), secret) {
		t.Fatalf("secret value leaked into results: %s", dump)
	}
}

func TestEnvRefs_UnsetSuggestsSettingVariable(t *testing.T) {
	env := check.Env{LookupEnv: func(string) (string, bool) { return "", false }}
	extract := func(check.Env) ([]check.TextSource, error) {
		return []check.TextSource{{Subject: "github", Text: "${EXAMPLE_TOKEN}"}}, nil
	}

	got := runCheck(t, check.EnvRefs("mcp-env", envRefPattern, extract), env)

	if len(got) != 1 || !strings.Contains(got[0].Suggestion, "set EXAMPLE_TOKEN") {
		t.Fatalf("got %+v, want one warning suggesting to set EXAMPLE_TOKEN", got)
	}
}
