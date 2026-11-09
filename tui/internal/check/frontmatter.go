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

type frontmatterFields struct {
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
				r := Result{CheckID: id, Subject: rel, Status: StatusOK}
				if err := validateFrontmatter(env.FS, filepath.Join(root, filepath.FromSlash(rel))); err != nil {
					r.Status, r.Message = StatusError, err.Error()
				}
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

func validateFrontmatter(fsys FS, file string) error {
	data, err := fsys.ReadFile(file)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if strings.TrimRight(lines[0], "\r") != frontmatterFence {
		return errors.New("missing opening --- fence")
	}
	end := slices.IndexFunc(lines[1:], func(l string) bool { return strings.TrimRight(l, "\r") == frontmatterFence })
	if end < 0 {
		return errors.New("missing closing --- fence")
	}
	var fields frontmatterFields
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:1+end], "\n")), &fields); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}
	if fields.Name == "" {
		return errors.New("missing name")
	}
	if fields.Description == "" {
		return errors.New("missing description")
	}
	return nil
}
