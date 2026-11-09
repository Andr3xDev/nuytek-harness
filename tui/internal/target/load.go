package target

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

const (
	manifestExt = ".toml"
	targetsDir  = "targets"
)

// ManifestRel is the repo-relative path of a target's manifest, the one place that convention lives.
func ManifestRel(id string) string {
	return filepath.Join(targetsDir, id+manifestExt)
}

type LoadManifest func(path string, kinds []manifest.Kind) (manifest.Manifest, error)

// Load returns the registered specs first, then manifests found in targets/ that no spec claims.
func Load(repo string, specs []Spec, load LoadManifest, fsys check.FS) []Target {
	targets := make([]Target, 0, len(specs))
	registered := map[string]bool{}
	for _, spec := range specs {
		registered[spec.ID] = true
		targets = append(targets, loadTarget(repo, spec, load, fsys))
	}
	entries, err := fsys.ReadDir(filepath.Join(repo, targetsDir))
	if err != nil && !errors.Is(err, iofs.ErrNotExist) {
		// An unreadable targets dir would otherwise hide every manifest; show it as a failed target.
		targets = append(targets, Target{Spec: Planned(targetsDir, targetsDir), Err: fmt.Errorf("reading %s: %w", filepath.Join(repo, targetsDir), err)})
	}
	for _, e := range entries { // ReadDir is sorted by name
		id, isManifest := strings.CutSuffix(e.Name(), manifestExt)
		if isManifest && !registered[id] {
			targets = append(targets, Target{Spec: Planned(id, id)})
		}
	}
	return targets
}

func loadTarget(repo string, spec Spec, load LoadManifest, fsys check.FS) Target {
	t := Target{Spec: spec}
	if spec.Checks == nil {
		return t
	}
	path := filepath.Join(repo, ManifestRel(spec.ID))
	if _, err := fsys.Lstat(path); err != nil {
		t.Err = fmt.Errorf("manifest %s: %w", path, err)
		return t
	}
	kinds := make([]manifest.Kind, 0, len(spec.Evaluators))
	for k := range spec.Evaluators {
		kinds = append(kinds, k)
	}
	slices.Sort(kinds)
	m, err := load(path, kinds)
	if err != nil {
		t.Err = fmt.Errorf("manifest %s: %w", path, err)
		return t
	}
	t.Manifest = m
	if t.Checks, err = spec.Checks(m); err != nil {
		t.Err = fmt.Errorf("composing checks for %s: %w", spec.ID, err)
	}
	return t
}
