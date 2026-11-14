package specter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const (
	defaultRootDisplay = "~/dev/specter"
	agentsRootNotice   = "only the TUI supports NUYTEK_SPECTER_ROOT; the rest of the harness uses " + defaultRootDisplay

	checkOpenspec = "specter.openspec"
	checkRepo     = "specter.repo"
	checkAgents   = "specter.agents"
	checkRepos    = "specter.repos"
)

// Checks derives every Specter check from facts. It does no IO.
func Checks(f Facts) []check.Result {
	results := []check.Result{openspecCheck(f)}
	if f.Repos.Err != nil {
		results = append(results, check.Result{CheckID: checkRepos, Status: check.StatusUnknown, Message: "repos.yaml unreadable: " + f.Repos.Err.Error()})
	} else {
		for _, id := range storeIDs(f.Repos) {
			results = append(results, storeCheck(f, id))
		}
		if f.RegisteredErr == nil {
			for _, s := range f.Registered {
				results = append(results, mappedCheck(f, s.ID))
			}
		}
		results = append(results, repoCheck(f))
	}
	return append(results, agentsCheck(f))
}

func storeIDs(repos ReposView) []string {
	ids := make([]string, 0, len(repos.Stores))
	for id := range repos.Stores {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func openspecCheck(f Facts) check.Result {
	if !f.Openspec.Found {
		return check.Result{
			CheckID:    checkOpenspec,
			Status:     check.StatusError,
			Message:    errOpenspecMissing.Error(),
			Suggestion: "install openspec and make it available on PATH",
		}
	}
	message := f.OpenspecVersion
	if message == "" {
		message = "found"
	}
	return check.Result{CheckID: checkOpenspec, Status: check.StatusOK, Message: message}
}

func storeCheck(f Facts, id string) check.Result {
	r := check.Result{CheckID: "specter.store." + id, Subject: id}
	switch {
	case !f.Openspec.Found:
		r.Status, r.Message = check.StatusSkipped, "needs openspec"
		return r
	case f.RegisteredErr != nil:
		r.Status, r.Message = check.StatusUnknown, "registry unknown: "+f.RegisteredErr.Error()
		return r
	}
	i := slices.IndexFunc(f.Registered, func(s StoreInfo) bool { return s.ID == id })
	if i < 0 {
		r.Status, r.Message = check.StatusWarning, fmt.Sprintf("store %s is not registered", id)
		r.Suggestion = "/specter setup " + id
		if j := slices.IndexFunc(f.Folders, func(sf StoreFolder) bool { return sf.ID == id }); j >= 0 {
			r.Suggestion = fmt.Sprintf("run Specter fast init (or openspec store register %s --id %s --yes)", f.Folders[j].Dir, id)
		}
		return r
	}
	if root := f.Registered[i].Root; pathMissing(f, root) {
		r.Status, r.Message = check.StatusWarning, fmt.Sprintf("store %s is registered at %s which does not exist", id, root)
		r.Suggestion = "/specter setup " + id
		return r
	}
	r.Status, r.Message = check.StatusOK, "registered at "+f.Registered[i].Root
	return r
}

func mappedCheck(f Facts, id string) check.Result {
	r := check.Result{CheckID: "specter.mapped." + id, Subject: id}
	if slices.ContainsFunc(f.Repos.Repos, func(e RepoEntry) bool { return e.Store == id }) {
		r.Status, r.Message = check.StatusOK, "has a mapped repo"
		return r
	}
	r.Status, r.Message = check.StatusWarning, fmt.Sprintf("store %s has no mapped repo", id)
	r.Suggestion = "/specter register <repo-path> " + id
	return r
}

func repoCheck(f Facts) check.Result {
	r := check.Result{CheckID: checkRepo}
	if !f.CurrentRepo.Found {
		r.Status, r.Message = check.StatusSkipped, "not inside a git repository"
		return r
	}
	r.Subject = f.CurrentRepo.Path
	if entry, ok := MatchRepo(f.Repos.Repos, f.CurrentRepo, f.Home); ok {
		r.Status, r.Message = check.StatusOK, "mapped to store "+entry.Store
		return r
	}
	r.Status, r.Message = check.StatusWarning, fmt.Sprintf("current repo %s is not mapped in repos.yaml", f.CurrentRepo.Path)
	r.Suggestion = fmt.Sprintf("run Specter fast init, or /specter register %s [store-id]", f.CurrentRepo.Path)
	return r
}

func agentsCheck(f Facts) check.Result {
	r := check.Result{CheckID: checkAgents}
	var found, stale []string
	for _, ref := range f.RootRefs {
		if !ref.Found {
			continue
		}
		found = append(found, ref.Label)
		if !ref.References {
			stale = append(stale, fmt.Sprintf("%s (%s copy)", ref.Label, ref.Copy))
		}
	}
	switch {
	case len(found) == 0:
		r.Status, r.Message = check.StatusUnknown, "no agent-facing copy found"
	case len(stale) == 0:
		r.Status, r.Message = check.StatusOK, "agent files reference "+f.Root.Path
	case f.Root.Source == RootSourceDefault:
		r.Status = check.StatusWarning
		r.Message = fmt.Sprintf("agent files do not reference the root %s: %s", f.Root.Path, strings.Join(stale, ", "))
	default:
		r.Status = check.StatusWarning
		r.Message = fmt.Sprintf("resolved root is %s but agent files reference %s: %s; %s",
			f.Root.Path, defaultRootDisplay, strings.Join(stale, ", "), agentsRootNotice)
	}
	return r
}

// pathMissing is true only for a definite "missing" answer from Load; an unchecked path is assumed present.
func pathMissing(f Facts, path string) bool {
	exists, checked := f.PathExists[path]
	return checked && !exists
}

// HealthSummary is the one-line state of checks plus drift for the dashboard.
type HealthSummary struct {
	Status      check.Status
	FailedCount int
	Text        string
}

// Summary counts warning and error results as failed; skipped never fails, any unknown keeps the status from reading ok.
func Summary(results []check.Result) HealthSummary {
	var failed, ok, unknown int
	status := check.StatusWarning
	for _, r := range results {
		switch r.Status {
		case check.StatusError:
			failed++
			status = check.StatusError
		case check.StatusWarning:
			failed++
		case check.StatusOK:
			ok++
		case check.StatusUnknown:
			unknown++
		}
	}
	switch {
	case failed > 0:
		return HealthSummary{Status: status, FailedCount: failed, Text: fmt.Sprintf("%d problem(s)", failed)}
	case ok > 0 && unknown == 0:
		return HealthSummary{Status: check.StatusOK, Text: "all checks passed"}
	}
	return HealthSummary{Status: check.StatusUnknown, Text: "state unknown"}
}
