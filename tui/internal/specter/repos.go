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

// ChangeRepoStore rewrites only the store value of the entry matching entry's remote (or path when it has none).
func ChangeRepoStore(old []byte, entry RepoEntry, store string) ([]byte, error) {
	lines, item, first, _, err := findRepoEntry(old, entry)
	if err != nil {
		return nil, err
	}
	value := mappingValue(item, "store")
	if value == nil {
		return nil, errors.New("repo entry has no store")
	}
	i := value.Line - 1
	if i < first || i >= len(lines) {
		return nil, errors.New("repo entry store is not on the entry's lines")
	}
	// shortcut: a quoted store value is assumed to have no escapes, so its raw width is len+2.
	width := len([]rune(value.Value))
	if value.Style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle) != 0 {
		width += 2
	}
	line := []rune(lines[i])
	start := value.Column - 1
	if start < 0 || start+width > len(line) {
		return nil, errors.New("repo entry store position does not match the file")
	}
	lines[i] = string(line[:start]) + yamlScalar(store) + string(line[start+width:])
	var want RepoEntry
	if err := item.Decode(&want); err != nil {
		return nil, err
	}
	want.Store = store
	return checkedRepos(lines, len(ParseRepos(old).Repos), want, true)
}

// RemoveRepoMapping drops the lines of the entry matching entry's remote (or path when it has none).
func RemoveRepoMapping(old []byte, entry RepoEntry) ([]byte, error) {
	lines, item, first, end, err := findRepoEntry(old, entry)
	if err != nil {
		return nil, err
	}
	var gone RepoEntry
	if err := item.Decode(&gone); err != nil {
		return nil, err
	}
	return checkedRepos(slices.Delete(lines, first, end), len(ParseRepos(old).Repos)-1, gone, false)
}

// checkedRepos parses the edited file and requires wantLen entries, and entry among them when present is set,
// so a line edit that hit the wrong bytes fails instead of being written.
func checkedRepos(lines []string, wantLen int, entry RepoEntry, present bool) ([]byte, error) {
	out := []byte(strings.Join(lines, ""))
	view := ParseRepos(out)
	if view.Err != nil {
		return nil, view.Err
	}
	if len(view.Repos) != wantLen || (present && !slices.Contains(view.Repos, entry)) {
		return nil, errors.New("repos.yaml would not hold the edit as written")
	}
	return out, nil
}

var errReposFlowList = errors.New("unsupported: repos is a flow list; rewrite it as a block list to edit it from here")

// findRepoEntry returns the file lines, the matching item node and the entry's line range [first, end).
// The range stops before the next item, and trailing blank or comment lines belong to what follows.
func findRepoEntry(old []byte, entry RepoEntry) (lines []string, item *yaml.Node, first, end int, err error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(old, &doc); err != nil {
		return nil, nil, 0, 0, err
	}
	var items []*yaml.Node
	if len(doc.Content) > 0 {
		if seq := mappingValue(doc.Content[0], "repos"); seq != nil {
			if seq.Style&yaml.FlowStyle != 0 {
				return nil, nil, 0, 0, errReposFlowList
			}
			items = seq.Content
		}
	}
	at := slices.IndexFunc(items, func(n *yaml.Node) bool {
		var e RepoEntry
		if n.Decode(&e) != nil {
			return false
		}
		if entry.Remote != "" {
			return e.Remote == entry.Remote
		}
		return entry.Path != "" && e.Path == entry.Path
	})
	if at < 0 {
		return nil, nil, 0, 0, errors.New("repo mapping not found in repos.yaml")
	}

	lines = strings.SplitAfter(string(old), "\n")
	first = items[at].Line - 1
	end = len(lines)
	if at+1 < len(items) {
		end = items[at+1].Line - 1
	} else if i := slices.IndexFunc(lines[first+1:], endsReposList); i >= 0 {
		end = first + 1 + i
	}
	for end-1 > first && isBlankOrComment(lines[end-1]) {
		end--
	}
	return lines, items[at], first, end, nil
}

// mappingValue is the value node of key in a mapping node, or nil.
func mappingValue(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
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
