package specter

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	storeConfigRel = "openspec/config.yaml"
	storeReadme    = "README.md"
	readmeMode     = 0o644
)

// PlanCreateStore sets up a new store under the root and cross-references it with every existing one.
func PlanCreateStore(ctx context.Context, env check.Env, id string) (plan.Plan, error) {
	if !storeIDPattern.MatchString(id) {
		return plan.Plan{}, fmt.Errorf("invalid store id %q: use lowercase letters, digits, '.', '_' or '-', starting with a letter or digit", id)
	}
	reg := LoadRegistration(ctx, env)
	switch {
	case reg.Root.OutsideHome:
		return plan.Plan{}, ErrBlocked{noteOutsideHome}
	case !reg.RootExists && reg.FoldersErr == nil:
		return plan.Plan{}, ErrBlocked{"specter root " + reg.Root.Path + " does not exist"}
	}
	if blocker := registerBlocker(reg); blocker != "" {
		return plan.Plan{}, ErrBlocked{blocker}
	}
	existing := CandidateStores(reg)
	if slices.Contains(existing, id) {
		return plan.Plan{}, fmt.Errorf("store id %q is already used", id)
	}

	dir := filepath.Join(reg.Root.Path, id)
	if _, err := env.FS.Lstat(dir); err == nil {
		return plan.Plan{}, ErrBlocked{"cannot create store " + id + ": " + dir + " already exists"}
	}
	for _, other := range existing {
		config := filepath.Join(storeDir(reg, other), storeConfigRel)
		if _, err := readCapped(env.FS, config, maxConfigBytes); err != nil {
			return plan.Plan{}, ErrBlocked{"cannot cross-reference store " + other + ": " + storeConfigRel + " unreadable: " + err.Error()}
		}
	}
	steps := []plan.Step{plan.Exec{
		Argv:    []string{openspecBinary, "store", "setup", id, "--path", dir, "--no-init-git"},
		Timeout: RegisterTimeout,
	}}
	if len(existing) > 0 {
		steps = append(steps, editStoreConfig(dir, "reference "+fmt.Sprint(existing), func(old []byte) ([]byte, error) {
			return SetReferences(old, existing)
		}))
	}
	for _, other := range existing {
		steps = append(steps, editStoreConfig(storeDir(reg, other), "reference "+id, func(old []byte) ([]byte, error) {
			return AddReference(old, id)
		}))
	}
	steps = append(steps, plan.Edit{
		Path: filepath.Join(dir, storeReadme),
		Edit: func(old []byte, exists bool) ([]byte, error) {
			if exists {
				return old, nil
			}
			return []byte("# " + id + "\n\nSpecter store `" + id + "`.\n"), nil
		},
		Rows:        []plan.Change{{Effect: plan.EffectCreate, Path: filepath.Join(dir, storeReadme)}},
		NewFileMode: readmeMode,
	})
	return plan.New("Create store "+id, nil, steps), nil
}

// storeDir prefers the registered root, which is where openspec reads the config from.
func storeDir(reg Registration, id string) string {
	if root, ok := registeredAt(reg, id); ok {
		return root
	}
	i := slices.IndexFunc(reg.Folders, func(f StoreFolder) bool { return f.ID == id })
	return reg.Folders[i].Dir
}

func editStoreConfig(dir, note string, edit func(old []byte) ([]byte, error)) plan.Edit {
	path := filepath.Join(dir, storeConfigRel)
	return plan.Edit{
		Path: path,
		Edit: func(old []byte, exists bool) ([]byte, error) {
			if !exists {
				return nil, errors.New(storeConfigRel + " not found")
			}
			return edit(old)
		},
		Rows: []plan.Change{{Effect: plan.EffectOverwrite, Path: path, Note: note}},
	}
}
