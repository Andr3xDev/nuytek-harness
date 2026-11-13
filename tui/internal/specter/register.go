package specter

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	repoEnvVar         = "NUYTEK_SPECTER_REPO"
	repoURLPlaceholder = "<your-specter-repo-url>"
	repoEnvVarHint     = " (set " + repoEnvVar + " to fill it in)"
)

// CloneCommand is shown as text when the root is missing; the TUI never runs git.
func CloneCommand(lookupEnv func(string) (string, bool), root Root) string {
	url, hint := repoURLPlaceholder, repoEnvVarHint
	if v, ok := lookupEnv(repoEnvVar); ok && v != "" {
		url, hint = v, ""
	}
	return "git clone " + url + " " + root.Path + hint
}

const (
	noteOutsideHome = "root outside HOME is not supported"
	noteNoID        = "no id in store.yaml"
)

// registerBlocker is the first unmet precondition, or "" when registration can be planned.
func registerBlocker(reg Registration) string {
	switch {
	case reg.Root.OutsideHome:
		return noteOutsideHome
	case !reg.RootExists:
		return "specter root " + reg.Root.Path + " not found; clone it with: " + reg.CloneCommand
	case !reg.Openspec.Found:
		return errOpenspecMissing.Error()
	case reg.RegisteredErr != nil:
		return "registry unknown: " + reg.RegisteredErr.Error()
	}
	return ""
}

// registerPart is the only place that decides registrations: it never unregisters and never moves a store.
func registerPart(reg Registration) plan.Part {
	if note := registerBlocker(reg); note != "" {
		return plan.Part{Notes: []string{note}}
	}
	var part plan.Part
	for _, folder := range reg.Folders {
		if folder.ID == "" {
			reason := noteNoID
			if folder.BadID != "" {
				reason = fmt.Sprintf("invalid store id %q in store.yaml", folder.BadID)
			}
			part.Skipped = append(part.Skipped, plan.Skip{Name: folder.Dir, Reason: reason})
			continue
		}
		root, registered := registeredAt(reg, folder.ID)
		switch {
		case !registered:
			part.Steps = append(part.Steps, plan.Exec{
				Argv:    []string{openspecBinary, "store", "register", folder.Dir, "--id", folder.ID, "--yes"},
				Timeout: RegisterTimeout,
			})
		case !sameDir(root, folder.Dir):
			part.Skipped = append(part.Skipped, plan.Skip{Name: folder.ID, Reason: "registered at " + root})
		}
	}
	return part
}

func registeredAt(reg Registration, id string) (root string, ok bool) {
	i := slices.IndexFunc(reg.Registered, func(s StoreInfo) bool { return s.ID == id })
	if i < 0 {
		return "", false
	}
	return reg.Registered[i].Root, true
}

// sameDir compares cleaned paths only; a symlinked root or store folder reads as a different directory.
func sameDir(a, b string) bool { return filepath.Clean(a) == filepath.Clean(b) }
