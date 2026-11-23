package specter_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

func TestLoad_ProbesGitAndGhAuthThroughExec(t *testing.T) {
	tests := []struct {
		name     string
		onPath   map[string]bool
		authExit int
		wantGit  bool
		wantGh   specter.GhInfo
		wantAuth bool // whether `gh auth status` must have run
	}{
		{"both ready", map[string]bool{"git": true, "gh": true}, 0, true, specter.GhInfo{Found: true, Authenticated: true}, true},
		{"gh not authenticated", map[string]bool{"git": true, "gh": true}, 1, true, specter.GhInfo{Found: true}, true},
		{"gh missing", map[string]bool{"git": true}, 0, true, specter.GhInfo{}, false},
		{"git missing", map[string]bool{"gh": true}, 0, false, specter.GhInfo{Found: true, Authenticated: true}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &fakeExec{onPath: tt.onPath, results: map[string]check.ExecResult{"gh auth status": {ExitCode: tt.authExit}}}

			facts := specter.Load(context.Background(), envWith(t, exec, t.TempDir()))

			if facts.Git.Found != tt.wantGit || facts.Gh != tt.wantGh {
				t.Errorf("Git.Found = %v, Gh = %+v, want %v and %+v", facts.Git.Found, facts.Gh, tt.wantGit, tt.wantGh)
			}
			if ran := len(exec.callsTo("gh auth status")) > 0; ran != tt.wantAuth {
				t.Errorf("gh auth status ran = %v, want %v", ran, tt.wantAuth)
			}
		})
	}
}

func TestLoad_IsRepoOnlyWithTopLevelReposYAML(t *testing.T) {
	withRepos := t.TempDir()
	writeFile(t, filepath.Join(withRepos, "repos.yaml"), "repos: []\n")
	nested := t.TempDir()
	writeFile(t, filepath.Join(nested, "sub", "repos.yaml"), "repos: []\n")
	tests := []struct {
		name string
		root string
		want bool
	}{
		{"root with repos.yaml", withRepos, true},
		{"existing folder without repos.yaml", t.TempDir(), false},
		{"repos.yaml only in a subfolder", nested, false},
		{"missing root", filepath.Join(t.TempDir(), "absent"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := specter.Load(context.Background(), envWith(t, &fakeExec{}, tt.root))

			if facts.IsRepo != tt.want {
				t.Errorf("IsRepo = %v, want %v", facts.IsRepo, tt.want)
			}
		})
	}
}
