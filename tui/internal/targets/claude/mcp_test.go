package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
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

const ownedSource = `{
  "github": {"command": "npx", "args": ["-y", "x"], "env": {"T": "${GITHUB_TOKEN}"}},
  "engram": {"command": "engram", "url": "http://h?a=1&b=2"}
}`

const ownedGithubJSON = `{"command": "npx", "args": ["-y", "x"], "env": {"T": "${GITHUB_TOKEN}"}}`
const ownedEngramJSON = `{"command": "engram", "url": "http://h?a=1&b=2"}`

func ownedFromSource(t *testing.T) map[string]json.RawMessage {
	t.Helper()
	var owned map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ownedSource), &owned); err != nil {
		t.Fatal(err)
	}
	return owned
}

func decodeAny(t *testing.T, data []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, data)
	}
	return v
}

func mustDecodeString(t *testing.T, s string) any {
	t.Helper()
	return decodeAny(t, []byte(s))
}

func compactJSON(t *testing.T, data []byte) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func rawTopLevel(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	return top
}

func TestMergeServers(t *testing.T) {
	tests := []struct {
		name     string
		old      string
		want     string
		wantErr  bool
		contains []string
	}{
		{"other key and foreign server kept", `{"theme": "dark", "mcpServers": {"mine": {"command": "custom"}}}`,
			`{"theme": "dark", "mcpServers": {"mine": {"command": "custom"}, "github": ` + ownedGithubJSON + `, "engram": ` + ownedEngramJSON + `}}`, false, nil},
		{"owned overrides same-named server", `{"mcpServers": {"github": {"command": "old"}}}`,
			`{"mcpServers": {"github": ` + ownedGithubJSON + `, "engram": ` + ownedEngramJSON + `}}`, false, nil},
		{"missing file starts from empty object", ``,
			`{"mcpServers": {"github": ` + ownedGithubJSON + `, "engram": ` + ownedEngramJSON + `}}`, false, nil},
		{"invalid JSON is an error", `{"mcpServers": `, "", true, nil},
		{"ampersand and variable stay literal", `{}`,
			`{"mcpServers": {"github": ` + ownedGithubJSON + `, "engram": ` + ownedEngramJSON + `}}`, false, []string{"a=1&b=2", "${GITHUB_TOKEN}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var old []byte
			if tt.old != "" {
				old = []byte(tt.old)
			}

			got, err := MergeServers(old, ownedFromSource(t))

			if tt.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decodeAny(t, got), mustDecodeString(t, tt.want)) {
				t.Errorf("output = %s, want %s", got, tt.want)
			}
			if !bytes.HasSuffix(got, []byte("}\n")) {
				t.Errorf("output must end with a newline: %q", got)
			}
			for _, s := range tt.contains {
				if !strings.Contains(string(got), s) {
					t.Errorf("output lacks literal %q:\n%s", s, got)
				}
			}
		})
	}
}

func TestReplaceServers(t *testing.T) {
	const projects = `{"/p": {"b": 1, "a": 1.50}}`
	wantOwned := `{"github": ` + ownedGithubJSON + `, "engram": ` + ownedEngramJSON + `}`
	tests := []struct {
		name     string
		old      string
		wantTop  string
		wantErr  bool
		contains []string
	}{
		{"foreign server dropped", `{"projects": ` + projects + `, "mcpServers": {"mine": {"command": "custom"}, "github": {"command": "old"}}}`,
			wantOwned, false, nil},
		{"missing file creates owned keys", ``, wantOwned, false, nil},
		{"invalid JSON is an error", `{"mcpServers": `, "", true, nil},
		{"ampersand and variable stay literal", `{"mcpServers": {}}`, wantOwned, false, []string{"a=1&b=2", "${GITHUB_TOKEN}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var old []byte
			if tt.old != "" {
				old = []byte(tt.old)
			}

			got, err := ReplaceServers(old, ownedFromSource(t))

			if tt.wantErr {
				if err == nil {
					t.Fatal("want error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			servers := rawTopLevel(t, got)["mcpServers"]
			if !reflect.DeepEqual(decodeAny(t, servers), mustDecodeString(t, tt.wantTop)) {
				t.Errorf("mcpServers = %s, want %s", servers, tt.wantTop)
			}
			if !bytes.HasSuffix(got, []byte("}\n")) {
				t.Errorf("output must end with a newline: %q", got)
			}
			for _, s := range tt.contains {
				if !strings.Contains(string(got), s) {
					t.Errorf("output lacks literal %q:\n%s", s, got)
				}
			}
		})
	}
}

func TestReplaceServers_KeepsOtherTopLevelKeysExact(t *testing.T) {
	const projects = `{"z": 1, "a": 1.50, "/p": {"k": [1, 2]}}`
	old := `{"projects": ` + projects + `, "mcpServers": {"mine": {"command": "custom"}}, "theme": "dark"}`

	got, err := ReplaceServers([]byte(old), ownedFromSource(t))
	if err != nil {
		t.Fatal(err)
	}

	top := rawTopLevel(t, got)
	if compactJSON(t, top["projects"]) != compactJSON(t, []byte(projects)) {
		t.Errorf("projects = %s, want %s", top["projects"], projects)
	}
	if string(top["theme"]) != `"dark"` {
		t.Errorf("theme = %s", top["theme"])
	}
}

func setupMCP(t *testing.T, target *string) check.Env {
	t.Helper()
	env := newEnv(t)
	writeFile(t, env.Repo, mcpSourceRel, ownedSource)
	if target != nil {
		writeFile(t, env.Home, mcpTargetRel, *target)
	}
	return env
}

func onlyEdit(t *testing.T, part plan.Part) plan.Edit {
	t.Helper()
	if len(part.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(part.Steps))
	}
	edit, ok := part.Steps[0].(plan.Edit)
	if !ok {
		t.Fatalf("step is %T, want plan.Edit", part.Steps[0])
	}
	return edit
}

func serverNames(t *testing.T, out []byte) []string {
	t.Helper()
	var doc struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(maps.Keys(doc.MCPServers))
}

func TestInstallMCP(t *testing.T) {
	t.Run("edit merges owned servers and keeps the rest", func(t *testing.T) {
		env := setupMCP(t, ptr(`{"theme": "dark", "mcpServers": {"mine": {"command": "custom"}}}`))

		part, err := installMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		edit := onlyEdit(t, part)
		if want := filepath.Join(env.Home, mcpTargetRel); edit.Path != want {
			t.Errorf("Path = %q, want %q", edit.Path, want)
		}
		out, err := edit.Edit([]byte(`{"theme": "dark", "mcpServers": {"mine": {"command": "custom"}}}`), true)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := serverNames(t, out), []string{"engram", "github", "mine"}; !slices.Equal(got, want) {
			t.Errorf("servers = %v, want %v", got, want)
		}
	})

	t.Run("plan time invalid JSON fails early", func(t *testing.T) {
		env := setupMCP(t, ptr(`{"mcpServers": `))

		_, err := installMCP(context.Background(), env, mcpComponent())

		if err == nil || !strings.Contains(err.Error(), mcpTargetRel) {
			t.Fatalf("err = %v, want one naming %s", err, mcpTargetRel)
		}
	})

	t.Run("apply time invalid JSON names the file", func(t *testing.T) {
		env := setupMCP(t, nil)
		part, err := installMCP(context.Background(), env, mcpComponent())
		if err != nil {
			t.Fatal(err)
		}

		_, err = onlyEdit(t, part).Edit([]byte(`{"mcpServers": `), true)

		if err == nil || !strings.Contains(err.Error(), mcpTargetRel) {
			t.Fatalf("err = %v, want one naming %s", err, mcpTargetRel)
		}
	})

	t.Run("in sync gives no step", func(t *testing.T) {
		env := setupMCP(t, ptr(`{"mcpServers": {"engram": `+ownedEngramJSON+`, "github": `+ownedGithubJSON+`, "mine": {"command": "custom"}}}`))

		part, err := installMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		if len(part.Steps) != 0 {
			t.Errorf("steps = %d, want 0", len(part.Steps))
		}
	})

	t.Run("missing file still plans an edit", func(t *testing.T) {
		env := setupMCP(t, nil)

		part, err := installMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		out, err := onlyEdit(t, part).Edit(nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := serverNames(t, out), []string{"engram", "github"}; !slices.Equal(got, want) {
			t.Errorf("servers = %v, want %v", got, want)
		}
	})
}

func TestRemoveMCP(t *testing.T) {
	t.Run("deletes only owned keys with one row per key", func(t *testing.T) {
		const target = `{"theme": "dark", "mcpServers": {"github": {"command": "old"}, "mine": {"command": "custom"}}}`
		env := setupMCP(t, ptr(target))

		part, err := removeMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		edit := onlyEdit(t, part)
		wantRows := []plan.Change{{Effect: plan.EffectDelete, Path: "~/.claude.json#mcpServers.github"}}
		if !slices.Equal(edit.Rows, wantRows) {
			t.Errorf("Rows = %+v, want %+v", edit.Rows, wantRows)
		}
		out, err := edit.Edit([]byte(target), true)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := serverNames(t, out), []string{"mine"}; !slices.Equal(got, want) {
			t.Errorf("servers = %v, want %v", got, want)
		}
		if string(rawTopLevel(t, out)["theme"]) != `"dark"` {
			t.Errorf("theme lost: %s", out)
		}
	})

	for _, tt := range []struct {
		name   string
		target *string
	}{
		{"missing file gives no step", nil},
		{"no owned key gives no step", ptr(`{"mcpServers": {"mine": {"command": "custom"}}}`)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := setupMCP(t, tt.target)

			part, err := removeMCP(context.Background(), env, mcpComponent())

			if err != nil {
				t.Fatal(err)
			}
			if len(part.Steps) != 0 {
				t.Errorf("steps = %d, want 0", len(part.Steps))
			}
		})
	}
}

func TestHardInstallMCP(t *testing.T) {
	t.Run("drops foreign server and lists one delete row for it", func(t *testing.T) {
		const target = `{"projects": {"/p": 1}, "mcpServers": {"mine": {"command": "custom"}, "github": {"command": "old"}}}`
		env := setupMCP(t, ptr(target))

		part, err := hardInstallMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		edit := onlyEdit(t, part)
		if want := filepath.Join(env.Home, mcpTargetRel); edit.Path != want {
			t.Errorf("Path = %q, want %q", edit.Path, want)
		}
		var deletes []string
		for _, row := range edit.Rows {
			if row.Effect == plan.EffectDelete {
				deletes = append(deletes, row.Path)
			}
		}
		if want := []string{"~/.claude.json#mcpServers.mine"}; !slices.Equal(deletes, want) {
			t.Errorf("delete rows = %v, want %v", deletes, want)
		}
		out, err := edit.Edit([]byte(target), true)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := serverNames(t, out), []string{"engram", "github"}; !slices.Equal(got, want) {
			t.Errorf("servers = %v, want %v", got, want)
		}
		if compactJSON(t, rawTopLevel(t, out)["projects"]) != `{"/p":1}` {
			t.Errorf("projects changed: %s", out)
		}
	})

	t.Run("missing file creates owned keys without delete rows", func(t *testing.T) {
		env := setupMCP(t, nil)

		part, err := hardInstallMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		edit := onlyEdit(t, part)
		for _, row := range edit.Rows {
			if row.Effect == plan.EffectDelete {
				t.Errorf("unexpected delete row %+v", row)
			}
		}
		out, err := edit.Edit(nil, false)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := serverNames(t, out), []string{"engram", "github"}; !slices.Equal(got, want) {
			t.Errorf("servers = %v, want %v", got, want)
		}
	})

	t.Run("plan time invalid JSON fails early", func(t *testing.T) {
		env := setupMCP(t, ptr(`{"mcpServers": `))

		_, err := hardInstallMCP(context.Background(), env, mcpComponent())

		if err == nil || !strings.Contains(err.Error(), mcpTargetRel) {
			t.Fatalf("err = %v, want one naming %s", err, mcpTargetRel)
		}
	})

	t.Run("apply time invalid JSON names the file", func(t *testing.T) {
		env := setupMCP(t, nil)
		part, err := hardInstallMCP(context.Background(), env, mcpComponent())
		if err != nil {
			t.Fatal(err)
		}

		_, err = onlyEdit(t, part).Edit([]byte(`{"mcpServers": `), true)

		if err == nil || !strings.Contains(err.Error(), mcpTargetRel) {
			t.Fatalf("err = %v, want one naming %s", err, mcpTargetRel)
		}
	})

	t.Run("no step when mcpServers already equals the owned set", func(t *testing.T) {
		env := setupMCP(t, ptr(`{"theme": "dark", "mcpServers": {"engram": `+ownedEngramJSON+`, "github": `+ownedGithubJSON+`}}`))

		part, err := hardInstallMCP(context.Background(), env, mcpComponent())

		if err != nil {
			t.Fatal(err)
		}
		if len(part.Steps) != 0 {
			t.Errorf("steps = %d, want 0", len(part.Steps))
		}
	})
}
