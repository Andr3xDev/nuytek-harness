package specter

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

// maxConfigBytes caps every file read from outside the harness (repos.yaml, config.yaml, .git/config, root files).
const maxConfigBytes = 1 << 20

type RepoEntry struct {
	Remote string `yaml:"remote"`
	Path   string `yaml:"path"`
	Store  string `yaml:"store"`
}

type StoreMapping struct {
	Engram []string `yaml:"engram"`
}

// ReposView is the parsed repos.yaml; Err alone is set when the file is missing or invalid.
type ReposView struct {
	Repos  []RepoEntry
	Stores map[string]StoreMapping
	Err    error
}

func ParseRepos(data []byte) ReposView {
	var raw struct {
		Repos  []RepoEntry             `yaml:"repos"`
		Stores map[string]StoreMapping `yaml:"stores"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return ReposView{Err: err}
	}
	return ReposView{Repos: raw.Repos, Stores: raw.Stores}
}

// AddRepoMapping inserts entry after the last item of the `repos:` list and leaves every other byte alone,
// so comments and flow-style maps in the tracked file survive. An already mapped entry returns old unchanged.
func AddRepoMapping(old []byte, entry RepoEntry) ([]byte, error) {
	if entry.Remote == "" && entry.Path == "" {
		return nil, errors.New("repo entry has neither remote nor path")
	}
	view := ParseRepos(old)
	if view.Err != nil {
		return nil, view.Err
	}
	// Home "~" keeps a `~/rel` entry path comparable with the `~/rel` paths already in the file.
	if _, mapped := MatchRepo(view.Repos, CurrentRepo{Found: true, Path: entry.Path, Remote: entry.Remote}, "~"); mapped {
		return old, nil
	}

	eol := "\n"
	if bytes.Contains(old, []byte("\r\n")) {
		eol = "\r\n"
	}
	lines := strings.SplitAfter(string(old), "\n")
	at, comment, emptyFlow := -1, "", false
	for i, l := range lines {
		if m := reposKeyLine.FindStringSubmatch(strings.TrimRight(l, "\r\n")); m != nil {
			at, emptyFlow, comment = i, m[1] != "", m[2]
			if comment != "" {
				comment = " " + comment
			}
			break
		}
	}
	if at < 0 {
		return nil, errors.New("repos.yaml has no repos: list")
	}
	if emptyFlow {
		lines[at] = "repos:" + comment + eol
	}
	insertAt, indent := at+1, "  "
	foundItem := false
	for i := at + 1; i < len(lines) && !endsReposList(lines[i]); i++ {
		if isBlankOrComment(lines[i]) {
			continue
		}
		insertAt = i + 1
		if trimmed := strings.TrimLeft(lines[i], " \t"); !foundItem && strings.HasPrefix(trimmed, "-") {
			foundItem, indent = true, lines[i][:len(lines[i])-len(trimmed)]
		}
	}

	key, value := "remote", entry.Remote
	if entry.Remote == "" {
		key, value = "path", entry.Path
	}
	item := fmt.Sprintf("%s- %s: %s%s%s  store: %s%s", indent, key, yamlScalar(value), eol, indent, yamlScalar(entry.Store), eol)
	if !strings.HasSuffix(lines[insertAt-1], "\n") {
		lines[insertAt-1] += eol
	}
	result := strings.Join(slices.Insert(lines, insertAt, item), "")

	if !slices.Contains(ParseRepos([]byte(result)).Repos, entry) {
		return nil, errors.New("repos.yaml would not hold the new entry as written")
	}
	return []byte(result), nil
}

// reposKeyLine matches `repos:` with an optional empty flow list and trailing comment; group 1 is `[]`, group 2 the comment.
var reposKeyLine = regexp.MustCompile(`^repos:[ \t]*(\[\])?[ \t]*(#.*)?$`)

// yamlScalar quotes only when the value needs it, so a ` #` or `: ` in a remote cannot change the document.
func yamlScalar(value string) string {
	out, err := yaml.Marshal(value)
	if err != nil {
		return value
	}
	return strings.TrimSuffix(string(out), "\n")
}

// endsReposList is true for the first column-0 line that is neither blank, a comment nor a list item.
func endsReposList(line string) bool {
	return line != "" && !strings.ContainsRune(" \t#-\r\n", rune(line[0]))
}

func isBlankOrComment(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

var (
	errNotRegular = errors.New("not a regular file")
	errTooLarge   = errors.New("file too large")
)

// readCapped follows links (deployed configs are symlinks) but only reads regular files up to max bytes,
// so a link to a device or a huge file cannot hang or balloon memory.
func readCapped(fsys check.FS, path string, max int64) ([]byte, error) {
	info, err := fsys.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s: %w", path, errNotRegular)
	}
	if info.Size() > max {
		return nil, fmt.Errorf("%s: %w (limit %d bytes)", path, errTooLarge, max)
	}
	data, err := fsys.ReadFile(path)
	if err == nil && int64(len(data)) > max {
		return nil, fmt.Errorf("%s: %w (limit %d bytes)", path, errTooLarge, max)
	}
	return data, err
}
