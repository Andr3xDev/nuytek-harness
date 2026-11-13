package specter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const driftPrefix = "drift."

// Drift reports inconsistencies between registration, folders and repos.yaml. It never changes anything.
func Drift(f Facts) []check.Result {
	var results []check.Result
	if f.RegisteredErr == nil {
		results = append(results, registeredElsewhere(f)...)
		results = append(results, danglingReferences(f)...)
	}
	return append(results, stalePaths(f)...)
}

func driftResult(kind, subject, message, suggestion string) check.Result {
	return check.Result{CheckID: driftPrefix + kind + "." + subject, Subject: subject, Status: check.StatusWarning, Message: message, Suggestion: suggestion}
}

func registeredElsewhere(f Facts) []check.Result {
	var results []check.Result
	for _, folder := range f.Folders {
		i := slices.IndexFunc(f.Registered, func(s StoreInfo) bool { return s.ID == folder.ID })
		if folder.ID == "" || i < 0 || sameDir(f.Registered[i].Root, folder.Dir) {
			continue
		}
		results = append(results, driftResult("root", folder.ID,
			fmt.Sprintf("store %s is registered at %s but its folder is %s", folder.ID, f.Registered[i].Root, folder.Dir),
			fmt.Sprintf("/specter unregister %s, then /specter register %s", folder.ID, folder.Dir)))
	}
	return results
}

func danglingReferences(f Facts) []check.Result {
	registered := make([]string, len(f.Registered))
	for i, s := range f.Registered {
		registered[i] = s.ID
	}
	stores := make([]string, 0, len(f.References))
	for id := range f.References {
		stores = append(stores, id)
	}
	slices.Sort(stores)

	var results []check.Result
	for _, store := range stores {
		for _, ref := range f.References[store] {
			if slices.Contains(registered, ref) {
				continue
			}
			results = append(results, driftResult("reference", store+"."+ref,
				fmt.Sprintf("store %s references %s, which is not registered", store, ref),
				fmt.Sprintf("/specter references %s %s", store, strings.Join(registered, " "))))
		}
	}
	return results
}

func stalePaths(f Facts) []check.Result {
	var results []check.Result
	for _, entry := range f.Repos.Repos {
		if entry.Path == "" {
			continue
		}
		if p := entryPath(entry, f.Home); pathMissing(f, p) {
			results = append(results, driftResult("path", p,
				fmt.Sprintf("repos.yaml path %s does not exist", p),
				fmt.Sprintf("/specter register <repo-path> %s, after removing the entry", entry.Store)))
		}
	}
	return results
}
