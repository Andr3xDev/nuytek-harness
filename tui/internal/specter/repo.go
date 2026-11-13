package specter

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/repo"
)

const (
	gitMarker     = ".git"
	originSection = `[remote "origin"]`
)

// CurrentRepo is the git repository containing the working directory; Remote is empty when unknown.
type CurrentRepo struct {
	Found  bool
	Path   string
	Remote string
}

// DetectRepo reads .git/config directly and never runs git. A `.git` file (worktree) yields no remote.
func DetectRepo(env check.Env) CurrentRepo {
	dir, ok := repo.FindUp(env.FS, env.Cwd, gitMarker)
	if !ok {
		return CurrentRepo{}
	}
	cur := CurrentRepo{Found: true, Path: dir}
	gitPath := filepath.Join(dir, gitMarker)
	if info, err := env.FS.Lstat(gitPath); err != nil || !info.IsDir() {
		return cur
	}
	if data, err := readCapped(env.FS, filepath.Join(gitPath, "config"), maxConfigBytes); err == nil {
		cur.Remote = stripUserinfo(originURL(string(data)))
	}
	return cur
}

// stripUserinfo drops credentials from a remote so a token never reaches the tracked repos.yaml: all userinfo for
// http(s), the password for any other scheme:// URL (ssh://git@host keeps its login name). An unparsable URL yields ""
// rather than risk writing the credentials; scp-like remotes carry no password and pass through.
func stripUserinfo(remote string) string {
	if !strings.Contains(remote, "://") {
		return remote
	}
	u, err := url.Parse(remote)
	if err != nil {
		return ""
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		u.User = nil
	default:
		if u.User != nil {
			u.User = url.User(u.User.Username())
		}
	}
	return u.String()
}

func originURL(gitConfig string) string {
	inOrigin := false
	for _, line := range strings.Split(gitConfig, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			end := strings.Index(line, "]")
			inOrigin = end >= 0 && line[:end+1] == originSection
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok && inOrigin && strings.EqualFold(strings.TrimSpace(key), "url") {
			return configValue(value)
		}
	}
	return ""
}

// configValue unquotes a git config value or drops its trailing ` #`/` ;` comment.
func configValue(value string) string {
	value = strings.TrimSpace(value)
	if rest, ok := strings.CutPrefix(value, `"`); ok {
		unquoted, _, _ := strings.Cut(rest, `"`)
		return unquoted
	}
	for _, marker := range []string{" #", " ;"} {
		value, _, _ = strings.Cut(value, marker)
	}
	return strings.TrimSpace(value)
}

// MatchRepo prefers an exact remote; otherwise the longest `path:` entry that is an ancestor of, or equal to, the repo path.
func MatchRepo(repos []RepoEntry, current CurrentRepo, home string) (RepoEntry, bool) {
	var best RepoEntry
	bestLen := -1
	currentRemote := stripUserinfo(current.Remote)
	for _, entry := range repos {
		if currentRemote != "" && stripUserinfo(entry.Remote) == currentRemote {
			return entry, true
		}
		if entry.Path == "" {
			continue
		}
		p := entryPath(entry, home)
		if (current.Path == p || strings.HasPrefix(current.Path, strings.TrimSuffix(p, "/")+"/")) && len(p) > bestLen {
			best, bestLen = entry, len(p)
		}
	}
	return best, bestLen >= 0
}

func entryPath(entry RepoEntry, home string) string {
	return filepath.Clean(expandHome(entry.Path, home))
}

func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		return filepath.Join(home, rest)
	}
	return path
}
