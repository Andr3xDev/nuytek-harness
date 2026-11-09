package claude

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

const (
	mcpSourceRel = "claude/claude-mcp.json"
	mcpTargetRel = ".claude.json"
)

const mcpSource = `{
  "github": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-github"],
    "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"}
  },
  "linear-server": {"type": "http", "url": "https://mcp.linear.app/mcp", "headers": {"Authorization": "Bearer ${LINEAR_KEY}"}},
  "engram": {"command": "engram", "args": ["mcp"], "type": "${IGNORED_KEY}"}
}`

func mcpComponent() manifest.Component {
	return manifest.Component{Name: "mcp", Kind: "mcp", Source: mcpSourceRel, Target: mcpTargetRel}
}

func TestMCPEnvTexts(t *testing.T) {
	env := newEnv(t)
	writeFile(t, env.Repo, mcpSourceRel, mcpSource)

	got, err := mcpEnvTexts(mcpSourceRel)(env)
	if err != nil {
		t.Fatal(err)
	}

	var subjects []string
	texts := map[string]string{}
	for _, src := range got {
		subjects = append(subjects, src.Subject)
		texts[src.Subject] = src.Text
	}
	if want := []string{"engram", "github", "linear-server"}; !slices.Equal(subjects, want) {
		t.Fatalf("subjects = %v, want %v", subjects, want)
	}
	for subject, want := range map[string]string{
		"github":        "${GITHUB_TOKEN}",
		"linear-server": "${LINEAR_KEY}",
	} {
		if !strings.Contains(texts[subject], want) {
			t.Errorf("%s text %q lacks %q", subject, texts[subject], want)
		}
	}
	if strings.Contains(texts["engram"], "IGNORED_KEY") {
		t.Errorf("engram text %q includes a key outside command/args/env/url/headers", texts["engram"])
	}
}

func TestMCPEnvTexts_InvalidSourceIsError(t *testing.T) {
	env := newEnv(t)
	writeFile(t, env.Repo, mcpSourceRel, `{"github": `)

	if _, err := mcpEnvTexts(mcpSourceRel)(env); err == nil {
		t.Fatal("want error for invalid MCP source")
	}
}

func TestMCPCommandRequirements_SkipsServersWithoutCommand(t *testing.T) {
	env := newEnv(t)
	writeFile(t, env.Repo, mcpSourceRel, mcpSource)

	got, err := mcpCommandRequirements(mcpSourceRel)(env)
	if err != nil {
		t.Fatal(err)
	}

	want := []check.Requirement{
		{Subject: "engram", Command: "engram"},
		{Subject: "github", Command: "npx"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("requirements = %+v, want %+v", got, want)
	}
}

func TestEvaluateMCP(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		target       *string
		want         drift.State
		reasonSubstr string
	}{
		{"owned keys equal plus user server", mcpSource, ptr(`{"mcpServers": {
			"github": {"command": "npx", "args": ["-y", "@modelcontextprotocol/server-github"], "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"}},
			"linear-server": {"type": "http", "url": "https://mcp.linear.app/mcp", "headers": {"Authorization": "Bearer ${LINEAR_KEY}"}},
			"engram": {"command": "engram", "args": ["mcp"], "type": "${IGNORED_KEY}"},
			"mine": {"command": "custom"}}, "other": 1}`), drift.StateInstalled, ""},
		{"owned value differs", mcpSource, ptr(`{"mcpServers": {
			"github": {"command": "npx", "args": ["-y", "other-server"], "env": {"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"}},
			"linear-server": {"type": "http", "url": "https://mcp.linear.app/mcp", "headers": {"Authorization": "Bearer ${LINEAR_KEY}"}},
			"engram": {"command": "engram", "args": ["mcp"], "type": "${IGNORED_KEY}"}}}`), drift.StateDrift, "github"},
		{"one owned key missing", `{"a": {"command": "x"}, "b": {"command": "y"}}`,
			ptr(`{"mcpServers": {"a": {"command": "x"}}}`), drift.StateDrift, "b"},
		{"no owned key present", mcpSource, ptr(`{"mcpServers": {"mine": {"command": "custom"}}}`), drift.StateNotInstalled, ""},
		{"no mcpServers key", mcpSource, ptr(`{"theme": "dark"}`), drift.StateNotInstalled, ""},
		{"deployed file absent", mcpSource, nil, drift.StateNotInstalled, ""},
		{"deployed file invalid JSON", mcpSource, ptr(`{"mcpServers": `), drift.StateUnknown, ""},
		{"repo source invalid JSON", `{"a": `, ptr(`{"mcpServers": {}}`), drift.StateUnknown, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(t)
			writeFile(t, env.Repo, mcpSourceRel, tt.source)
			if tt.target != nil {
				writeFile(t, env.Home, mcpTargetRel, *tt.target)
			}

			got := evaluateMCP(context.Background(), env, mcpComponent())

			requireState(t, got, tt.want)
			if got.Name != "mcp" {
				t.Errorf("Name = %q, want mcp", got.Name)
			}
			if !strings.Contains(got.Reason, tt.reasonSubstr) {
				t.Errorf("reason %q lacks %q", got.Reason, tt.reasonSubstr)
			}
			if tt.want == drift.StateUnknown && got.Reason == "" {
				t.Error("unknown state must carry the parse reason")
			}
		})
	}
}
