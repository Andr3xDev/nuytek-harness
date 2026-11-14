package specter_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func resultByID(t *testing.T, results []check.Result, id string) check.Result {
	t.Helper()
	for _, r := range results {
		if r.CheckID == id {
			return r
		}
	}
	t.Fatalf("no result with CheckID %q in %+v", id, results)
	return check.Result{}
}

func isFailed(s check.Status) bool { return s == check.StatusWarning || s == check.StatusError }

// healthyFacts has one registered store "a" mapped to a repo, the current repo mapped by remote, and agents files agreeing with the root.
func healthyFacts(t *testing.T) specter.Facts {
	t.Helper()
	root := specter.Root{Path: t.TempDir(), Source: specter.RootSourceDefault}
	storeRoot := t.TempDir()
	return specter.Facts{
		Registration: specter.Registration{
			Root:       root,
			RootExists: true,
			Openspec:   specter.ToolInfo{Found: true},
			Registered: []specter.StoreInfo{{ID: "a", Root: storeRoot}},
		},
		OpenspecVersion: "1.2.3",
		Repos: specter.ReposView{
			Repos:  []specter.RepoEntry{{Remote: originURL, Store: "a"}},
			Stores: map[string]specter.StoreMapping{"a": {}},
		},
		CurrentRepo: specter.CurrentRepo{Found: true, Path: "/work/demo", Remote: originURL},
		RootRefs:    []specter.RootRef{{Label: "global instructions", Copy: specter.CopyRepo, Found: true, References: true}},
	}
}

func TestChecks_AllHealthy_EveryCheckOK(t *testing.T) {
	results := specter.Checks(healthyFacts(t))

	for _, id := range []string{"specter.openspec", "specter.store.a", "specter.mapped.a", "specter.repo", "specter.agents"} {
		if got := resultByID(t, results, id); got.Status != check.StatusOK {
			t.Errorf("%s = %s (%s), want ok", id, got.Status, got.Message)
		}
	}
	if v := resultByID(t, results, "specter.openspec"); !strings.Contains(v.Message, "1.2.3") {
		t.Errorf("openspec message = %q, want the version", v.Message)
	}
}

func TestChecks_OpenspecMissing_FailsWithFixAndSkipsDependents(t *testing.T) {
	f := healthyFacts(t)
	f.Openspec = specter.ToolInfo{Err: os.ErrNotExist}
	f.Registered, f.RegisteredErr, f.OpenspecVersion = nil, errors.New("openspec not found on PATH"), ""

	results := specter.Checks(f)

	tool := resultByID(t, results, "specter.openspec")
	if !isFailed(tool.Status) || tool.Message != "openspec not found on PATH" || tool.Suggestion == "" {
		t.Errorf("openspec = %+v, want failed with message and suggestion", tool)
	}
	store := resultByID(t, results, "specter.store.a")
	if store.Status != check.StatusSkipped || store.Message == "" {
		t.Errorf("store.a = %+v, want skipped with a reason", store)
	}
}

func TestChecks_StoreNotRegistered_FailsWithSetupSuggestion(t *testing.T) {
	f := healthyFacts(t)
	f.Registered = nil

	got := resultByID(t, specter.Checks(f), "specter.store.a")

	if !isFailed(got.Status) || got.Message != "store a is not registered" {
		t.Errorf("store.a = %+v", got)
	}
	if !strings.Contains(got.Suggestion, "/specter setup a") {
		t.Errorf("suggestion = %q, want /specter setup a", got.Suggestion)
	}
}

func TestChecks_StoreNotRegisteredButFolderExists_SuggestsFastInit(t *testing.T) {
	f := healthyFacts(t)
	f.Registered = nil
	f.Folders = []specter.StoreFolder{{Dir: f.Root.Path + "/a", ID: "a"}}

	got := resultByID(t, specter.Checks(f), "specter.store.a")

	if !strings.Contains(got.Suggestion, "fast init") {
		t.Errorf("suggestion = %q, want fast init", got.Suggestion)
	}
}

func TestChecks_RegisteredRootMissingOnDisk_Fails(t *testing.T) {
	f := healthyFacts(t)
	f.Registered = []specter.StoreInfo{{ID: "a", Root: "/nonexistent/store-a"}}
	f.PathExists = map[string]bool{"/nonexistent/store-a": false}

	got := resultByID(t, specter.Checks(f), "specter.store.a")

	if !isFailed(got.Status) || !strings.Contains(got.Message, "registered at /nonexistent/store-a which does not exist") {
		t.Errorf("store.a = %+v", got)
	}
}

func TestChecks_StoreWithoutMappedRepo_SuggestsSpecterRegister(t *testing.T) {
	f := healthyFacts(t)
	f.Repos.Repos = nil

	got := resultByID(t, specter.Checks(f), "specter.mapped.a")

	if !isFailed(got.Status) || got.Message != "store a has no mapped repo" {
		t.Errorf("mapped.a = %+v", got)
	}
	if !strings.HasPrefix(got.Suggestion, "/specter register") {
		t.Errorf("suggestion = %q, want a /specter register command", got.Suggestion)
	}
}

func TestChecks_CurrentRepoUnmapped_FailsWithSuggestion(t *testing.T) {
	f := healthyFacts(t)
	f.CurrentRepo = specter.CurrentRepo{Found: true, Path: "/work/other", Remote: "git@github.com:x/other.git"}

	got := resultByID(t, specter.Checks(f), "specter.repo")

	if !isFailed(got.Status) || !strings.Contains(got.Message, "current repo /work/other is not mapped in repos.yaml") {
		t.Errorf("repo = %+v", got)
	}
	if !strings.Contains(got.Suggestion, "/specter register") {
		t.Errorf("suggestion = %q", got.Suggestion)
	}
}

func TestChecks_NotInGitRepo_SkippedNotFailed(t *testing.T) {
	f := healthyFacts(t)
	f.CurrentRepo = specter.CurrentRepo{}

	got := resultByID(t, specter.Checks(f), "specter.repo")

	if got.Status != check.StatusSkipped || got.Message != "not inside a git repository" {
		t.Errorf("repo = %+v, want skipped: not inside a git repository", got)
	}
}

func TestChecks_ReposYAMLUnreadable_YieldsOneUnknown(t *testing.T) {
	f := healthyFacts(t)
	f.Repos = specter.ReposView{Err: errors.New("repos.yaml: permission denied")}

	var unknown, failed int
	for _, r := range specter.Checks(f) {
		switch {
		case r.Status == check.StatusUnknown:
			unknown++
			if !strings.Contains(r.Message, "permission denied") {
				t.Errorf("unknown message = %q, want the reason", r.Message)
			}
		case isFailed(r.Status):
			failed++
		}
	}
	if unknown != 1 || failed != 0 {
		t.Errorf("unknown = %d, failed = %d, want 1 and 0", unknown, failed)
	}
}

func TestChecks_AgentsCopiesDifferFromRoot_WarnsWithoutSuggestion(t *testing.T) {
	f := healthyFacts(t)
	f.Root = specter.Root{Path: "/data/specs", Source: specter.RootSourceEnv, OutsideHome: true}
	f.RootRefs = []specter.RootRef{
		{Label: "global instructions", Copy: specter.CopyRepo, Found: true, References: false},
		{Label: "guard hook", Copy: specter.CopyDeployed, Found: true, References: false},
		{Label: "SDD rules", Found: false},
	}

	got := resultByID(t, specter.Checks(f), "specter.agents")

	if got.Status != check.StatusWarning || got.Suggestion != "" {
		t.Fatalf("agents = %+v, want warning without suggestion", got)
	}
	for _, want := range []string{
		"/data/specs", "~/dev/specter", "global instructions", "guard hook",
		"only the TUI supports NUYTEK_SPECTER_ROOT; the rest of the harness uses ~/dev/specter",
	} {
		if !strings.Contains(got.Message, want) {
			t.Errorf("message %q lacks %q", got.Message, want)
		}
	}
	if strings.Contains(got.Message, "SDD rules") {
		t.Errorf("message %q names a missing copy", got.Message)
	}
}

func TestChecks_AgentsDefaultRootNotReferenced_OmitsEnvVarNotice(t *testing.T) {
	f := healthyFacts(t)
	f.Root = specter.Root{Path: "/home/u/dev/specter", Source: specter.RootSourceDefault}
	f.RootRefs = []specter.RootRef{{Label: "guard hook", Copy: specter.CopyDeployed, Found: true, References: false}}

	got := resultByID(t, specter.Checks(f), "specter.agents")

	if got.Status != check.StatusWarning || got.Suggestion != "" {
		t.Fatalf("agents = %+v, want warning without suggestion", got)
	}
	for _, want := range []string{"do not reference", "/home/u/dev/specter", "guard hook"} {
		if !strings.Contains(got.Message, want) {
			t.Errorf("message %q lacks %q", got.Message, want)
		}
	}
	if strings.Contains(got.Message, "NUYTEK_SPECTER_ROOT") {
		t.Errorf("message %q names the env var for a default root", got.Message)
	}
}

func TestChecks_RegistryUnknown(t *testing.T) {
	f := healthyFacts(t)
	f.RegisteredErr = errors.New("openspec store list: exit 1: boom")

	results := specter.Checks(f)

	store := resultByID(t, results, "specter.store.a")
	if store.Status != check.StatusUnknown || !strings.Contains(store.Message, "registry unknown: ") || !strings.Contains(store.Message, "boom") {
		t.Errorf("store.a = %+v, want unknown naming the registry error", store)
	}
	for _, r := range results {
		if strings.HasPrefix(r.CheckID, "specter.mapped.") {
			t.Errorf("mapped check %+v emitted while the registry is unknown", r)
		}
	}
}

func TestChecks_AgentsAllCopiesMissing_Unknown(t *testing.T) {
	f := healthyFacts(t)
	f.RootRefs = []specter.RootRef{{Label: "global instructions"}, {Label: "guard hook"}}

	got := resultByID(t, specter.Checks(f), "specter.agents")

	if got.Status != check.StatusUnknown {
		t.Errorf("agents = %+v, want unknown", got)
	}
}

func TestSummary(t *testing.T) {
	ok := check.Result{CheckID: "a", Status: check.StatusOK}
	cases := []struct {
		name       string
		results    []check.Result
		wantFailed int
		wantStatus func(check.Status) bool
	}{
		{"all ok", []check.Result{ok, ok}, 0, func(s check.Status) bool { return s == check.StatusOK }},
		{"warning and error count as failed", []check.Result{ok, {Status: check.StatusWarning}, {Status: check.StatusError}}, 2, isFailed},
		{"ok with unknown is unknown", []check.Result{ok, {Status: check.StatusUnknown}}, 0, func(s check.Status) bool { return s == check.StatusUnknown }},
		{"only unknown and skipped is unknown", []check.Result{{Status: check.StatusSkipped}, {Status: check.StatusUnknown}}, 0, func(s check.Status) bool { return s == check.StatusUnknown }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := specter.Summary(tc.results)

			if got.FailedCount != tc.wantFailed || !tc.wantStatus(got.Status) || got.Text == "" {
				t.Errorf("Summary = %+v, want %d failed and a matching status", got, tc.wantFailed)
			}
			if tc.wantFailed > 0 && !strings.Contains(got.Text, "2") {
				t.Errorf("text %q lacks the failed count", got.Text)
			}
		})
	}
}
