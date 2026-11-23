// Package render swaps the Specter root placeholder for the resolved root in deployed files and back.
package render

import (
	"bytes"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/config"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// Placeholder stands for the Specter root in repo sources.
const Placeholder = "{{SPECTER_ROOT}}"

// ListFile is the repo-relative list of deployed files that carry the placeholder, shared with the shell installer.
const ListFile = "targets/rendered-files.txt"

// DefaultRootRel is the Specter root under HOME when none is saved.
const DefaultRootRel = "dev/specter"

// Render replaces Placeholder with root, in the home-relative form such as ~/dev/specter.
func Render(data []byte, root string) []byte {
	return bytes.ReplaceAll(data, []byte(Placeholder), []byte(root))
}

// Unrender replaces root in data with Placeholder, unless the next byte continues a path name.
func Unrender(data []byte, root string) []byte {
	needle := []byte(root)
	if len(needle) == 0 {
		return data
	}
	var out []byte
	rest := data
	for {
		i := bytes.Index(rest, needle)
		if i < 0 {
			break
		}
		end := i + len(needle)
		out = append(out, rest[:i]...)
		if end < len(rest) && IsNameByte(rest[end]) {
			out = append(out, rest[i:end]...)
		} else {
			out = append(out, Placeholder...)
		}
		rest = rest[end:]
	}
	return append(out, rest...)
}

// IsNameByte reports a byte that would continue a path element, so `~/dev/specter-old` is not `~/dev/specter`.
func IsNameByte(b byte) bool {
	return b == '-' || b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// RenderFor returns the transform for the files of one component target: identity for unlisted files.
// listed holds HOME-relative deployed paths; rel is relative to target, "." for a single-file component.
func RenderFor(listed []string, target, root string) plan.Transform {
	return selector(listed, target, func(data []byte) []byte { return Render(data, root) })
}

// UnrenderFor is the inverse selector of RenderFor.
func UnrenderFor(listed []string, target, root string) plan.Transform {
	return selector(listed, target, func(data []byte) []byte { return Unrender(data, root) })
}

func selector(listed []string, target string, apply func([]byte) []byte) plan.Transform {
	return func(rel string, data []byte) []byte {
		if slices.Contains(listed, path.Join(target, rel)) {
			return apply(data)
		}
		return data
	}
}

// ForComponent returns the render and unrender transforms of component target under env, or nil, nil
// when the list file is absent or unreadable or lists no file of target (nothing is rendered).
// The root is the saved one when valid (see ValidRoot), else HOME/dev/specter.
func ForComponent(env check.Env, target string) (renderFn, unrenderFn plan.Transform) {
	data, err := env.FS.ReadFile(filepath.Join(env.Repo, ListFile))
	if err != nil {
		return nil, nil
	}
	var listed []string
	for _, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			listed = append(listed, line)
		}
	}
	if !slices.ContainsFunc(listed, func(file string) bool { return file == target || strings.HasPrefix(file, target+"/") }) {
		return nil, nil
	}
	rootPath := env.Saved.SpecterRoot
	if !ValidRoot(env.Home, rootPath) {
		rootPath = filepath.Join(env.Home, DefaultRootRel)
	}
	root := config.PathLabel(env.Home, filepath.Clean(rootPath))
	return RenderFor(listed, target, root), UnrenderFor(listed, target, root)
}

// ValidRoot is the saved-root rule shared with the shell installer and the hook:
// absolute, strictly under home, no ".." segment ("..foo" is a plain name).
func ValidRoot(home, root string) bool {
	if !filepath.IsAbs(root) || slices.Contains(strings.Split(root, "/"), "..") {
		return false
	}
	rel, err := filepath.Rel(home, filepath.Clean(root))
	if err != nil || rel == "." {
		return false
	}
	first, _, _ := strings.Cut(rel, "/")
	return first != ".."
}
