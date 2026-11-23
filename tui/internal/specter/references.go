package specter

import (
	"bytes"
	"errors"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const referencesKey = "references:"

var (
	// referencesFlow matches `references: [a, b]` on one line; group 1 is the key part, group 2 what follows the list.
	referencesFlow      = regexp.MustCompile(`^(references:[ \t]*)\[[^\]]*\](.*)$`)
	referencesBlockHead = regexp.MustCompile(`^references:[ \t]*(#.*)?$`)
)

// AddReference adds id to the `references:` list of a store config and leaves every other byte alone.
// An id already listed returns old unchanged.
func AddReference(old []byte, id string) ([]byte, error) {
	var cfg struct {
		References []string `yaml:"references"`
	}
	if err := yaml.Unmarshal(old, &cfg); err != nil {
		return nil, err
	}
	if slices.Contains(cfg.References, id) {
		return old, nil
	}
	return SetReferences(old, append(cfg.References, id))
}

// SetReferences replaces the `references:` list, keeping flow or block style and the trailing comment of a flow list;
// a block list is rewritten item by item, so comments on its items are not kept. A missing key is appended. shortcut: a comment line between block items ends the list, upgrade if configs grow them.
func SetReferences(old []byte, ids []string) ([]byte, error) {
	eol := "\n"
	if bytes.Contains(old, []byte("\r\n")) {
		eol = "\r\n"
	}
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = yamlScalar(id)
	}
	lines := strings.SplitAfter(string(old), "\n")
	for i, line := range lines {
		text := strings.TrimRight(line, "\r\n")
		if !strings.HasPrefix(text, referencesKey) {
			continue
		}
		if m := referencesFlow.FindStringSubmatch(text); m != nil {
			lines[i] = m[1] + "[" + strings.Join(quoted, ", ") + "]" + m[2] + line[len(text):]
			return []byte(strings.Join(lines, "")), nil
		}
		if !referencesBlockHead.MatchString(text) {
			return nil, errors.New("references: has an unsupported form")
		}
		return []byte(replaceBlockItems(lines, i, quoted, eol)), nil
	}
	out := string(old)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += eol
	}
	return []byte(out + referencesKey + " [" + strings.Join(quoted, ", ") + "]" + eol), nil
}

// replaceBlockItems swaps the `- x` lines after the head line for items, keeping the first item's indent.
func replaceBlockItems(lines []string, head int, items []string, eol string) string {
	end, indent := head+1, "  "
	for ; end < len(lines) && strings.HasPrefix(strings.TrimLeft(lines[end], " \t"), "-"); end++ {
		if end == head+1 {
			indent = lines[end][:len(lines[end])-len(strings.TrimLeft(lines[end], " \t"))]
		}
	}
	var b strings.Builder
	b.WriteString(strings.Join(lines[:head+1], ""))
	if !strings.HasSuffix(lines[head], "\n") {
		b.WriteString(eol)
	}
	for _, item := range items {
		b.WriteString(indent + "- " + item + eol)
	}
	b.WriteString(strings.Join(lines[end:], ""))
	return b.String()
}
