package specter_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const betaBlock = "  - remote: git@github.com:example/beta.git\n    store: beta\n"

func TestChangeRepoStore(t *testing.T) {
	tests := []struct {
		name  string
		old   string
		entry specter.RepoEntry
		want  string
	}{
		{
			"block entry changes only its store line",
			sampleReposYAML,
			specter.RepoEntry{Remote: "git@github.com:example/beta.git", Store: "beta"},
			strings.Replace(sampleReposYAML, betaBlock, "  - remote: git@github.com:example/beta.git\n    store: alpha\n", 1),
		},
		{
			"last entry before stores",
			sampleReposYAML,
			specter.RepoEntry{Remote: "git@github.com:example/gamma-app.git", Store: "gamma-app"},
			strings.Replace(sampleReposYAML, "    store: gamma-app\nstores:", "    store: alpha\nstores:", 1),
		},
		{
			"flow entry",
			"repos:\n  - { remote: git@x:a.git, store: a }\n  - remote: git@x:b.git\n    store: b\n",
			specter.RepoEntry{Remote: "git@x:a.git", Store: "a"},
			"repos:\n  - { remote: git@x:a.git, store: alpha }\n  - remote: git@x:b.git\n    store: b\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := specter.ChangeRepoStore([]byte(tt.old), tt.entry, "alpha")

			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestChangeRepoStore_UnknownEntryIsAnError(t *testing.T) {
	_, err := specter.ChangeRepoStore([]byte(sampleReposYAML), specter.RepoEntry{Remote: "git@x:absent.git"}, "alpha")

	if err == nil {
		t.Error("err = nil, want an error for a mapping that is not in the file")
	}
}

func TestRemoveRepoMapping(t *testing.T) {
	tests := []struct {
		name  string
		old   string
		entry specter.RepoEntry
		want  string
	}{
		{
			"block entry in the middle",
			sampleReposYAML,
			specter.RepoEntry{Remote: "git@github.com:example/beta.git", Store: "beta"},
			strings.Replace(sampleReposYAML, betaBlock, "", 1),
		},
		{
			"path entry",
			sampleReposYAML,
			specter.RepoEntry{Path: "~/dev/example-alpha", Store: "alpha"},
			strings.Replace(sampleReposYAML, "  - path: ~/dev/example-alpha\n    store: alpha\n", "", 1),
		},
		{
			"last entry stays clear of stores",
			sampleReposYAML,
			specter.RepoEntry{Remote: "git@github.com:example/gamma-app.git", Store: "gamma-app"},
			strings.Replace(sampleReposYAML, "  - remote: git@github.com:example/gamma-app.git\n    store: gamma-app\n", "", 1),
		},
		{
			"flow entry",
			"repos:\n  - { remote: git@x:a.git, store: a }\n  - remote: git@x:b.git\n    store: b\n",
			specter.RepoEntry{Remote: "git@x:a.git", Store: "a"},
			"repos:\n  - remote: git@x:b.git\n    store: b\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := specter.RemoveRepoMapping([]byte(tt.old), tt.entry)

			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestRemoveRepoMapping_UnknownEntryIsAnError(t *testing.T) {
	_, err := specter.RemoveRepoMapping([]byte(sampleReposYAML), specter.RepoEntry{Remote: "git@x:absent.git"})

	if err == nil {
		t.Error("err = nil, want an error for a mapping that is not in the file")
	}
}

func TestRemoveRepoMapping_OnlyEntryOfTheTemplateLeavesAParsableFile(t *testing.T) {
	template, err := os.ReadFile("testdata/repos.yaml")
	if err != nil {
		t.Fatal(err)
	}
	entry := specter.RepoEntry{Remote: "git@github.com:example/demo.git", Store: "demo"}
	added, err := specter.AddRepoMapping(template, entry)
	if err != nil {
		t.Fatal(err)
	}

	got, err := specter.RemoveRepoMapping(added, entry)

	if err != nil {
		t.Fatal(err)
	}
	if view := specter.ParseRepos(got); view.Err != nil || len(view.Repos) != 0 {
		t.Errorf("repos = %+v (err %v), want none and a valid file:\n%s", view.Repos, view.Err, got)
	}
}

// Several entries share one line in a flow list, so a line edit would hit a neighbour or the `repos:` key itself.
func TestRepoEdits_FlowListsOnOneLineAreUnsupportedNotSilentlyEdited(t *testing.T) {
	tests := map[string]string{
		"several items on one line": "repos: [{ remote: git@x:a.git, store: a }, { remote: git@x:b.git, store: b }]\n",
		"single item flow list":     "repos: [{ remote: git@x:a.git, store: a }]\n",
	}
	entry := specter.RepoEntry{Remote: "git@x:a.git", Store: "a"}
	for name, old := range tests {
		t.Run(name, func(t *testing.T) {
			changed, changeErr := specter.ChangeRepoStore([]byte(old), entry, "alpha")
			removed, removeErr := specter.RemoveRepoMapping([]byte(old), entry)

			for op, err := range map[string]error{"change": changeErr, "remove": removeErr} {
				if err == nil || !strings.Contains(err.Error(), "unsupported") {
					t.Errorf("%s err = %v, want an \"unsupported\" error", op, err)
				}
			}
			if changed != nil || removed != nil {
				t.Errorf("bytes = %q / %q, want none with an error", changed, removed)
			}
		})
	}
}

func TestRepoEdits_EndStateHoldsTheEditAndKeepsTheNeighbours(t *testing.T) {
	const old = "repos:\n  - { remote: git@x:a.git, store: a }\n  - remote: git@x:b.git\n    store: b\n"
	entry := specter.RepoEntry{Remote: "git@x:a.git"}

	changed, err := specter.ChangeRepoStore([]byte(old), entry, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	removed, err := specter.RemoveRepoMapping([]byte(old), entry)
	if err != nil {
		t.Fatal(err)
	}

	wantChanged := []specter.RepoEntry{{Remote: "git@x:a.git", Store: "alpha"}, {Remote: "git@x:b.git", Store: "b"}}
	if got := specter.ParseRepos(changed).Repos; !reflect.DeepEqual(got, wantChanged) {
		t.Errorf("after change repos = %+v, want %+v", got, wantChanged)
	}
	wantRemoved := []specter.RepoEntry{{Remote: "git@x:b.git", Store: "b"}}
	if got := specter.ParseRepos(removed).Repos; !reflect.DeepEqual(got, wantRemoved) {
		t.Errorf("after remove repos = %+v, want %+v", got, wantRemoved)
	}
}

const duplicateRemoteYAML = "repos:\n" +
	"  - remote: git@x:dup.git\n    store: first\n" +
	"  - remote: git@x:dup.git\n    store: second\n"

func TestChangeRepoStore_DuplicateRemoteChangesTheFirstEntry(t *testing.T) {
	got, err := specter.ChangeRepoStore([]byte(duplicateRemoteYAML), specter.RepoEntry{Remote: "git@x:dup.git"}, "other")

	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(duplicateRemoteYAML, "store: first", "store: other", 1)
	if string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestRemoveRepoMapping_DuplicateRemoteRemovesTheFirstEntry(t *testing.T) {
	got, err := specter.RemoveRepoMapping([]byte(duplicateRemoteYAML), specter.RepoEntry{Remote: "git@x:dup.git"})

	if err != nil {
		t.Fatal(err)
	}
	want := "repos:\n  - remote: git@x:dup.git\n    store: second\n"
	if string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestAddRepoMapping_TemplateReposYAMLAcceptsAnEntry(t *testing.T) {
	template, err := os.ReadFile("testdata/repos.yaml")
	if err != nil {
		t.Fatal(err)
	}
	entry := specter.RepoEntry{Remote: "git@github.com:example/demo.git", Store: "demo"}

	got, err := specter.AddRepoMapping(template, entry)

	if err != nil {
		t.Fatal(err)
	}
	view := specter.ParseRepos(got)
	if view.Err != nil || len(view.Repos) != 1 || view.Repos[0] != entry {
		t.Errorf("repos = %+v (err %v), want exactly %+v", view.Repos, view.Err, entry)
	}
	if !strings.Contains(string(got), "stores: {}") {
		t.Errorf("result =\n%s\nwant the stores key kept", got)
	}
}
