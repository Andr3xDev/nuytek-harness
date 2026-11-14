package specter_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func requireOneDrift(t *testing.T, results []check.Result) check.Result {
	t.Helper()
	if len(results) != 1 {
		t.Fatalf("Drift returned %d results, want 1: %+v", len(results), results)
	}
	r := results[0]
	if !strings.HasPrefix(r.CheckID, "drift.") || r.Status != check.StatusWarning || !strings.Contains(r.Suggestion, "/specter") {
		t.Errorf("drift result = %+v, want drift.* warning with a /specter suggestion", r)
	}
	return r
}

func TestDrift_CleanFacts_ReportsNothing(t *testing.T) {
	if got := specter.Drift(healthyFacts(t)); len(got) != 0 {
		t.Errorf("Drift = %+v, want none", got)
	}
}

func TestDrift_StoreRegisteredAtAnotherRoot_ReportsBothPaths(t *testing.T) {
	f := healthyFacts(t)
	f.Folders = []specter.StoreFolder{{Dir: "/specs/a", ID: "a"}}
	f.Registered = []specter.StoreInfo{{ID: "a", Root: "/old/a"}}

	got := requireOneDrift(t, specter.Drift(f))

	if !strings.Contains(got.Message, "/specs/a") || !strings.Contains(got.Message, "/old/a") {
		t.Errorf("message %q lacks both paths", got.Message)
	}
}

func TestDrift_FolderRegisteredAtItsOwnDir_NotDrift(t *testing.T) {
	f := healthyFacts(t)
	f.Folders = []specter.StoreFolder{{Dir: f.Registered[0].Root, ID: "a"}}

	if got := specter.Drift(f); len(got) != 0 {
		t.Errorf("Drift = %+v, want none", got)
	}
}

func TestDrift_TrailingSlashOnRootIsNotDrift(t *testing.T) {
	f := healthyFacts(t)
	f.Folders = []specter.StoreFolder{{Dir: "/specs/a", ID: "a"}}
	f.Registered = []specter.StoreInfo{{ID: "a", Root: "/specs/a/"}}

	if got := specter.Drift(f); len(got) != 0 {
		t.Errorf("Drift = %+v, want none", got)
	}
}

func TestDrift_RegistryUnknown_SkipsRootAndReferenceDrift(t *testing.T) {
	f := healthyFacts(t)
	f.Folders = []specter.StoreFolder{{Dir: "/specs/a", ID: "a"}}
	f.Registered = []specter.StoreInfo{{ID: "a", Root: "/old/a"}}
	f.References = map[string][]string{"a": {"ghost"}}
	f.RegisteredErr = errors.New("openspec store list: exit 1: boom")

	if got := specter.Drift(f); len(got) != 0 {
		t.Errorf("Drift = %+v, want none while the registry is unknown", got)
	}
}

func TestDrift_StalePathAbsentFromPathExists_IsNotChecked(t *testing.T) {
	f := healthyFacts(t)
	f.Repos.Repos = []specter.RepoEntry{{Path: "/never/probed", Store: "a"}}
	f.PathExists = map[string]bool{}

	if got := specter.Drift(f); len(got) != 0 {
		t.Errorf("Drift = %+v, want none for an unchecked path", got)
	}
}

func TestDrift_DanglingReference_NamesIDWithReferencesCommand(t *testing.T) {
	f := healthyFacts(t)
	f.References = map[string][]string{"a": {"ghost"}}

	got := requireOneDrift(t, specter.Drift(f))

	if !strings.Contains(got.Message, "ghost") || !strings.Contains(got.Suggestion, "/specter references") {
		t.Errorf("drift = %+v", got)
	}
}

func TestDrift_ReferenceToRegisteredStore_NotDrift(t *testing.T) {
	f := healthyFacts(t)
	f.Registered = append(f.Registered, specter.StoreInfo{ID: "b", Root: t.TempDir()})
	f.References = map[string][]string{"a": {"b"}}

	if got := specter.Drift(f); len(got) != 0 {
		t.Errorf("Drift = %+v, want none", got)
	}
}

func TestDrift_StalePath_ReportsOnlyMissingPaths(t *testing.T) {
	f := healthyFacts(t)
	existing, missing := t.TempDir(), t.TempDir()+"/gone"
	f.Repos.Repos = []specter.RepoEntry{
		{Path: existing, Store: "a"},
		{Path: missing, Store: "a"},
	}
	f.PathExists = map[string]bool{existing: true, missing: false}

	got := requireOneDrift(t, specter.Drift(f))

	if !strings.Contains(got.Message, missing) {
		t.Errorf("message %q lacks the stale path", got.Message)
	}
}

func TestDrift_UnmappedCurrentRepo_NotDrift(t *testing.T) {
	f := healthyFacts(t)
	f.CurrentRepo = specter.CurrentRepo{Found: true, Path: "/work/other", Remote: "git@github.com:x/other.git"}

	if got := specter.Drift(f); len(got) != 0 {
		t.Errorf("Drift = %+v, want none", got)
	}
}
