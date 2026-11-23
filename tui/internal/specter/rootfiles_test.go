package specter_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

// The bash renderer reads targets/rendered-files.txt; the Go side keeps RootFiles. They must name the same files.
func TestRootFiles_MatchRenderedFilesList(t *testing.T) {
	const guardHook = ".claude/hooks/infra-agent-bash-guard.py"
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "targets", "rendered-files.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, f := range specter.RootFiles {
		if f.Label == "" {
			t.Errorf("RootFile %+v has no label", f)
		}
		if _, err := os.Stat(filepath.Join("..", "..", "..", f.RepoPath)); err != nil {
			t.Errorf("RootFile %q: repo source missing: %v", f.Label, err)
		}
		if f.HomePath != guardHook {
			want = append(want, f.HomePath)
		}
	}
	var got []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			got = append(got, line)
		}
	}
	slices.Sort(want)
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("rendered-files.txt = %v, want RootFiles minus the guard hook = %v", got, want)
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
			root := filepath.Join(env.Home, "dev", "specter")
			writeFile(t, filepath.Join(env.Home, ".claude", "CLAUDE.md"), tt.content(env.Home, root))

			ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.Saved.SpecterRoot, env.Home)), "global instructions")

			if !ref.Found || !ref.References {
				t.Errorf("ref = %+v, want found and referencing", ref)
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
			writeFile(t, filepath.Join(env.Home, ".claude", "CLAUDE.md"), tt.content)

			ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.Saved.SpecterRoot, env.Home)), "global instructions")

			if ref.References != tt.want {
				t.Errorf("References = %v, want %v", ref.References, tt.want)
			}
		})
	}
}

func TestLoadRootRefs_FileWithoutRootDoesNotReference(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	writeFile(t, filepath.Join(env.Home, ".claude", "CLAUDE.md"), "mentions ~/dev/other only")

	ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.Saved.SpecterRoot, env.Home)), "global instructions")

	if !ref.Found || ref.References {
		t.Errorf("ref = %+v, want found and not referencing", ref)
	}
}

func TestLoadRootRefs_RepoCopyWithPlaceholderIsIgnored(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.Repo = t.TempDir()
	writeFile(t, filepath.Join(env.Repo, "claude", "CLAUDE.md"), "root {{SPECTER_ROOT}}")
	writeFile(t, filepath.Join(env.Home, ".claude", "CLAUDE.md"), "root ~/dev/specter")

	ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.Saved.SpecterRoot, env.Home)), "global instructions")

	if !ref.Found || !ref.References {
		t.Errorf("ref = %+v, want the deployed copy read and referencing", ref)
	}
}

func TestLoadRootRefs_RepoCopyAloneDoesNotCountAsFound(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.Repo = t.TempDir()
	writeFile(t, filepath.Join(env.Repo, "claude", "CLAUDE.md"), "root ~/dev/specter")

	ref := findRef(t, specter.LoadRootRefs(env, specter.ResolveRoot(env.Saved.SpecterRoot, env.Home)), "global instructions")

	if ref.Found || ref.References {
		t.Errorf("ref = %+v, want not found: only the deployed copy is read", ref)
	}
}

func TestLoadRootRefs_MissingCopyIsNotFoundAndEveryEntryStillListed(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")

	refs := specter.LoadRootRefs(env, specter.ResolveRoot(env.Saved.SpecterRoot, env.Home))

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
