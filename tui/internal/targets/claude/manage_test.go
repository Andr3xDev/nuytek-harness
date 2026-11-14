package claude

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	manageMCPTarget     = ".claude.json"
	manageMCPSource     = "claude/claude-mcp.json"
	manageSkillsTarget  = ".claude/skills"
	manageSkillsSource  = "skills"
	manageSettingsRel   = ".claude/settings.json"
	manageResolvedNode  = "/fake/bin/node"
	manageNoSummaryText = "no summary available"
)

// lookPathExec resolves only the commands in found; nothing else runs.
type lookPathExec map[string]string

func (e lookPathExec) LookPath(name string) (string, error) {
	if p, ok := e[name]; ok {
		return p, nil
	}
	return "", errors.New("not found")
}
func (lookPathExec) Output(context.Context, string, ...string) ([]byte, error) {
	return nil, errors.New("manage must not run commands")
}
func (lookPathExec) Run(context.Context, string, ...string) (check.ExecResult, error) {
	return check.ExecResult{}, errors.New("manage must not run commands")
}

func manageManifest() manifest.Manifest {
	return manifest.Manifest{Components: []manifest.Component{
		{Name: componentMCP, Kind: kindMCP, Source: manageMCPSource, Target: manageMCPTarget},
		{Name: componentSettings, Kind: manifest.KindFile, Source: "claude/claude-settings.json", Target: manageSettingsRel},
		{Name: "plugins", Kind: kindPlugins, Source: pluginsSourceRel, Target: manageSettingsRel},
		{Name: componentSkills, Kind: manifest.KindDir, Source: manageSkillsSource, Target: manageSkillsTarget},
	}}
}

func manageEnv(t *testing.T) check.Env {
	t.Helper()
	env := newEnv(t)
	env.Exec = lookPathExec{"node": manageResolvedNode}
	return env
}

func runManage(t *testing.T, env check.Env, m manifest.Manifest) []target.ManageSection {
	t.Helper()
	if Spec().Manage == nil {
		t.Fatal("claude Spec().Manage is nil")
	}
	return Spec().Manage(context.Background(), env, m)
}

func sectionByTitle(t *testing.T, sections []target.ManageSection, title string) target.ManageSection {
	t.Helper()
	for _, s := range sections {
		if strings.Contains(strings.ToLower(s.Title), title) {
			return s
		}
	}
	t.Fatalf("no section titled like %q in %+v", title, sections)
	return target.ManageSection{}
}

func rowByName(t *testing.T, s target.ManageSection, name string) target.ManageRow {
	t.Helper()
	i := slices.IndexFunc(s.Rows, func(r target.ManageRow) bool { return r.Name == name })
	if i < 0 {
		t.Fatalf("no row %q in %+v", name, s.Rows)
	}
	return s.Rows[i]
}

func TestManage_SectionsShown(t *testing.T) {
	sections := runManage(t, manageEnv(t), manageManifest())
	for _, title := range []string{"mcp", "plugin", "skill"} {
		sectionByTitle(t, sections, title)
	}
}

func TestManageMCPs(t *testing.T) {
	const servers = `{"mcpServers": {
		"owned-srv": {"command": "node", "args": ["a.js"]},
		"ext-srv": {"command": "ghost-cmd"},
		"remote-srv": {"type": "http", "url": "https://example.invalid/mcp"}}}`
	env := manageEnv(t)
	writeFile(t, env.Repo, manageMCPSource, `{"owned-srv": {"command": "node", "args": ["a.js"]}}`)
	writeFile(t, env.Home, manageMCPTarget, servers)
	section := sectionByTitle(t, runManage(t, env, manageManifest()), "mcp")

	t.Run("owned server with resolving command", func(t *testing.T) {
		row := rowByName(t, section, "owned-srv")
		if row.Status != check.StatusOK || !strings.Contains(row.Detail, "harness-owned") || !strings.Contains(row.Detail, manageResolvedNode) {
			t.Fatalf("row = %+v, want ok, harness-owned and the resolved path", row)
		}
	})
	t.Run("external server with missing command", func(t *testing.T) {
		row := rowByName(t, section, "ext-srv")
		if row.Status == check.StatusOK || row.Status == "" || !strings.Contains(row.Detail, "external") || !strings.Contains(row.Detail, "not found") {
			t.Fatalf("row = %+v, want non-ok status, external and not found", row)
		}
	})
	t.Run("server without command", func(t *testing.T) {
		row := rowByName(t, section, "remote-srv")
		if !strings.Contains(row.Detail, "no command") || strings.Contains(row.Detail, "not found") || row.Status == check.StatusWarning || row.Status == check.StatusError {
			t.Fatalf("row = %+v, want `no command` and no failure", row)
		}
	})
}

func TestManageMCPs_EmptyWhenAbsentOrNoServers(t *testing.T) {
	for name, content := range map[string]*string{"absent file": nil, "no servers": ptr(`{"theme": "dark"}`), "empty servers": ptr(`{"mcpServers": {}}`)} {
		t.Run(name, func(t *testing.T) {
			env := manageEnv(t)
			if content != nil {
				writeFile(t, env.Home, manageMCPTarget, *content)
			}
			section := sectionByTitle(t, runManage(t, env, manageManifest()), "mcp")
			if len(section.Rows) != 0 || section.Empty == "" || section.Unknown != "" {
				t.Fatalf("section = %+v, want no rows, an Empty message and no Unknown", section)
			}
		})
	}
}

func TestManageMCPs_InvalidJSONIsUnknownWithReason(t *testing.T) {
	env := manageEnv(t)
	writeFile(t, env.Home, manageMCPTarget, `{"mcpServers": `)
	section := sectionByTitle(t, runManage(t, env, manageManifest()), "mcp")
	if section.Unknown == "" || len(section.Rows) != 0 {
		t.Fatalf("section = %+v, want Unknown with the parse reason and no rows", section)
	}
}

func TestManagePlugins(t *testing.T) {
	const settings = `{"enabledPlugins": {
		"array@m": true, "object@m": true, "badshape@m": true, "gone@m": true, "off@m": false}}`
	const installed = `{"version": 2, "plugins": {
		"array@m": [{"scope": "user", "version": "1.2.3"}, {"version": "0.0.1"}],
		"object@m": {"version": "2.0.0"},
		"badshape@m": "odd",
		"off@m": [{"version": "9.9.9"}]}}`
	env := manageEnv(t)
	writeFile(t, env.Home, manageSettingsRel, settings)
	writeFile(t, env.Home, installedPluginsRel, installed)
	section := sectionByTitle(t, runManage(t, env, manageManifest()), "plugin")

	tests := []struct {
		id        string
		want      []string
		wantNo    []string
		wantOK    bool
		wantNotOK bool
	}{
		{"array@m", []string{"installed", "1.2.3"}, []string{"not installed", "0.0.1"}, true, false},
		{"object@m", []string{"installed", "2.0.0"}, []string{"not installed"}, true, false},
		{"badshape@m", []string{"installed", "version unknown"}, []string{"not installed"}, true, false},
		{"gone@m", []string{"not installed"}, nil, false, true},
		{"off@m", []string{"disabled"}, []string{"not installed", "9.9.9"}, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			row := rowByName(t, section, tc.id)
			for _, w := range tc.want {
				if !strings.Contains(row.Detail, w) {
					t.Errorf("detail %q lacks %q", row.Detail, w)
				}
			}
			for _, w := range tc.wantNo {
				if strings.Contains(row.Detail, w) {
					t.Errorf("detail %q must not contain %q", row.Detail, w)
				}
			}
			if tc.wantOK && row.Status != check.StatusOK {
				t.Errorf("status = %q, want ok", row.Status)
			}
			if tc.wantNotOK && (row.Status == check.StatusOK || row.Status == "") {
				t.Errorf("status = %q, want a failing status", row.Status)
			}
		})
	}
}

func TestManagePlugins_UnparseableInstalledListIsUnknownWithReason(t *testing.T) {
	env := manageEnv(t)
	writeFile(t, env.Home, manageSettingsRel, `{"enabledPlugins": {"p@m": true, "off@m": false}}`)
	writeFile(t, env.Home, installedPluginsRel, `{"plugins": `)
	section := sectionByTitle(t, runManage(t, env, manageManifest()), "plugin")

	row := rowByName(t, section, "p@m")
	if row.Status != check.StatusUnknown || strings.Contains(row.Detail, "not installed") {
		t.Fatalf("row = %+v, want unknown status and not `not installed`", row)
	}
	if off := rowByName(t, section, "off@m"); !strings.Contains(off.Detail, "disabled") {
		t.Fatalf("disabled row = %+v, want disabled even when the installed list is unreadable", off)
	}
}

func TestManageSkills(t *testing.T) {
	env := manageEnv(t)
	writeFile(t, env.Repo, "skills/foo/SKILL.md", "---\nname: foo\ndescription: Does foo things\n---\nbody\n")
	writeFile(t, env.Home, ".claude/skills/foo/SKILL.md", "---\nname: foo\ndescription: Does foo things\n---\nbody\n")
	writeFile(t, env.Home, ".claude/skills/mine/SKILL.md", "---\nname: mine\ndescription: My own skill\n---\n")
	writeFile(t, env.Home, ".claude/skills/nosummary/SKILL.md", "---\nname: nosummary\n---\n")
	writeFile(t, env.Home, ".claude/skills/nofile/notes.txt", "x")
	writeFile(t, env.Home, ".claude/skills/stray.txt", "not a skill")
	writeFile(t, env.Home, ".claude/plugins/cache/mk/pl/1.0.0/skills/pluginskill/SKILL.md", "---\nname: pluginskill\ndescription: from a plugin\n---\n")
	writeFile(t, env.Home, ".claude/settings.json", `{"enabledPlugins": {"pl@mk": true}}`)

	linkTarget := t.TempDir()
	writeFile(t, linkTarget, "SKILL.md", "---\nname: linked\ndescription: Via symlink\n---\n")
	if err := os.Symlink(linkTarget, filepath.Join(env.Home, ".claude/skills/linked")); err != nil {
		t.Fatal(err)
	}

	section := sectionByTitle(t, runManage(t, env, manageManifest()), "skill")

	t.Run("harness skill has no local tag and a preview", func(t *testing.T) {
		row := rowByName(t, section, "foo")
		if strings.Contains(row.Detail, "local") || !strings.Contains(row.Preview, "Does foo things") {
			t.Fatalf("row = %+v, want no local tag and the description as preview", row)
		}
	})
	t.Run("skill outside the manifest source is local", func(t *testing.T) {
		row := rowByName(t, section, "mine")
		if !strings.Contains(row.Detail, "local") || !strings.Contains(row.Preview, "My own skill") {
			t.Fatalf("row = %+v, want local tag and preview", row)
		}
	})
	t.Run("symlink to a directory is listed", func(t *testing.T) {
		if row := rowByName(t, section, "linked"); !strings.Contains(row.Preview, "Via symlink") {
			t.Fatalf("row = %+v, want the symlink target description", row)
		}
	})
	t.Run("missing description or SKILL.md falls back", func(t *testing.T) {
		for _, name := range []string{"nosummary", "nofile"} {
			if row := rowByName(t, section, name); !strings.Contains(row.Preview, manageNoSummaryText) {
				t.Errorf("%s preview = %q, want %q", name, row.Preview, manageNoSummaryText)
			}
		}
	})
	t.Run("plain files and plugin skills are not listed", func(t *testing.T) {
		for _, r := range section.Rows {
			if r.Name == "stray.txt" || r.Name == "pluginskill" {
				t.Errorf("row %q must not be listed", r.Name)
			}
		}
		if len(section.Rows) != 5 {
			t.Errorf("rows = %d, want 5 (foo, mine, linked, nosummary, nofile)", len(section.Rows))
		}
	})
}

func TestManage_MissingComponentsAreUnknown(t *testing.T) {
	for title, drop := range map[string]func(manifest.Component) bool{
		"mcp":    func(c manifest.Component) bool { return c.Name == componentMCP },
		"plugin": func(c manifest.Component) bool { return c.Kind == kindPlugins },
	} {
		t.Run(title, func(t *testing.T) {
			m := manageManifest()
			m.Components = slices.DeleteFunc(m.Components, drop)
			if section := sectionByTitle(t, runManage(t, manageEnv(t), m), title); section.Unknown == "" {
				t.Fatalf("section = %+v, want Unknown", section)
			}
		})
	}
}

func TestManage_AbsentOrEmptySourcesAreEmpty(t *testing.T) {
	tests := map[string]struct {
		title string
		setup func(env check.Env)
	}{
		"settings file absent":    {"plugin", func(check.Env) {}},
		"empty enabledPlugins":    {"plugin", func(env check.Env) { writeFile(t, env.Home, manageSettingsRel, `{"enabledPlugins": {}}`) }},
		"skills directory absent": {"skill", func(check.Env) {}},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env := manageEnv(t)
			tc.setup(env)
			section := sectionByTitle(t, runManage(t, env, manageManifest()), tc.title)
			if len(section.Rows) != 0 || section.Empty == "" || section.Unknown != "" {
				t.Fatalf("section = %+v, want no rows, an Empty message and no Unknown", section)
			}
		})
	}
}

func TestManageMCPs_AbsentHarnessSourceMakesEveryServerExternal(t *testing.T) {
	env := manageEnv(t)
	writeFile(t, env.Home, manageMCPTarget, `{"mcpServers": {"zeta": {"command": "node"}, "alpha": {"command": "node"}}}`)
	section := sectionByTitle(t, runManage(t, env, manageManifest()), "mcp")
	if len(section.Rows) != 2 || section.Rows[0].Name != "alpha" || section.Rows[1].Name != "zeta" {
		t.Fatalf("rows = %+v, want alpha then zeta", section.Rows)
	}
	for _, r := range section.Rows {
		if !strings.Contains(r.Detail, "external") {
			t.Errorf("row = %+v, want external", r)
		}
	}
}

func TestManagePlugins_RowsAreSorted(t *testing.T) {
	env := manageEnv(t)
	writeFile(t, env.Home, manageSettingsRel, `{"enabledPlugins": {"z@m": false, "a@m": false}}`)
	section := sectionByTitle(t, runManage(t, env, manageManifest()), "plugin")
	if len(section.Rows) != 2 || section.Rows[0].Name != "a@m" {
		t.Fatalf("rows = %+v, want a@m first", section.Rows)
	}
}

func TestManageSkills_SortedDanglingSkippedAndUntaggedWithoutSource(t *testing.T) {
	env := manageEnv(t)
	writeFile(t, env.Home, ".claude/skills/zed/SKILL.md", "---\nname: zed\n---\n")
	writeFile(t, env.Home, ".claude/skills/alpha/SKILL.md", "---\nname: alpha\n---\n")
	if err := os.Symlink(filepath.Join(env.Home, "missing"), filepath.Join(env.Home, ".claude/skills/dangling")); err != nil {
		t.Fatal(err)
	}

	section := sectionByTitle(t, runManage(t, env, manageManifest()), "skill")

	if len(section.Rows) != 2 || section.Rows[0].Name != "alpha" || section.Rows[1].Name != "zed" {
		t.Fatalf("rows = %+v, want alpha then zed (dangling skipped)", section.Rows)
	}
	for _, r := range section.Rows {
		if r.Detail != "" {
			t.Errorf("row = %+v, want no local tag while the harness skill set is unreadable", r)
		}
	}
}

func TestManageSkills_MissingComponentIsUnknownNamingIt(t *testing.T) {
	m := manageManifest()
	m.Components = slices.DeleteFunc(m.Components, func(c manifest.Component) bool { return c.Name == componentSkills })
	section := sectionByTitle(t, runManage(t, manageEnv(t), m), "skill")
	if !strings.Contains(section.Unknown, componentSkills) {
		t.Fatalf("Unknown = %q, want it to name %q", section.Unknown, componentSkills)
	}
}
