package main

import "github.com/Andr3xDev/nuytek-harness/tui/internal/state"

// saveRepoPath persists path on top of the loaded state so the --repo save keeps the other saved config.
func saveRepoPath(store stateStore, saved state.State, path string) error {
	saved.RepoPath = path
	return store.Save(saved)
}
