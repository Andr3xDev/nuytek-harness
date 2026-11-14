package action

import (
	"fmt"
	"slices"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

var storeNames = map[backup.StoreKind]string{backup.StoreHome: "home", backup.StoreRepo: "repo"}

func storeDirs(t target.Target, env check.Env) []backup.Dir {
	if t.Spec.BackupDirs == nil {
		return nil
	}
	return t.Spec.BackupDirs(env.Home, env.Repo)
}

// storeOf returns the target's backup store of the kind.
func storeOf(t target.Target, env check.Env, kind backup.StoreKind) (*backup.Dir, error) {
	for _, dir := range storeDirs(t, env) {
		if dir.Kind == kind {
			return &dir, nil
		}
	}
	return nil, fmt.Errorf("target %s has no %s backup store", t.Spec.ID, storeNames[kind])
}

// inStores reports whether root is the path of one of the target's backup stores.
func inStores(t target.Target, env check.Env, root string) bool {
	return slices.ContainsFunc(storeDirs(t, env), func(dir backup.Dir) bool { return dir.Path == root })
}
