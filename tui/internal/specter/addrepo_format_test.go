package specter_test

import (
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestAddRepoMapping_FormatPreserved(t *testing.T) {
	entry := specter.RepoEntry{Remote: "git@x:b.git", Store: "a"}
	tests := []struct {
		name string
		old  string
		want string
	}{
		{
			"CRLF file gets CRLF lines",
			"repos:\r\n  - remote: git@x:a.git\r\n    store: a\r\nstores:\r\n  a: { engram: [a] }\r\n",
			"repos:\r\n  - remote: git@x:a.git\r\n    store: a\r\n  - remote: git@x:b.git\r\n    store: a\r\nstores:\r\n  a: { engram: [a] }\r\n",
		},
		{
			"no trailing newline",
			"repos:\n  - remote: git@x:a.git\n    store: a",
			"repos:\n  - remote: git@x:a.git\n    store: a\n  - remote: git@x:b.git\n    store: a\n",
		},
		{
			"indent copied from the existing items",
			"repos:\n- remote: git@x:a.git\n  store: a\n",
			"repos:\n- remote: git@x:a.git\n  store: a\n- remote: git@x:b.git\n  store: a\n",
		},
		{
			"four space indent",
			"repos:\n    - remote: git@x:a.git\n      store: a\n",
			"repos:\n    - remote: git@x:a.git\n      store: a\n    - remote: git@x:b.git\n      store: a\n",
		},
		{
			"key with trailing comment",
			"repos: # map\n  - remote: git@x:a.git\n    store: a\n",
			"repos: # map\n  - remote: git@x:a.git\n    store: a\n  - remote: git@x:b.git\n    store: a\n",
		},
		{
			"empty block",
			"repos:\nstores:\n  a: { engram: [a] }\n",
			"repos:\n  - remote: git@x:b.git\n    store: a\nstores:\n  a: { engram: [a] }\n",
		},
		{
			"empty block at end without newline",
			"repos:",
			"repos:\n  - remote: git@x:b.git\n    store: a\n",
		},
		{
			"empty flow list",
			"repos: []\nstores:\n",
			"repos:\n  - remote: git@x:b.git\n    store: a\nstores:\n",
		},
		{
			"empty flow list with comment",
			"repos: [] # none yet\n",
			"repos: # none yet\n  - remote: git@x:b.git\n    store: a\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := specter.AddRepoMapping([]byte(tt.old), entry)

			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("result = %q, want %q", got, tt.want)
			}
			if view := specter.ParseRepos(got); view.Err != nil || view.Repos[len(view.Repos)-1] != entry {
				t.Errorf("re-parsed = %+v (err %v), want %+v last", view.Repos, view.Err, entry)
			}
		})
	}
}

func TestAddRepoMapping_ValuesThatNeedQuotingRoundTrip(t *testing.T) {
	for _, remote := range []string{"https://h/x # y", "https://h/a: b", "git@x:b.git #", "'quoted'"} {
		t.Run(remote, func(t *testing.T) {
			entry := specter.RepoEntry{Remote: remote, Store: "a"}

			got, err := specter.AddRepoMapping([]byte("repos:\n  - remote: git@x:a.git\n    store: a\n"), entry)

			if err != nil {
				t.Fatal(err)
			}
			if view := specter.ParseRepos(got); view.Err != nil || len(view.Repos) != 2 || view.Repos[1] != entry {
				t.Errorf("re-parsed = %+v (err %v), want %+v second\n%s", view.Repos, view.Err, entry, got)
			}
		})
	}
}

func TestAddRepoMapping_MultilineValueIsRejectedNotWritten(t *testing.T) {
	_, err := specter.AddRepoMapping([]byte("repos:\n  - remote: git@x:a.git\n    store: a\n"),
		specter.RepoEntry{Remote: "git@x:b.git\nstore: evil", Store: "a"})

	if err == nil {
		t.Error("err = nil, want the entry rejected")
	}
}
