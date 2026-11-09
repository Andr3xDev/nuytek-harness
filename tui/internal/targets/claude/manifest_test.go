package claude

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
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

func treeHash(t *testing.T, roots ...string) string {
	t.Helper()
	h := sha256.New()
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "%s|%v|%d\n", path, info.Mode(), info.Size())
			if info.Mode().IsRegular() {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				h.Write(data)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
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

	before := treeHash(t, env.Home, env.Repo)
	results := check.Run(context.Background(), env, checks)
	if len(results) == 0 {
		t.Fatal("no results: the checks did no work")
	}
	if after := treeHash(t, env.Home, env.Repo); after != before {
		t.Fatal("running the Claude checks changed the HOME or repo tree")
	}
}
