// Package config validates and formats user-configured paths.
package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

// Expand resolves a leading "~" against home and cleans the result.
func Expand(raw, home string) string {
	switch {
	case raw == "~":
		return filepath.Clean(home)
	case strings.HasPrefix(raw, "~/"):
		return filepath.Join(home, raw[2:])
	}
	return filepath.Clean(raw)
}

func expandAbsolute(home, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("path is empty")
	}
	path := Expand(raw, home)
	if !filepath.IsAbs(path) {
		return "", errors.New("must be an absolute path")
	}
	return path, nil
}

// ValidateSpecterRoot returns the expanded path of an existing directory, or "" and an error.
func ValidateSpecterRoot(fs check.FS, home, raw string) (string, error) {
	path, err := expandAbsolute(home, raw)
	if err != nil {
		return "", err
	}
	info, err := fs.Stat(path)
	if err != nil {
		return "", errors.New("directory does not exist")
	}
	if !info.IsDir() {
		return "", errors.New("not a directory")
	}
	return path, nil
}

// ValidateBackupsDir returns the expanded path, rejecting ownTree and anything inside it, the filesystem
// root, HOME itself and any reserved path (such as the repo's own store) or its ancestor. Checks are
// lexical. The directory need not exist: the backup run creates it.
func ValidateBackupsDir(home, raw, ownTree string, reserved ...string) (string, error) {
	path, err := expandAbsolute(home, raw)
	if err != nil {
		return "", err
	}
	if path == string(filepath.Separator) || path == filepath.Clean(home) {
		return "", errors.New("too broad: choose a dedicated directory")
	}
	for _, r := range reserved {
		if isWithin(r, path) || isWithin(path, r) {
			return "", fmt.Errorf("overlaps backup store %s", r)
		}
	}
	if isWithin(ownTree, path) {
		return "", fmt.Errorf("inside %s", ownTree)
	}
	return path, nil
}

// isWithin is true when path is root or below it. Segment-wise via Rel: a string prefix would wrongly
// match "~/.claudeX" for "~/.claude".
func isWithin(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// PathLabel shows a path under home as "~/...", anything else unchanged.
func PathLabel(home, path string) string {
	home = filepath.Clean(home)
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	if rel == "." {
		return "~"
	}
	return "~/" + rel
}
