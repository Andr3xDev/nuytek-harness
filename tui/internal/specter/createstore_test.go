package specter_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

// storesEnv has openspec on PATH and the given stores both registered and present as folders under the root.
func storesEnv(t *testing.T, ids ...string) (env check.Env, root string, exec *fakeExec) {
	t.Helper()
	exec = &fakeExec{onPath: map[string]bool{"openspec": true}, results: map[string]check.ExecResult{}}
	env, root = homeEnv(t, exec)
	exec.results[storeListCommand] = storeListJSON(root, ids...)
	for _, id := range ids {
		writeStore(t, root, id, "id: "+id+"\n")
		writeFile(t, filepath.Join(root, id, "openspec", "config.yaml"), "schema: spec-driven\n")
	}
	return env, root, exec
}

func editsByPath(p plan.Plan) map[string]plan.Edit {
	edits := map[string]plan.Edit{}
	for _, s := range p.Steps {
		if e, ok := s.(plan.Edit); ok {
			edits[e.Path] = e
		}
	}
	return edits
}

func TestPlanCreateStore_NewStoreIsSetUpThenCrossReferencedWithReadme(t *testing.T) {
	env, root, _ := storesEnv(t, "a", "b")

	p, err := specter.PlanCreateStore(context.Background(), env, "c")

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 5 {
		t.Fatalf("steps = %+v, want setup Exec plus 4 Edits (new config, a, b, README)", p.Steps)
	}
	setup, ok := p.Steps[0].(plan.Exec)
	if want := "openspec store setup c --path " + filepath.Join(root, "c") + " --no-init-git"; !ok || strings.Join(setup.Argv, " ") != want {
		t.Errorf("step 0 = %+v, want Exec %q", p.Steps[0], want)
	}
	edits := editsByPath(p)

	got, err := edits[filepath.Join(root, "c", "openspec", "config.yaml")].Edit([]byte("schema: spec-driven\n"), true)
	if err != nil {
		t.Fatal(err)
	}
	if refs := referencesOf(t, got); !reflect.DeepEqual(refs, []string{"a", "b"}) {
		t.Errorf("new store references = %v, want [a b]", refs)
	}
	for _, id := range []string{"a", "b"} {
		edit, found := edits[filepath.Join(root, id, "openspec", "config.yaml")]
		if !found {
			t.Fatalf("no Edit for store %s config.yaml (edits: %v)", id, edits)
		}
		got, err := edit.Edit([]byte("schema: spec-driven\nreferences: [x]\n"), true)
		if err != nil {
			t.Fatal(err)
		}
		if refs := referencesOf(t, got); !slices.Contains(refs, "c") || !slices.Contains(refs, "x") {
			t.Errorf("store %s references = %v, want x kept and c added", id, refs)
		}
	}
	readme, found := edits[filepath.Join(root, "c", "README.md")]
	if !found {
		t.Fatalf("no Edit for the new README.md (edits: %v)", edits)
	}
	if got, err := readme.Edit(nil, false); err != nil || !strings.Contains(string(got), "c") {
		t.Errorf("README = %q (err %v), want text naming the id", got, err)
	}
	if readme.NewFileMode != 0o644 {
		t.Errorf("README NewFileMode = %o, want 644", readme.NewFileMode)
	}
}

func TestPlanCreateStore_IsSnapshotFree(t *testing.T) {
	env, _, _ := storesEnv(t, "a")

	p, err := specter.PlanCreateStore(context.Background(), env, "b")

	if err != nil {
		t.Fatal(err)
	}
	for _, s := range p.Steps {
		if _, ok := s.(plan.Snapshot); ok {
			t.Error("plan holds a Snapshot, want none")
		}
	}
	if p.Empty() {
		t.Error("plan is empty, want rows")
	}
}

func TestPlanCreateStore_RejectsInvalidOrUsedIdWithNothingPlanned(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantReason string
	}{
		{"empty", "", "id"},
		{"uppercase", "Bad", "id"},
		{"leading dash", "-x", "id"},
		{"path separator", "a/b", "id"},
		{"already used", "a", "already"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, _, _ := storesEnv(t, "a")

			p, err := specter.PlanCreateStore(context.Background(), env, tt.id)

			if err == nil {
				t.Fatal("err = nil, want a rejection reason")
			}
			if !strings.Contains(err.Error(), tt.wantReason) {
				t.Errorf("err = %q, want %q mentioned", err, tt.wantReason)
			}
			if len(p.Steps) != 0 {
				t.Errorf("steps = %+v, want none", p.Steps)
			}
		})
	}
}

func TestPlanCreateStore_IdUsedByOnlyOneSourceIsStillTaken(t *testing.T) {
	folderOnly, folderRoot, _ := storesEnv(t)
	writeStore(t, folderRoot, "a", "id: a\n")
	registeredOnly, registeredRoot, exec := storesEnv(t)
	exec.results[storeListCommand] = storeListJSON(registeredRoot, "a")
	if err := os.MkdirAll(registeredRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		env  check.Env
	}{
		{"folder without registration", folderOnly},
		{"registration without folder", registeredOnly},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := specter.PlanCreateStore(context.Background(), tt.env, "a")

			if err == nil || !strings.Contains(err.Error(), "already") {
				t.Errorf("err = %v, want the id reported as already used", err)
			}
			if len(p.Steps) != 0 {
				t.Errorf("steps = %+v, want none", p.Steps)
			}
		})
	}
}

func TestPlanCreateStore_BlockedReasons(t *testing.T) {
	missingRoot, _ := homeEnv(t, openspecExec(storeListJSON("")))
	noOpenspec, _, exec := storesEnv(t, "a")
	delete(exec.onPath, "openspec")
	outsideRoot := t.TempDir()
	outside := envWith(t, openspecExec(storeListJSON(outsideRoot)), outsideRoot)
	tests := []struct {
		name string
		env  check.Env
		want string
	}{
		{"openspec missing", noOpenspec, "openspec"},
		{"root missing", missingRoot, "does not exist"},
		{"root outside HOME", outside, "outside HOME"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := specter.PlanCreateStore(context.Background(), tt.env, "c")

			if reason := blockedReason(t, err); !strings.Contains(reason, tt.want) {
				t.Errorf("reason = %q, want %q mentioned", reason, tt.want)
			}
			if len(p.Steps) != 0 {
				t.Errorf("steps = %+v, want none", p.Steps)
			}
		})
	}
}

// The cross-reference edits only fail at apply time, after `openspec store setup` already ran, so planning must catch them.
func TestPlanCreateStore_BlocksBeforeSetupWhenTheStoreCannotBeCreatedOrLinked(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T, root string)
		want    []string
	}{
		{"other store without config.yaml", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "b", "openspec", "config.yaml")); err != nil {
				t.Fatal(err)
			}
		}, []string{"b", "config.yaml"}},
		{"other store config.yaml unreadable", func(t *testing.T, root string) {
			path := filepath.Join(root, "b", "openspec", "config.yaml")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
		}, []string{"b", "config.yaml"}},
		{"target directory already exists", func(t *testing.T, root string) {
			if err := os.MkdirAll(filepath.Join(root, "c"), 0o755); err != nil {
				t.Fatal(err)
			}
		}, []string{"c", "already exists"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, root, _ := storesEnv(t, "a", "b")
			tt.arrange(t, root)

			p, err := specter.PlanCreateStore(context.Background(), env, "c")

			reason := blockedReason(t, err)
			for _, fragment := range tt.want {
				if !strings.Contains(reason, fragment) {
					t.Errorf("reason = %q, want %q mentioned", reason, fragment)
				}
			}
			if len(p.Steps) != 0 {
				t.Errorf("steps = %+v, want none", p.Steps)
			}
		})
	}
}

func TestPlanAddMapping_EditsReposYAMLWithoutSnapshot(t *testing.T) {
	env, root, _ := storesEnv(t, "alpha")
	reposPath := filepath.Join(root, "repos.yaml")
	writeFile(t, reposPath, sampleReposYAML)
	entry := specter.RepoEntry{Remote: "git@github.com:example/new.git", Store: "alpha"}

	p, err := specter.PlanAddMapping(context.Background(), env, entry)

	if err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 1 {
		t.Fatalf("steps = %+v, want one Edit", p.Steps)
	}
	edit, ok := p.Steps[0].(plan.Edit)
	if !ok || edit.Path != reposPath {
		t.Fatalf("step = %+v, want Edit of %s", p.Steps[0], reposPath)
	}
	got, err := edit.Edit([]byte(sampleReposYAML), true)
	want, _ := specter.AddRepoMapping([]byte(sampleReposYAML), entry)
	if err != nil || string(got) != string(want) || !strings.Contains(string(got), entry.Remote) {
		t.Errorf("edit = %q (err %v), want the mapping added as AddRepoMapping does", got, err)
	}
	if rows := p.Steps[0].Changes(); len(rows) == 0 || rows[0].Path != reposPath {
		t.Errorf("rows = %+v, want repos.yaml shown as touched", rows)
	}
}

func TestPlanChangeAndRemoveMapping_EditReposYAMLWithoutSnapshot(t *testing.T) {
	env, root, _ := storesEnv(t, "alpha", "beta")
	reposPath := filepath.Join(root, "repos.yaml")
	writeFile(t, reposPath, sampleReposYAML)
	entry := specter.RepoEntry{Remote: "git@github.com:example/beta.git"}
	wantChanged, _ := specter.ChangeRepoStore([]byte(sampleReposYAML), entry, "alpha")
	wantRemoved, _ := specter.RemoveRepoMapping([]byte(sampleReposYAML), entry)

	changed, err := specter.PlanChangeMapping(context.Background(), env, entry, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := specter.PlanRemoveMapping(context.Background(), env, entry)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		p    plan.Plan
		want []byte
	}{{"change", changed, wantChanged}, {"remove", removed, wantRemoved}} {
		edit, ok := tc.p.Steps[0].(plan.Edit)
		if len(tc.p.Steps) != 1 || !ok || edit.Path != reposPath {
			t.Fatalf("%s steps = %+v, want one Edit of %s", tc.name, tc.p.Steps, reposPath)
		}
		if got, err := edit.Edit([]byte(sampleReposYAML), true); err != nil || string(got) != string(tc.want) {
			t.Errorf("%s edit = %q (err %v), want %q", tc.name, got, err, tc.want)
		}
		if rows := edit.Changes(); len(rows) == 0 || rows[0].Path != reposPath {
			t.Errorf("%s rows = %+v, want repos.yaml shown as touched", tc.name, rows)
		}
	}
}

func TestPlanChangeMapping_NoRegisteredStoreIsBlocked(t *testing.T) {
	env, root, _ := storesEnv(t)
	writeFile(t, filepath.Join(root, "repos.yaml"), sampleReposYAML)

	_, err := specter.PlanChangeMapping(context.Background(), env, specter.RepoEntry{Remote: "git@x:a.git"}, "alpha")

	if reason := blockedReason(t, err); !strings.Contains(reason, "no registered store") {
		t.Errorf("reason = %q, want %q", reason, "no registered store")
	}
}

// Removing a mapping needs no store, so a root with stale mappings and no registered store can still be cleaned up.
func TestPlanRemoveMapping_NoRegisteredStoreIsNotBlocked(t *testing.T) {
	env, root, _ := storesEnv(t)
	reposPath := filepath.Join(root, "repos.yaml")
	writeFile(t, reposPath, sampleReposYAML)

	p, err := specter.PlanRemoveMapping(context.Background(), env, specter.RepoEntry{Remote: "git@github.com:example/beta.git"})

	if err != nil {
		t.Fatal(err)
	}
	if edit, ok := p.Steps[0].(plan.Edit); len(p.Steps) != 1 || !ok || edit.Path != reposPath {
		t.Errorf("steps = %+v, want one Edit of %s", p.Steps, reposPath)
	}
}

// Normalization is the one of MatchRepo: credentials are dropped from remotes, `~/` and absolute paths compare equal.
func TestPlanChangeAndRemoveMapping_FindTheEntryTheWayMatchRepoDoes(t *testing.T) {
	const reposYAML = "repos:\n" +
		"  - remote: https://github.com/example/web.git\n    store: alpha\n" +
		"  - path: ~/work/proj\n    store: alpha\n"
	tests := []struct {
		name         string
		entry        func(home string) specter.RepoEntry
		wantGone     string
		wantRepoLeft string
	}{
		{"remote with credentials", func(string) specter.RepoEntry {
			return specter.RepoEntry{Remote: "https://token@github.com/example/web.git"}
		}, "https://github.com/example/web.git", "~/work/proj"},
		{"absolute path for a tilde entry", func(home string) specter.RepoEntry {
			return specter.RepoEntry{Path: filepath.Join(home, "work", "proj")}
		}, "~/work/proj", "https://github.com/example/web.git"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, root, _ := storesEnv(t, "alpha", "beta")
			writeFile(t, filepath.Join(root, "repos.yaml"), reposYAML)
			entry := tt.entry(env.Home)

			removed, err := specter.PlanRemoveMapping(context.Background(), env, entry)
			if err != nil {
				t.Fatal(err)
			}
			changed, err := specter.PlanChangeMapping(context.Background(), env, entry, "beta")
			if err != nil {
				t.Fatal(err)
			}

			gotRemoved, err := removed.Steps[0].(plan.Edit).Edit([]byte(reposYAML), true)
			if err != nil || strings.Contains(string(gotRemoved), tt.wantGone) || !strings.Contains(string(gotRemoved), tt.wantRepoLeft) {
				t.Errorf("removed = %q (err %v), want %q gone and %q kept", gotRemoved, err, tt.wantGone, tt.wantRepoLeft)
			}
			gotChanged, err := changed.Steps[0].(plan.Edit).Edit([]byte(reposYAML), true)
			if view := specter.ParseRepos(gotChanged); err != nil || view.Err != nil || storeOf(view, tt.wantGone) != "beta" {
				t.Errorf("changed = %q (err %v), want the %q entry on store beta", gotChanged, err, tt.wantGone)
			}
		})
	}
}

func storeOf(view specter.ReposView, remoteOrPath string) string {
	for _, e := range view.Repos {
		if e.Remote == remoteOrPath || e.Path == remoteOrPath {
			return e.Store
		}
	}
	return ""
}

// Failing at plan time keeps the user from approving a preview that cannot be applied.
func TestPlanChangeAndRemoveMapping_UnknownEntryIsBlockedAtPlanTime(t *testing.T) {
	env, root, _ := storesEnv(t, "alpha")
	writeFile(t, filepath.Join(root, "repos.yaml"), sampleReposYAML)
	entry := specter.RepoEntry{Remote: "git@github.com:example/absent.git"}

	changed, changeErr := specter.PlanChangeMapping(context.Background(), env, entry, "alpha")
	removed, removeErr := specter.PlanRemoveMapping(context.Background(), env, entry)

	for name, err := range map[string]error{"change": changeErr, "remove": removeErr} {
		if reason := blockedReason(t, err); !strings.Contains(reason, "not found") {
			t.Errorf("%s reason = %q, want %q", name, reason, "not found")
		}
	}
	if len(changed.Steps)+len(removed.Steps) != 0 {
		t.Errorf("steps = %+v / %+v, want none", changed.Steps, removed.Steps)
	}
}

func TestPlanChangeAndRemoveMapping_FlowListIsBlockedAtPlanTime(t *testing.T) {
	env, root, _ := storesEnv(t, "alpha")
	writeFile(t, filepath.Join(root, "repos.yaml"), "repos: [{remote: \"git@x:a.git\", store: alpha}]\n")
	entry := specter.RepoEntry{Remote: "git@x:a.git"}

	_, changeErr := specter.PlanChangeMapping(context.Background(), env, entry, "alpha")
	_, removeErr := specter.PlanRemoveMapping(context.Background(), env, entry)

	for name, err := range map[string]error{"change": changeErr, "remove": removeErr} {
		if reason := blockedReason(t, err); !strings.Contains(reason, "unsupported") {
			t.Errorf("%s reason = %q, want %q", name, reason, "unsupported")
		}
	}
}

func TestPlanAddMapping_NoRegisteredStoreIsBlocked(t *testing.T) {
	env, root, _ := storesEnv(t)
	writeFile(t, filepath.Join(root, "repos.yaml"), sampleReposYAML)

	p, err := specter.PlanAddMapping(context.Background(), env, specter.RepoEntry{Remote: "git@x:a.git", Store: "alpha"})

	if reason := blockedReason(t, err); !strings.Contains(reason, "no registered store") {
		t.Errorf("reason = %q, want %q", reason, "no registered store")
	}
	if len(p.Steps) != 0 {
		t.Errorf("steps = %+v, want none", p.Steps)
	}
}
