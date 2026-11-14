package specter_test

import (
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

// Same shape as a real repos.yaml with fictional values: header comments, block repos, flow-style stores.
const reposYAML = `# Repo -> OpenSpec store map (` + "`repos:`" + `), read by the orchestrator.
# ` + "`stores:`" + ` maps each store to the Engram projects its memories live in.
repos:
  - remote: git@github.com:example/alpha.git
    store: alpha
  - path: ~/dev/example-alpha
    store: alpha
  - remote: git@github.com:example/gamma-app.git
    store: gamma-app
stores:
  alpha:     { engram: [alpha, alpha-docs, alpha-cli] }
  gamma-app: { engram: [gamma-app] }
`

func TestParseRepos_ReadsReposAndStores(t *testing.T) {
	view := specter.ParseRepos([]byte(reposYAML))

	if view.Err != nil {
		t.Fatalf("Err = %v, want nil", view.Err)
	}
	want := []specter.RepoEntry{
		{Remote: "git@github.com:example/alpha.git", Store: "alpha"},
		{Path: "~/dev/example-alpha", Store: "alpha"},
		{Remote: "git@github.com:example/gamma-app.git", Store: "gamma-app"},
	}
	if !reflect.DeepEqual(view.Repos, want) {
		t.Errorf("Repos = %+v, want %+v", view.Repos, want)
	}
	if got := view.Stores["alpha"].Engram; !reflect.DeepEqual(got, []string{"alpha", "alpha-docs", "alpha-cli"}) {
		t.Errorf("alpha engram = %v", got)
	}
}

func TestParseRepos_IgnoresUnknownKeys(t *testing.T) {
	view := specter.ParseRepos([]byte("version: 3\nrepos:\n  - remote: r\n    store: a\n    note: x\nextra: {a: 1}\n"))

	if view.Err != nil || len(view.Repos) != 1 || view.Repos[0].Store != "a" {
		t.Errorf("view = %+v, want one repo and no Err", view)
	}
}

func TestParseRepos_InvalidYAMLSetsOnlyErr(t *testing.T) {
	view := specter.ParseRepos([]byte("repos: [unclosed\n  - : :"))

	if view.Err == nil {
		t.Fatal("Err = nil, want a parse error")
	}
	if len(view.Repos) != 0 || len(view.Stores) != 0 {
		t.Errorf("view = %+v, want no repos or stores alongside Err", view)
	}
}
