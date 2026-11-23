package specter

import (
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/render"
)

// RootFile is an agent-facing file that names the Specter root; RepoPath is relative to the harness repo, HomePath to HOME.
type RootFile struct {
	Label    string
	RepoPath string
	HomePath string
}

// RootFiles lists the deployed files that name the Specter root; targets/claude.toml decides where each deploys.
// Keep in sync with targets/rendered-files.txt (TestRootFiles_MatchRenderedFilesList).
var RootFiles = []RootFile{
	{Label: "global instructions", RepoPath: "claude/CLAUDE.md", HomePath: ".claude/CLAUDE.md"},
	{Label: "orchestrator agent", RepoPath: "agents/tech-orchestrator.md", HomePath: ".claude/agents/tech-orchestrator.md"},
	{Label: "specter-ops agent", RepoPath: "agents/sub-agents/infrastructure/specter-ops.md", HomePath: ".claude/agents/sub-agents/infrastructure/specter-ops.md"},
	{Label: "/specter command", RepoPath: "claude/commands/specter.md", HomePath: ".claude/commands/specter.md"},
	{Label: "SDD rules", RepoPath: "shared/common-sdd.md", HomePath: ".claude/common-sdd.md"},
	{Label: "git-ops agent", RepoPath: "agents/sub-agents/general/git-ops.md", HomePath: ".claude/agents/sub-agents/general/git-ops.md"},
	{Label: "engram-ops agent", RepoPath: "agents/sub-agents/infrastructure/engram-ops.md", HomePath: ".claude/agents/sub-agents/infrastructure/engram-ops.md"},
}

// RootRef is what the deployed copy of one listed file says about the root.
type RootRef struct {
	Label      string
	Found      bool
	References bool
}

// LoadRootRefs reads the deployed copy of each listed file; the repo copy holds a placeholder, not a root.
func LoadRootRefs(env check.Env, root Root) []RootRef {
	forms := rootForms(root.Path, env.Home)
	refs := make([]RootRef, len(RootFiles))
	for i, f := range RootFiles {
		refs[i] = RootRef{Label: f.Label}
		data, err := readCapped(env.FS, filepath.Join(env.Home, f.HomePath), maxConfigBytes)
		if err != nil {
			continue
		}
		refs[i].Found = true
		refs[i].References = containsAny(string(data), forms)
	}
	return refs
}

// rootForms lists the equivalent spellings of root; the HOME-relative ones apply only strictly under HOME.
func rootForms(root, home string) []string {
	forms := []string{root}
	if rel, err := filepath.Rel(home, root); err == nil && rel != "." && !outsideHome(home, root) {
		forms = append(forms, "~/"+rel, "$HOME/"+rel, "${HOME}/"+rel)
	}
	return forms
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		for rest, offset := s, 0; ; {
			i := strings.Index(rest, sub)
			if i < 0 {
				break
			}
			end := offset + i + len(sub)
			if end == len(s) || !render.IsNameByte(s[end]) {
				return true
			}
			rest, offset = s[end:], end
		}
	}
	return false
}
