package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/testutil"
)

const (
	repoStoreDirMode = fs.FileMode(0o700)
	stubLogName      = "claude.log"
	stubDirName      = "stubbin"
	uninstallPrefix  = "plugin uninstall "
	marketplaceDrop  = "plugin marketplace remove "
	editedJSON       = `{"edited":true}`
)

var allNames = []string{action.AllNames}

// seedRepo writes a valid source for every component of the fixture manifest.
func (w *world) seedRepo() {
	w.putRepo("claude/agents/a.md", "new")
	w.putRepo("claude/agents/__pycache__/a.pyc", "bytecode")
	w.putRepo("claude/agents/sub/s.md", "sub")
	hook := w.putRepo("claude/hooks/run.sh", "#!/bin/sh\n")
	if err := os.Chmod(hook, 0o755); err != nil {
		w.t.Fatal(err)
	}
	w.putRepo("claude/claude-mcp.json", `{"ours":{"command":"x"}}`)
	w.putRepo("claude/claude-settings.json", `{"enabledPlugins":{},"extraKnownMarketplaces":{}}`)
}

func TestSmokeScenarios(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		run  func(t *testing.T, w *world)
	}{
		{"install over a custom file overwrites owned files only", func(t *testing.T, w *world) {
			w.putHome(".claude/agents/a.md", "old")
			w.putHome(".claude/agents/custom.md", "mine")

			w.mustApply(action.PlanInstall(ctx, w.env, w.target, allNames))

			want := map[string]string{"a.md": "new", "sub/s.md": "sub", "custom.md": "mine"}
			if got := files(t, filepath.Join(w.home, ".claude/agents")); !reflect.DeepEqual(got, want) {
				t.Errorf("deployed agents = %v, want %v", got, want)
			}
		}},
		{"install snapshot mirrors the path and holds the old file only", func(t *testing.T, w *world) {
			w.putHome(".claude/agents/a.md", "old")
			w.putHome(".claude/agents/custom.md", "mine")

			result := w.mustApply(action.PlanInstall(ctx, w.env, w.target, allNames))

			if got := w.stamps(w.homeStore()); !reflect.DeepEqual(got, []string{result.Stamp}) || result.Stamp == "" {
				t.Fatalf("stamp dirs = %v, result.Stamp = %q", got, result.Stamp)
			}
			wantSnapshot := map[string]string{".claude/agents/a.md": "old"}
			if got := files(t, filepath.Join(w.homeStore(), result.Stamp)); !reflect.DeepEqual(got, wantSnapshot) {
				t.Errorf("snapshot = %v, want %v", got, wantSnapshot)
			}
		}},
		{"remove keeps a foreign file and snapshots what it deletes", func(t *testing.T, w *world) {
			w.mustApply(action.PlanInstall(ctx, w.env, w.target, []string{"agents"}))
			w.putHome(".claude/agents/foreign.md", "theirs")

			result := w.mustApply(action.PlanRemove(ctx, w.env, w.target, []string{"agents"}))

			want := map[string]string{"foreign.md": "theirs"}
			if got := files(t, filepath.Join(w.home, ".claude/agents")); !reflect.DeepEqual(got, want) {
				t.Errorf("deployed agents = %v, want %v", got, want)
			}
			if exists(filepath.Join(w.home, ".claude/agents/sub")) {
				t.Error("emptied directory sub was not cleaned")
			}
			wantSnapshot := map[string]string{".claude/agents/a.md": "new", ".claude/agents/sub/s.md": "sub"}
			if got := files(t, filepath.Join(w.homeStore(), result.Stamp)); !reflect.DeepEqual(got, wantSnapshot) {
				t.Errorf("snapshot = %v, want %v", got, wantSnapshot)
			}
		}},
		{"pull excludes secrets and writes a private repo store", func(t *testing.T, w *world) {
			w.putHome(".claude/agents/a.md", "deployed")
			w.putHome(".claude/agents/deployed-only.md", "x")
			w.putHome(".claude/agents/.env", "TOKEN=1")
			w.putHome(".claude/agents/auth.json", "{}")
			w.putHome(".claude/agents/key.pem", "pem")
			w.putHome(".claude/agents/__pycache__/a.pyc", "deployed bytecode")
			// Excluded names that already exist in the repo prove the filter: without it the deployed copy would overwrite them.
			w.putRepo("claude/agents/.env", "repo env")
			w.putRepo("claude/agents/auth.json", "repo auth")
			w.putRepo("claude/agents/key.pem", "repo pem")

			result := w.mustApply(action.PlanPull(ctx, w.env, w.target, []string{"agents"}))

			want := map[string]string{
				"a.md": "deployed", "sub/s.md": "sub", "__pycache__/a.pyc": "bytecode",
				".env": "repo env", "auth.json": "repo auth", "key.pem": "repo pem",
			}
			if got := files(t, filepath.Join(w.repo, "claude/agents")); !reflect.DeepEqual(got, want) {
				t.Errorf("repo agents = %v, want %v", got, want)
			}
			info, err := os.Stat(filepath.Join(w.repoStore(), result.Stamp))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != repoStoreDirMode {
				t.Errorf("repo store stamp dir mode = %v, want %v", info.Mode().Perm(), repoStoreDirMode)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t)
			w.skipPlugins()
			w.seedRepo()
			tt.run(t, w)
		})
	}
}

// editedWorld has a deployed install with a snapshot of the pre-install state, then local edits and an added file.
func editedWorld(t *testing.T) *world {
	t.Helper()
	w := newWorld(t)
	w.skipPlugins()
	w.seedRepo()
	w.putHome(".claude/agents/a.md", "old")
	w.putHome(".claude.json", `{"theme":"dark","mcpServers":{"theirs":{"command":"t"}}}`)
	w.mustApply(action.PlanInstall(context.Background(), w.env, w.target, allNames))
	w.putHome(".claude/agents/a.md", "EDITED")
	w.putHome(".claude.json", editedJSON)
	w.putHome(".claude/agents/added.md", "added")
	return w
}

func TestRestoreBringsBackSnapshotFilesAndKeepsLaterOnes(t *testing.T) {
	w := editedWorld(t)
	snaps := w.snapshots()
	if len(snaps) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(snaps))
	}
	install := snaps[0]
	wantJSON := readFile(t, filepath.Join(install.Path, ".claude.json"))

	result := w.mustApply(action.PlanRestore(context.Background(), w.env, w.target, install))

	if got := readFile(t, filepath.Join(w.home, ".claude/agents/a.md")); got != "old" {
		t.Errorf("a.md = %q, want the snapshot content", got)
	}
	if got := readFile(t, filepath.Join(w.home, ".claude.json")); got != wantJSON {
		t.Errorf(".claude.json = %q, want whole snapshot file %q", got, wantJSON)
	}
	if got := readFile(t, filepath.Join(w.home, ".claude/agents/added.md")); got != "added" {
		t.Errorf("file added after the snapshot = %q, want it kept", got)
	}
	if result.Stamp == "" || result.Stamp == install.Stamp {
		t.Fatalf("safety snapshot stamp = %q, install stamp = %q", result.Stamp, install.Stamp)
	}
	wantSafety := map[string]string{".claude/agents/a.md": "EDITED", ".claude.json": editedJSON}
	if got := files(t, filepath.Join(w.homeStore(), result.Stamp)); !reflect.DeepEqual(got, wantSafety) {
		t.Errorf("safety snapshot = %v, want %v", got, wantSafety)
	}
}

func TestPlanningWritesNothing(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		plan func(w *world) (plan.Plan, error)
	}{
		{"install", func(w *world) (plan.Plan, error) { return action.PlanInstall(ctx, w.env, w.target, allNames) }},
		{"hard install", func(w *world) (plan.Plan, error) { return action.PlanHardInstall(ctx, w.env, w.target) }},
		{"pull", func(w *world) (plan.Plan, error) { return action.PlanPull(ctx, w.env, w.target, []string{"agents"}) }},
		{"restore", func(w *world) (plan.Plan, error) { return action.PlanRestore(ctx, w.env, w.target, w.snapshots()[0]) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := editedWorld(t)
			homeBefore, repoBefore := testutil.TreeHash(t, w.home), testutil.TreeHash(t, w.repo)

			p, err := tt.plan(w)
			if err != nil {
				t.Fatal(err)
			}

			if p.Empty() {
				t.Error("plan is empty, so the check proves nothing")
			}
			if testutil.TreeHash(t, w.home) != homeBefore {
				t.Error("planning changed HOME")
			}
			if testutil.TreeHash(t, w.repo) != repoBefore {
				t.Error("planning changed the repo")
			}
		})
	}
}

func TestSmokeHardInstall(t *testing.T) {
	tests := []struct {
		name       string
		exitCode   int
		wantWarned int
	}{
		{"claude stub succeeds", 0, 0},
		{"claude stub fails", 1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := newWorld(t)
			w.seedRepo()
			w.putHome(".claude/agents/a.md", "old")
			w.putHome(".claude/agents/custom.md", "mine")
			w.putHome(".claude/hooks/run.sh", "old hook")
			w.putHome(".claude/hooks/mine.sh", "my hook")
			w.putHome(".claude/projects/p1/session.jsonl", "session")
			w.putHome(".claude/.credentials.json", "secret")
			w.putHome(".claude/plugins/installed_plugins.json", `{"version":1,"plugins":{"mine@local":{}}}`)
			w.putHome(".claude/plugins/known_marketplaces.json", `{"localmk":{}}`)
			oldJSON := `{"projects":{"/x":{"a":1}},"theme":"dark","mcpServers":{"foreign":{"command":"f"}}}`
			w.putHome(".claude.json", oldJSON)
			log := installStub(t, tt.exitCode)

			projectsBefore := testutil.TreeHash(t, filepath.Join(w.home, ".claude/projects"))
			credentialsBefore := testutil.TreeHash(t, filepath.Join(w.home, ".claude/.credentials.json"))

			p, err := action.PlanHardInstall(context.Background(), w.env, w.target)
			if err != nil {
				t.Fatal(err)
			}
			result := w.run(p)

			if result.Failed != nil {
				t.Fatalf("run failed: %+v", result.Failed)
			}
			if len(result.Warned) != tt.wantWarned {
				t.Errorf("Warned = %v, want %d", result.Warned, tt.wantWarned)
			}
			wantAgents := map[string]string{"a.md": "new", "sub/s.md": "sub"}
			if got := files(t, filepath.Join(w.home, ".claude/agents")); !reflect.DeepEqual(got, wantAgents) {
				t.Errorf("agents = %v, want exactly the filtered source %v", got, wantAgents)
			}
			if got := files(t, filepath.Join(w.home, ".claude/hooks")); !reflect.DeepEqual(got, map[string]string{"run.sh": "#!/bin/sh\n"}) {
				t.Errorf("hooks = %v", got)
			}
			if testutil.TreeHash(t, filepath.Join(w.home, ".claude/projects")) != projectsBefore {
				t.Error("projects/ changed")
			}
			if testutil.TreeHash(t, filepath.Join(w.home, ".claude/.credentials.json")) != credentialsBefore {
				t.Error(".credentials.json changed")
			}

			if got := w.stamps(w.homeStore()); !reflect.DeepEqual(got, []string{result.Stamp}) || result.Stamp == "" {
				t.Fatalf("stamp dirs = %v, result.Stamp = %q; want exactly one", got, result.Stamp)
			}
			wantSnapshot := map[string]string{
				".claude/agents/a.md":      "old",
				".claude/agents/custom.md": "mine",
				".claude/hooks/run.sh":     "old hook",
				".claude/hooks/mine.sh":    "my hook",
				".claude.json":             oldJSON,
			}
			if got := files(t, filepath.Join(w.homeStore(), result.Stamp)); !reflect.DeepEqual(got, wantSnapshot) {
				t.Errorf("snapshot = %v, want %v", got, wantSnapshot)
			}

			var deployed map[string]json.RawMessage
			if err := json.Unmarshal([]byte(readFile(t, filepath.Join(w.home, ".claude.json"))), &deployed); err != nil {
				t.Fatal(err)
			}
			var servers map[string]json.RawMessage
			if err := json.Unmarshal(deployed["mcpServers"], &servers); err != nil {
				t.Fatal(err)
			}
			if got := slicesOfKeys(servers); !reflect.DeepEqual(got, []string{"ours"}) {
				t.Errorf("mcpServers keys = %v, want only ours", got)
			}
			var before map[string]json.RawMessage
			if err := json.Unmarshal([]byte(oldJSON), &before); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"projects", "theme"} {
				if !jsonEqual(t, deployed[key], before[key]) {
					t.Errorf("top-level key %q = %s, want %s", key, deployed[key], before[key])
				}
			}

			wantCalls := []string{uninstallPrefix + "mine@local", marketplaceDrop + "localmk"}
			if got := removalCalls(t, log); !reflect.DeepEqual(got, wantCalls) {
				t.Errorf("removal argv = %v, want %v", got, wantCalls)
			}
		})
	}
}

// installStub puts a recording claude script first on PATH and returns its log path.
func installStub(t *testing.T, exitCode int) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), stubDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(dir, stubLogName)
	script := fmt.Sprintf("#!/bin/sh\necho \"$@\" >> '%s'\nexit %d\n", log, exitCode)
	if err := os.WriteFile(filepath.Join(dir, "claude"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

// removalCalls returns the logged uninstall and marketplace remove argv lines in call order.
func removalCalls(t *testing.T, log string) []string {
	t.Helper()
	var calls []string
	for _, line := range strings.Split(readFile(t, log), "\n") {
		if strings.HasPrefix(line, uninstallPrefix) || strings.HasPrefix(line, marketplaceDrop) {
			calls = append(calls, line)
		}
	}
	return calls
}

func slicesOfKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for key := range maps.Keys(m) {
		keys = append(keys, key)
	}
	return keys
}

func jsonEqual(t *testing.T, a, b json.RawMessage) bool {
	t.Helper()
	var x, y any
	if err := json.Unmarshal(a, &x); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &y); err != nil {
		t.Fatal(err)
	}
	return reflect.DeepEqual(x, y)
}
