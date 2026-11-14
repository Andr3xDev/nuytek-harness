package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

const frontmatterFence = "---"

// Set selects markdown files under a home-relative Root by Pattern (filepath.Match, relative to Root); Skip lists base names to ignore.
type Set struct {
	Root, Pattern string
	Skip          []string
}

type FrontmatterFields struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func Frontmatter(id string, sets []Set) Check {
	return Check{ID: id, Run: func(_ context.Context, env Env) []Result {
		var results []Result
		for _, set := range sets {
			root := filepath.Join(env.Home, set.Root)
			// A pattern of N segments only matches files N levels deep.
			files, err := listFiles(env.FS, root, "", strings.Count(set.Pattern, "/")+1)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				results = append(results, Result{CheckID: id, Subject: set.Root, Status: StatusUnknown, Message: err.Error()})
				continue
			}
			for _, rel := range files {
				if ok, _ := path.Match(set.Pattern, rel); !ok || slices.Contains(set.Skip, path.Base(rel)) {
					continue
				}
				r := validateFrontmatter(env.FS, filepath.Join(root, filepath.FromSlash(rel)))
				r.CheckID, r.Subject = id, rel
				results = append(results, r)
			}
		}
		return results
	}}
}

// listFiles returns slash-separated paths of files under dir (relative to prefix), at most depth levels deep.
func listFiles(fsys FS, dir, prefix string, depth int) ([]string, error) {
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		rel := path.Join(prefix, e.Name())
		if !e.IsDir() {
			files = append(files, rel)
			continue
		}
		if depth > 1 {
			sub, err := listFiles(fsys, filepath.Join(dir, e.Name()), rel, depth-1)
			if err != nil {
				return nil, err
			}
			files = append(files, sub...)
		}
	}
	return files, nil
}

// ParseFrontmatter reads the YAML block between the --- fences; empty fields are not an error.
func ParseFrontmatter(fsys FS, file string) (FrontmatterFields, error) {
	var fields FrontmatterFields
	data, err := fsys.ReadFile(file)
	if err != nil {
		return fields, err
	}
	lines := strings.Split(string(data), "\n")
	if strings.TrimRight(lines[0], "\r") != frontmatterFence {
		return fields, errors.New("missing opening --- fence")
	}
	end := slices.IndexFunc(lines[1:], func(l string) bool { return strings.TrimRight(l, "\r") == frontmatterFence })
	if end < 0 {
		return fields, errors.New("missing closing --- fence")
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:1+end], "\n")), &fields); err != nil {
		return fields, fmt.Errorf("invalid YAML: %w", err)
	}
	return fields, nil
}

func validateFrontmatter(fsys FS, file string) Result {
	fields, err := ParseFrontmatter(fsys, file)
	switch {
	case err != nil:
		return Result{Status: StatusError, Message: err.Error()}
	case fields.Name == "":
		return Result{Status: StatusError, Message: "missing name"}
	case fields.Description == "":
		return Result{Status: StatusError, Message: "missing description", Suggestion: "add a description: field to the frontmatter"}
	}
	return Result{Status: StatusOK}
}
