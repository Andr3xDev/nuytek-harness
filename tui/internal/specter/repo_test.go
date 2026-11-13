package specter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const originURL = "git@github.com:example/demo.git"

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectRepo_ReadsOriginFromGitConfigWalkingUp(t *testing.T) {
	repoDir := t.TempDir()
	writeFile(t, filepath.Join(repoDir, ".git", "config"),
		"[core]\n\tbare = false\n[remote \"upstream\"]\n\turl = git@example.com:other.git\n[remote \"origin\"]\n\turl = "+originURL+"\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n")
	sub := filepath.Join(repoDir, "pkg", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	env := envWith(t, &fakeExec{}, "")
	env.Cwd = sub

	cur := specter.DetectRepo(env)

	if !cur.Found || cur.Path != repoDir || cur.Remote != originURL {
		t.Errorf("repo = %+v, want Found at %s with remote %s", cur, repoDir, originURL)
	}
}

func TestDetectRepo_GitFileWorktreeHasNoRemote(t *testing.T) {
	repoDir := t.TempDir()
	writeFile(t, filepath.Join(repoDir, ".git"), "gitdir: /elsewhere/.git/worktrees/w\n")
	env := envWith(t, &fakeExec{}, "")
	env.Cwd = repoDir

	cur := specter.DetectRepo(env)

	if !cur.Found || cur.Path != repoDir || cur.Remote != "" {
		t.Errorf("repo = %+v, want Found at %s with no remote", cur, repoDir)
	}
}

func TestDetectRepo_OutsideGitRepoIsNotFound(t *testing.T) {
	env := envWith(t, &fakeExec{}, "")
	env.Cwd = t.TempDir()

	if cur := specter.DetectRepo(env); cur.Found {
		t.Errorf("repo = %+v, want not found", cur)
	}
}

func TestOriginURL_ToleratesGitConfigSyntaxVariants(t *testing.T) {
	tests := map[string]string{
		"header comment": "[remote \"origin\"] # main\n\turl = " + originURL + "\n",
		"header spacing": "[remote \"origin\"]   \n\turl = " + originURL + "\n",
		"no spaces":      "[remote \"origin\"]\n\turl=" + originURL + "\n",
		"quoted value":   "[remote \"origin\"]\n\turl = \"" + originURL + "\"\n",
		"inline comment": "[remote \"origin\"]\n\turl = " + originURL + " # mirror\n",
		"uppercase key":  "[remote \"origin\"]\n\tURL = " + originURL + "\n",
	}
	for name, config := range tests {
		t.Run(name, func(t *testing.T) {
			repoDir := t.TempDir()
			writeFile(t, filepath.Join(repoDir, ".git", "config"), config)
			env := envWith(t, &fakeExec{}, "")
			env.Cwd = repoDir

			if got := specter.DetectRepo(env).Remote; got != originURL {
				t.Errorf("Remote = %q, want %q", got, originURL)
			}
		})
	}
}

func TestDetectRepo_StripsCredentialsFromHTTPSOrigin(t *testing.T) {
	tests := map[string]string{
		"https://user:token@github.com/example/demo.git": "https://github.com/example/demo.git",
		"https://token@github.com/example/demo.git":      "https://github.com/example/demo.git",
		"https://github.com/example/demo.git":            "https://github.com/example/demo.git",
		"ssh://git@github.com/example/demo.git":          "ssh://git@github.com/example/demo.git",
		"https://user:tok%zzen@github.com/x.git":         "",
	}
	for configured, want := range tests {
		t.Run(configured, func(t *testing.T) {
			repoDir := t.TempDir()
			writeFile(t, filepath.Join(repoDir, ".git", "config"), "[remote \"origin\"]\n\turl = "+configured+"\n")
			env := envWith(t, &fakeExec{}, "")
			env.Cwd = repoDir

			if got := specter.DetectRepo(env).Remote; got != want {
				t.Errorf("Remote = %q, want %q", got, want)
			}
		})
	}
}

func TestMatchRepo_ComparesRemotesWithoutCredentials(t *testing.T) {
	repos := []specter.RepoEntry{{Remote: "https://old:tok@github.com/example/demo.git", Store: "demo"}}
	current := specter.CurrentRepo{Found: true, Path: "/srv/x", Remote: "https://github.com/example/demo.git"}

	if entry, ok := specter.MatchRepo(repos, current, "/home/u"); !ok || entry.Store != "demo" {
		t.Errorf("MatchRepo = (%+v, %v), want the demo entry", entry, ok)
	}
}

func TestMatchRepo_CleansEntryPaths(t *testing.T) {
	const home = "/home/u"
	repos := []specter.RepoEntry{
		{Path: "~/", Store: "home"},
		{Path: "/home/u/work/", Store: "work"},
		{Path: "~/work/client/", Store: "client"},
	}
	tests := []struct{ path, want string }{
		{"/home/u/work/app", "work"},
		{"/home/u/work/client", "client"},
		{"/home/u/work/client/api", "client"},
		{"/home/u/other", "home"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			entry, ok := specter.MatchRepo(repos, specter.CurrentRepo{Found: true, Path: tt.path}, home)

			if !ok || entry.Store != tt.want {
				t.Errorf("MatchRepo = (%+v, %v), want store %q", entry, ok, tt.want)
			}
		})
	}
}

func TestMatchRepo(t *testing.T) {
	const home = "/home/u"
	repos := []specter.RepoEntry{
		{Remote: originURL, Store: "by-remote"},
		{Path: "~/work", Store: "work"},
		{Path: "~/work/client", Store: "client"},
	}
	tests := []struct {
		name    string
		current specter.CurrentRepo
		want    string // store of the matched entry; "" = no match
	}{
		{"exact remote", specter.CurrentRepo{Found: true, Path: "/srv/x", Remote: originURL}, "by-remote"},
		{"https clone of an ssh remote does not match", specter.CurrentRepo{Found: true, Path: "/srv/x", Remote: "https://github.com/example/demo.git"}, ""},
		{"path prefix", specter.CurrentRepo{Found: true, Path: "/home/u/work/app"}, "work"},
		{"longest path wins", specter.CurrentRepo{Found: true, Path: "/home/u/work/client/api"}, "client"},
		{"path equal to entry", specter.CurrentRepo{Found: true, Path: "/home/u/work"}, "work"},
		{"prefix must end on an element boundary", specter.CurrentRepo{Found: true, Path: "/home/u/workshop"}, ""},
		{"no match", specter.CurrentRepo{Found: true, Path: "/home/u/other"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := specter.MatchRepo(repos, tt.current, home)

			if ok != (tt.want != "") || entry.Store != tt.want {
				t.Errorf("MatchRepo = (%+v, %v), want store %q", entry, ok, tt.want)
			}
		})
	}
}
