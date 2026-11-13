package specter_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestRootFiles_ListsTheSixD6Entries(t *testing.T) {
	want := []specter.RootFile{
		{Label: "global instructions", RepoPath: "claude/CLAUDE.md", HomePath: ".claude/CLAUDE.md"},
		{Label: "orchestrator agent", RepoPath: "agents/tech-orchestrator.md", HomePath: ".claude/agents/tech-orchestrator.md"},
		{Label: "guard hook", RepoPath: "claude/hooks/infra-agent-bash-guard.py", HomePath: ".claude/hooks/infra-agent-bash-guard.py"},
		{Label: "specter-ops agent", RepoPath: "agents/sub-agents/infrastructure/specter-ops.md", HomePath: ".claude/agents/sub-agents/infrastructure/specter-ops.md"},
		{Label: "/specter command", RepoPath: "claude/commands/specter.md", HomePath: ".claude/commands/specter.md"},
		{Label: "SDD rules", RepoPath: "shared/common-sdd.md", HomePath: ".claude/common-sdd.md"},
	}
	if !reflect.DeepEqual(specter.RootFiles, want) {
		t.Errorf("RootFiles = %+v, want %+v", specter.RootFiles, want)
	}
}

func TestLoadRootRefs_MatchesEveryEquivalentRootForm(t *testing.T) {
	tests := []struct {
		name    string
		content func(home, root string) string
	}{
		{"absolute", func(_, root string) string { return "use " + root + " here" }},
		{"tilde", func(_, _ string) string { return "use ~/dev/specter here" }},
		{"home variable", func(_, _ string) string { return "SPECTER=$HOME/dev/specter" }},
		{"braced home variable", func(_, _ string) string { return "SPECTER=${HOME}/dev/specter" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := envWith(t, &fakeExec{}, "")
			env.Repo = t.TempDir()
			root := filepath.Join(env.Home, "dev", "specter")
			writeFile(t, filepath.Join(env.Repo, "claude", "CLAUDE.md"), tt.content(env.Home, root))

			ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.LookupEnv, env.Home)), "global instructions")

			if !ref.Found || !ref.References || string(ref.Copy) != "repo" {
				t.Errorf("ref = %+v, want found, referencing, read from the repo copy", ref)
			}
		})
	}
}

func TestLoadRootRefs_RootMustEndOnABoundary(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"use ~/dev/specter-old", false},
		{"use ~/dev/specter2", false},
		{"use ~/dev/specter", true},
		{"use ~/dev/specter/", true},
		{"see ~/dev/specter.", true},
		{`root "~/dev/specter"`, true},
		{"~/dev/specter-old and then ~/dev/specter", true},
	}
	for _, tt := range tests {
		t.Run(tt.content, func(t *testing.T) {
			env := envWith(t, &fakeExec{}, "")
			env.Repo = t.TempDir()
			writeFile(t, filepath.Join(env.Repo, "claude", "CLAUDE.md"), tt.content)

			ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.LookupEnv, env.Home)), "global instructions")

			if ref.References != tt.want {
				t.Errorf("References = %v, want %v", ref.References, tt.want)
			}
		})
	}
}

func TestLoadRootRefs_FileWithoutRootDoesNotReference(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.Repo = t.TempDir()
	writeFile(t, filepath.Join(env.Repo, "claude", "CLAUDE.md"), "mentions ~/dev/other only")

	ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.LookupEnv, env.Home)), "global instructions")

	if !ref.Found || ref.References {
		t.Errorf("ref = %+v, want found and not referencing", ref)
	}
}

func TestLoadRootRefs_FallsBackToDeployedCopy(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.Repo = t.TempDir()
	writeFile(t, filepath.Join(env.Home, ".claude", "common-sdd.md"), "root ~/dev/specter")

	ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.LookupEnv, env.Home)), "SDD rules")

	if !ref.Found || !ref.References || string(ref.Copy) != "deployed" {
		t.Errorf("ref = %+v, want found, referencing, read from the deployed copy", ref)
	}
}

func TestLoadRootRefs_MissingCopyIsNotFoundAndEveryEntryStillListed(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.Repo = t.TempDir()

	refs := specter.LoadRootRefs(env, specter.ResolveRoot(env.LookupEnv, env.Home))

	if len(refs) != len(specter.RootFiles) {
		t.Fatalf("got %d refs, want %d", len(refs), len(specter.RootFiles))
	}
	for _, r := range refs {
		if r.Found || r.References {
			t.Errorf("ref %+v, want not found", r)
		}
	}
}

func findRef(t *testing.T, refs []specter.RootRef, label string) specter.RootRef {
	t.Helper()
	for _, r := range refs {
		if r.Label == label {
			return r
		}
	}
	t.Fatalf("no ref labelled %q in %+v", label, refs)
	return specter.RootRef{}
}
