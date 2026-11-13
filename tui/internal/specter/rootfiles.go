package specter

import (
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

// RootFile is an agent-facing file that names the Specter root; RepoPath is relative to the harness repo, HomePath to HOME.
type RootFile struct {
	Label    string
	RepoPath string
	HomePath string
}

// RootFiles is the contract list (design D6); targets/claude.toml decides where each deploys.
var RootFiles = []RootFile{
	{Label: "global instructions", RepoPath: "claude/CLAUDE.md", HomePath: ".claude/CLAUDE.md"},
	{Label: "orchestrator agent", RepoPath: "agents/tech-orchestrator.md", HomePath: ".claude/agents/tech-orchestrator.md"},
	{Label: "guard hook", RepoPath: "claude/hooks/infra-agent-bash-guard.py", HomePath: ".claude/hooks/infra-agent-bash-guard.py"},
	{Label: "specter-ops agent", RepoPath: "agents/sub-agents/infrastructure/specter-ops.md", HomePath: ".claude/agents/sub-agents/infrastructure/specter-ops.md"},
	{Label: "/specter command", RepoPath: "claude/commands/specter.md", HomePath: ".claude/commands/specter.md"},
	{Label: "SDD rules", RepoPath: "shared/common-sdd.md", HomePath: ".claude/common-sdd.md"},
	{Label: "git-ops agent", RepoPath: "agents/sub-agents/general/git-ops.md", HomePath: ".claude/agents/sub-agents/general/git-ops.md"},
	{Label: "engram-ops agent", RepoPath: "agents/sub-agents/infrastructure/engram-ops.md", HomePath: ".claude/agents/sub-agents/infrastructure/engram-ops.md"},
}

type RootCopy string

const (
	CopyRepo     RootCopy = "repo"
	CopyDeployed RootCopy = "deployed"
)

// RootRef is what one listed file says about the root; Copy is empty when Found is false.
type RootRef struct {
	Label      string
	Copy       RootCopy
	Found      bool
	References bool
}

// LoadRootRefs reads each listed file, preferring the repo copy over the deployed one.
func LoadRootRefs(env check.Env, root Root) []RootRef {
	forms := rootForms(root.Path, env.Home)
	refs := make([]RootRef, len(RootFiles))
	for i, f := range RootFiles {
		refs[i] = RootRef{Label: f.Label}
		for _, src := range []struct {
			copy RootCopy
			base string
			rel  string
		}{{CopyRepo, env.Repo, f.RepoPath}, {CopyDeployed, env.Home, f.HomePath}} {
			if src.base == "" {
				continue
			}
			data, err := readCapped(env.FS, filepath.Join(src.base, src.rel), maxConfigBytes)
			if err != nil {
				continue
			}
			refs[i].Copy, refs[i].Found = src.copy, true
			refs[i].References = containsAny(string(data), forms)
			break
		}
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
			if end == len(s) || !isNameByte(s[end]) {
				return true
			}
			rest, offset = s[end:], end
		}
	}
	return false
}

// isNameByte reports a byte that would continue a path element, so `~/dev/specter-old` is not `~/dev/specter`.
func isNameByte(b byte) bool {
	return b == '-' || b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
