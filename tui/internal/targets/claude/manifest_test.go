package claude

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/testutil"
)

const realManifest = "../../../../targets/claude.toml"

func specKinds() []manifest.Kind {
	var kinds []manifest.Kind
	for k := range Spec().Evaluators {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	return kinds
}

func loadRealManifest(t *testing.T) manifest.Manifest {
	t.Helper()
	m, err := manifest.Load(realManifest, specKinds())
	if err != nil {
		t.Fatalf("manifest.Load(%s): %v", realManifest, err)
	}
	return m
}

func TestSpecChecks_RealManifestYieldsEveryCheck(t *testing.T) {
	m := loadRealManifest(t)
	got, err := Spec().Checks(m)
	if err != nil {
		t.Fatalf("Checks: %v", err)
	}
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	want := []string{checkJSON, checkHookScripts, checkHookArgs, checkMCPEnv, checkMCPCommands, checkFrontmatter, check.DependenciesID}
	slices.Sort(ids)
	slices.Sort(want)
	if !slices.Equal(ids, want) {
		t.Fatalf("check IDs = %v, want %v", ids, want)
	}
}

func TestSpecChecks_RealManifestWithoutSettingsNamesIt(t *testing.T) {
	m := loadRealManifest(t)
	m.Components = slices.DeleteFunc(m.Components, func(c manifest.Component) bool { return c.Name == componentSettings })
	_, err := Spec().Checks(m)
	if err == nil || !strings.Contains(err.Error(), componentSettings) {
		t.Fatalf("Checks error = %v, want it to name %q", err, componentSettings)
	}
}

// allToolsExec resolves every tool and answers every version probe, so no real process runs.
type allToolsExec struct{}

func (allToolsExec) LookPath(name string) (string, error) { return "/usr/bin/" + name, nil }
func (allToolsExec) Output(context.Context, string, ...string) ([]byte, error) {
	return []byte("1.2.3\n"), nil
}
func (allToolsExec) Run(context.Context, string, ...string) (check.ExecResult, error) {
	return check.ExecResult{Stdout: []byte("1.2.3\n")}, nil
}

func homeAndRepoHash(t *testing.T, env check.Env) string {
	t.Helper()
	return testutil.TreeHash(t, env.Home) + testutil.TreeHash(t, env.Repo)
}

func TestSpecChecks_RunningEveryCheckMutatesNothing(t *testing.T) {
	m := loadRealManifest(t)
	checks, err := Spec().Checks(m)
	if err != nil {
		t.Fatalf("Checks: %v", err)
	}
	env := newEnv(t)
	env.Exec = allToolsExec{}
	env.LookupEnv = func(name string) (string, bool) { return "set", name == "FOO" }

	writeFile(t, env.Home, ".claude/settings.json", `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"bash ~/.claude/hooks/stop.sh --opt ~/.claude/sounds/a.wav"}]}]}}`)
	writeFile(t, env.Home, ".claude/hooks/stop.sh", "#!/bin/sh\n")
	writeFile(t, env.Home, ".claude.json", `{"mcpServers":{"s":{"command":"node"}}}`)
	writeFile(t, env.Home, ".claude/plugins/claude-hud/config.json", `{}`)
	writeFile(t, env.Home, ".claude/agents/a.md", "---\nname: a\ndescription: d\n---\nbody\n")
	writeFile(t, env.Home, ".claude/skills/s/SKILL.md", "---\nname: s\ndescription: d\n---\nbody\n")
	writeFile(t, env.Repo, "claude/claude-mcp.json", `{"mcpServers":{"s":{"command":"node","args":["x"],"env":{"K":"${FOO}","M":"${MISSING}"}}}}`)

	before := homeAndRepoHash(t, env)
	results := check.Run(context.Background(), env, checks)
	if len(results) == 0 {
		t.Fatal("no results: the checks did no work")
	}
	if after := homeAndRepoHash(t, env); after != before {
		t.Fatal("running the Claude checks changed the HOME or repo tree")
	}
}

func TestSpec_KindOpsAndBackupDirs(t *testing.T) {
	spec := Spec()
	for kind := range spec.Evaluators {
		if _, special := spec.Special[kind]; !kind.IsData() && !special {
			t.Errorf("kind %q is neither data nor in Special", kind)
		}
	}
	if spec.Special[kindMCP].Hard == nil || spec.Special[kindPlugins].Hard == nil {
		t.Error("mcp and plugins must set Hard")
	}
	plugins := spec.Special[kindPlugins]
	if plugins.Remove != nil || !plugins.Late || !plugins.ExplicitUpdate {
		t.Errorf("plugins ops = Remove nil %v, Late %v, ExplicitUpdate %v; want true, true, true", plugins.Remove == nil, plugins.Late, plugins.ExplicitUpdate)
	}

	dirs := spec.BackupDirs("/home", "/repo")
	if len(dirs) != 2 || dirs[0].Path == dirs[1].Path {
		t.Fatalf("BackupDirs = %+v, want two distinct paths", dirs)
	}
	if dirs[0].Kind != backup.StoreHome || dirs[0].Base != "/home" || dirs[1].Kind != backup.StoreRepo || dirs[1].Base != "/repo" {
		t.Errorf("BackupDirs kinds/bases = %+v", dirs)
	}
}
