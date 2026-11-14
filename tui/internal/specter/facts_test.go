package specter_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func listCommands(id string) (active, archived, specs string) {
	return "openspec list --store " + id + " --json",
		"openspec list --store " + id + " --json --archived",
		"openspec list --store " + id + " --specs --json"
}

func storeListJSON(root string, ids ...string) check.ExecResult {
	out := `{"stores":[`
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += `{"id":"` + id + `","root":"` + filepath.Join(root, id) + `"}`
	}
	return check.ExecResult{Stdout: []byte(out + `],"status":[]}`)}
}

func addStoreResults(results map[string]check.ExecResult, id string, active, archived, specs int) {
	a, r, s := listCommands(id)
	change := func(n int, archivedFlag string) check.ExecResult {
		out := `{"changes":[`
		for i := 0; i < n; i++ {
			if i > 0 {
				out += ","
			}
			out += `{"name":"c` + string(rune('0'+i)) + `","completedTasks":1,"totalTasks":2,"lastModified":"2026-11-12T05:25:06.323Z","status":"in-progress"` + archivedFlag + `}`
		}
		return check.ExecResult{Stdout: []byte(out + `],"root":{"path":"/x","source":"store","store_id":"` + id + `"}}`)}
	}
	results[a] = change(active, "")
	results[r] = change(archived, `,"archived":true`)
	specList := `{"specs":[`
	for i := 0; i < specs; i++ {
		if i > 0 {
			specList += ","
		}
		specList += `{"id":"s` + string(rune('0'+i)) + `","requirementCount":3}`
	}
	results[s] = check.ExecResult{Stdout: []byte(specList + `],"root":{"path":"/x","source":"store","store_id":"` + id + `"}}`)}
}

func TestLoad_CountsActiveArchivedAndSpecsPerStore(t *testing.T) {
	root := t.TempDir()
	results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, "a")}
	addStoreResults(results, "a", 2, 1, 0)
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}

	facts := specter.Load(context.Background(), envWith(t, exec, root))

	got := facts.Stores["a"]
	if got.Err != nil || got.Active != 2 || got.Archived != 1 || got.Specs != 0 {
		t.Errorf("stats = %+v, want 2 active, 1 archived, 0 specs, no Err", got)
	}
	if calls := exec.callsTo("openspec list"); len(calls) != 3 {
		t.Errorf("list calls = %v, want 3 per registered store", calls)
	}
}

func TestLoad_FailingStoreSetsOnlyItsErr(t *testing.T) {
	root := t.TempDir()
	results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, "a", "b")}
	addStoreResults(results, "a", 2, 1, 4)
	addStoreResults(results, "b", 1, 0, 0)
	_, archivedB, _ := listCommands("b")
	results[archivedB] = check.ExecResult{ExitCode: 1, Stderr: []byte("boom")}
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}

	facts := specter.Load(context.Background(), envWith(t, exec, root))

	if facts.Stores["b"].Err == nil {
		t.Error("store b Err = nil, want the failed list call")
	}
	if a := facts.Stores["a"]; a.Err != nil || a.Active != 2 || a.Archived != 1 || a.Specs != 4 {
		t.Errorf("store a = %+v, want intact counts", a)
	}
}

func TestLoad_OpenspecMissingMakesNoCallAndKeepsRepos(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "repos.yaml"), reposYAML)
	exec := &fakeExec{}

	facts := specter.Load(context.Background(), envWith(t, exec, root))

	if got := exec.callsTo("openspec"); len(got) != 0 {
		t.Errorf("openspec calls = %v, want none", got)
	}
	if facts.Registration.RegisteredErr == nil || len(facts.Stores) != 0 {
		t.Errorf("RegisteredErr=%v Stores=%v, want unknown registry and no stats", facts.Registration.RegisteredErr, facts.Stores)
	}
	if facts.Repos.Err != nil || len(facts.Repos.Repos) != 3 {
		t.Errorf("Repos = %+v, want parsed repos.yaml", facts.Repos)
	}
}

func TestLoad_MissingOrInvalidReposYAMLSetsOnlyReposErr(t *testing.T) {
	for name, content := range map[string]string{"missing": "", "invalid": "repos: [unclosed\n  - : :"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			if content != "" {
				writeFile(t, filepath.Join(root, "repos.yaml"), content)
			}
			results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, "a")}
			addStoreResults(results, "a", 1, 0, 0)
			exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}

			facts := specter.Load(context.Background(), envWith(t, exec, root))

			if facts.Repos.Err == nil {
				t.Error("Repos.Err = nil, want the read or parse error")
			}
			if facts.Stores["a"].Err != nil || facts.Stores["a"].Active != 1 {
				t.Errorf("store a = %+v, want counts unaffected", facts.Stores["a"])
			}
		})
	}
}

func TestLoad_ReadsReferencesFromEachStoreConfig(t *testing.T) {
	root := t.TempDir()
	writeStore(t, root, "a", "id: a\n")
	writeFile(t, filepath.Join(root, "a", "openspec", "config.yaml"), "schema: spec-driven\nreferences:\n  - b\n  - c\n")
	results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, "a")}
	addStoreResults(results, "a", 0, 0, 0)
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}

	facts := specter.Load(context.Background(), envWith(t, exec, root))

	got := facts.References["a"]
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Errorf("References[a] = %v, want [b c]", got)
	}
}

func TestLoad_NeverRunsGit(t *testing.T) {
	root := t.TempDir()
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: map[string]check.ExecResult{storeListCommand: storeListJSON(root)}}
	env := envWith(t, exec, root)
	env.Cwd = t.TempDir()

	specter.Load(context.Background(), env)

	if got := exec.callsTo("git"); len(got) != 0 {
		t.Errorf("git calls = %v, want none", got)
	}
}

func TestLoad_ReposYAMLThroughSymlink(t *testing.T) {
	tests := []struct {
		name    string
		target  func(t *testing.T, dir string) string
		wantErr bool
	}{
		{"small regular file is read", func(t *testing.T, dir string) string {
			p := filepath.Join(dir, "real.yaml")
			writeFile(t, p, reposYAML)
			return p
		}, false},
		{"oversized file is rejected", func(t *testing.T, dir string) string {
			p := filepath.Join(dir, "big.yaml")
			writeFile(t, p, strings.Repeat("#", (1<<20)+1))
			return p
		}, true},
		{"device is rejected", func(*testing.T, string) string { return "/dev/zero" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Symlink(tt.target(t, t.TempDir()), filepath.Join(root, "repos.yaml")); err != nil {
				t.Fatal(err)
			}

			facts := specter.Load(context.Background(), envWith(t, &fakeExec{}, root))

			if (facts.Repos.Err != nil) != tt.wantErr {
				t.Errorf("Repos.Err = %v, wantErr %v", facts.Repos.Err, tt.wantErr)
			}
		})
	}
}

func TestLoad_RecordsChangeNamesAndLastModified(t *testing.T) {
	root := t.TempDir()
	results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, "a")}
	addStoreResults(results, "a", 2, 0, 0)
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}

	facts := specter.Load(context.Background(), envWith(t, exec, root))

	want := []specter.ChangeInfo{{Name: "c0", LastModified: "2026-11-12T05:25:06.323Z"}, {Name: "c1", LastModified: "2026-11-12T05:25:06.323Z"}}
	if got := facts.Stores["a"].Changes; !reflect.DeepEqual(got, want) {
		t.Errorf("Changes = %+v, want %+v", got, want)
	}
}

func TestParseLastModified(t *testing.T) {
	got, ok := specter.ParseLastModified("2026-11-12T05:25:06.323Z")
	if !ok || !got.Equal(time.Date(2026, 11, 12, 5, 25, 6, 323_000_000, time.UTC)) {
		t.Errorf("ParseLastModified = %v, %v", got, ok)
	}
	if _, ok := specter.ParseLastModified("yesterday"); ok {
		t.Error("unparseable value reported ok")
	}
}

func TestLoad_RelativeRootSetsReposErr(t *testing.T) {
	facts := specter.Load(context.Background(), envWith(t, &fakeExec{}, "relative/specter"))

	if facts.Repos.Err == nil {
		t.Error("Repos.Err = nil, want the relative-root error")
	}
}

func TestLoad_CancelledContextRunsNoListCalls(t *testing.T) {
	root := t.TempDir()
	results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, "a")}
	addStoreResults(results, "a", 1, 1, 1)
	exec := &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	facts := specter.Load(ctx, envWith(t, exec, root))

	if calls := exec.callsTo("openspec list"); len(calls) != 0 {
		t.Errorf("list calls = %v, want none after cancel", calls)
	}
	if facts.Stores["a"].Err == nil {
		t.Error("store a Err = nil, want the cancellation")
	}
}

func TestLoad_RecordsHomeAndPathExistence(t *testing.T) {
	root := t.TempDir()
	existing := t.TempDir()
	missing := filepath.Join(t.TempDir(), "gone")
	writeFile(t, filepath.Join(root, "repos.yaml"), "repos:\n  - path: "+existing+"\n    store: a\n  - path: "+missing+"\n    store: a\n")
	env := envWith(t, &fakeExec{}, root)

	facts := specter.Load(context.Background(), env)

	if facts.Home != env.Home {
		t.Errorf("Home = %q, want %q", facts.Home, env.Home)
	}
	if exists, ok := facts.PathExists[existing]; !ok || !exists {
		t.Errorf("PathExists[%s] = %v, %v, want true", existing, exists, ok)
	}
	if exists, ok := facts.PathExists[missing]; !ok || exists {
		t.Errorf("PathExists[%s] = %v, %v, want recorded false", missing, exists, ok)
	}
}

// concurrencyExec tracks how many Run calls overlap.
type concurrencyExec struct {
	*fakeExec
	running, peak atomic.Int32
}

func (c *concurrencyExec) Run(ctx context.Context, name string, args ...string) (check.ExecResult, error) {
	now := c.running.Add(1)
	defer c.running.Add(-1)
	for {
		peak := c.peak.Load()
		if now <= peak || c.peak.CompareAndSwap(peak, now) {
			break
		}
	}
	time.Sleep(10 * time.Millisecond)
	return c.fakeExec.Run(ctx, name, args...)
}

func TestLoad_BoundsConcurrentOpenspecCalls(t *testing.T) {
	root := t.TempDir()
	ids := []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	results := map[string]check.ExecResult{storeListCommand: storeListJSON(root, ids...)}
	for _, id := range ids {
		addStoreResults(results, id, 1, 1, 1)
	}
	exec := &concurrencyExec{fakeExec: &fakeExec{onPath: map[string]bool{"openspec": true}, results: results}}

	specter.Load(context.Background(), envWith(t, exec, root))

	if peak := exec.peak.Load(); peak > specter.MaxConcurrentOpenspec {
		t.Errorf("peak concurrent openspec calls = %d, want at most %d", peak, specter.MaxConcurrentOpenspec)
	}
}
