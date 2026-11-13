package specter_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

// Same shape as a real repos.yaml with fictional values: header comments, block repos, flow-style stores.
const realReposYAML = `# Repo -> OpenSpec store map (` + "`repos:`" + `), read by the orchestrator. Match by git remote; ` + "`path`" + ` entries match the working directory prefix (longest wins).
# ` + "`stores:`" + ` maps each store to the Engram projects its memories live in (one Engram project per repo/folder).
repos:
  - remote: git@github.com:example/alpha.git
    store: alpha
  - path: ~/dev/example-alpha
    store: alpha
  - remote: git@github.com:example/beta-plugin.git
    store: beta
  - remote: git@github.com:example/beta.git
    store: beta
  - remote: git@github.com:example/gamma-app.git
    store: gamma-app
stores:
  alpha:     { engram: [alpha, alpha-docs, alpha-cli, alpha-web, alpha-api, alpha-lib] }
  beta:      { engram: [beta-plugin, beta] }
  gamma-app: { engram: [gamma-app] }
`

func TestAddRepoMapping_RemoteEntryLandsAfterLastEntryAndKeepsEverythingElse(t *testing.T) {
	entry := specter.RepoEntry{Remote: originURL, Store: "gamma-app"}
	want := strings.Replace(realReposYAML, "stores:\n",
		"  - remote: "+originURL+"\n    store: gamma-app\nstores:\n", 1)

	got, err := specter.AddRepoMapping([]byte(realReposYAML), entry)

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("result =\n%s\nwant\n%s", got, want)
	}
	view := specter.ParseRepos(got)
	if view.Err != nil || !reflect.DeepEqual(view.Repos[len(view.Repos)-1], entry) {
		t.Errorf("re-parsed repos = %+v (err %v), want %+v last", view.Repos, view.Err, entry)
	}
}

func TestAddRepoMapping_PathEntryWhenThereIsNoRemote(t *testing.T) {
	entry := specter.RepoEntry{Path: "~/work/proj", Store: "alpha"}
	want := strings.Replace(realReposYAML, "stores:\n", "  - path: ~/work/proj\n    store: alpha\nstores:\n", 1)

	got, err := specter.AddRepoMapping([]byte(realReposYAML), entry)

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("result =\n%s\nwant\n%s", got, want)
	}
}

func TestAddRepoMapping_BlankAndCommentLinesAfterLastEntryStayBelowIt(t *testing.T) {
	const old = "repos:\n  - remote: git@x:a.git\n    store: a\n\n# engram map follows\nstores:\n  a: { engram: [a] }\n"
	const want = "repos:\n  - remote: git@x:a.git\n    store: a\n  - remote: git@x:b.git\n    store: a\n\n# engram map follows\nstores:\n  a: { engram: [a] }\n"

	got, err := specter.AddRepoMapping([]byte(old), specter.RepoEntry{Remote: "git@x:b.git", Store: "a"})

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("result =\n%s\nwant\n%s", got, want)
	}
}

func TestAddRepoMapping_ReposBlockAtEndOfFileAppends(t *testing.T) {
	const old = "# map\nrepos:\n  - remote: git@x:a.git\n    store: a\n"
	const want = old + "  - remote: git@x:b.git\n    store: a\n"

	got, err := specter.AddRepoMapping([]byte(old), specter.RepoEntry{Remote: "git@x:b.git", Store: "a"})

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Errorf("result =\n%s\nwant\n%s", got, want)
	}
}

func TestAddRepoMapping_NoReposKeyIsAnError(t *testing.T) {
	_, err := specter.AddRepoMapping([]byte("stores:\n  a: { engram: [a] }\n"), specter.RepoEntry{Remote: "git@x:b.git", Store: "a"})

	if err == nil || !strings.Contains(err.Error(), "repos:") {
		t.Errorf("err = %v, want one naming the missing repos: list", err)
	}
}

func TestAddRepoMapping_AlreadyMappedReturnsInputUnchanged(t *testing.T) {
	entry := specter.RepoEntry{Remote: "git@github.com:example/gamma-app.git", Store: "gamma-app"}

	got, err := specter.AddRepoMapping([]byte(realReposYAML), entry)

	if err != nil {
		t.Fatal(err)
	}
	if string(got) != realReposYAML {
		t.Errorf("result changed for an already mapped repo:\n%s", got)
	}
}
