package specter

import (
	"context"
	"errors"
	"path/filepath"
	"slices"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// Both plans are snapshot-free and carry an empty Confirm: they only register stores and add one repos.yaml entry.

// PlanFastInit registers every unregistered store folder and, when mapTo names a store, maps the current repo to it.
func PlanFastInit(ctx context.Context, env check.Env, mapTo string) (plan.Plan, error) {
	reg := LoadRegistration(ctx, env)
	part := registerPart(reg)
	if registerBlocker(reg) == "" {
		if step, skip := mappingStep(env, reg, mapTo); step != nil {
			part.Steps = append(part.Steps, step)
		} else if skip != "" {
			part.Skipped = append(part.Skipped, plan.Skip{Name: reposFile, Reason: skip})
		}
	}
	return assemble("Specter fast init", part), nil
}

// PlanMaintenance registers unregistered store folders; other drift stays report-only.
func PlanMaintenance(ctx context.Context, env check.Env) (plan.Plan, error) {
	return assemble("Specter maintenance", registerPart(LoadRegistration(ctx, env))), nil
}

func assemble(title string, part plan.Part) plan.Plan {
	p := plan.New(title, nil, part.Steps)
	p.Skipped, p.Notes = part.Skipped, part.Notes
	return p
}

// CandidateStores lists the ids a repo can be mapped to: registered ones plus those fast init would register.
func CandidateStores(reg Registration) []string {
	var ids []string
	for _, s := range reg.Registered {
		ids = append(ids, s.ID)
	}
	for _, f := range reg.Folders {
		if f.ID != "" {
			ids = append(ids, f.ID)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

// mappingStep returns the repos.yaml edit, or a reason when none is planned; (nil, "") means the caller chose none.
func mappingStep(env check.Env, reg Registration, mapTo string) (plan.Step, string) {
	cur := DetectRepo(env)
	if !cur.Found {
		return nil, "not in a git repository, so no repo is mapped"
	}
	path := filepath.Join(reg.Root.Path, reposFile)
	data, err := readCapped(env.FS, path, maxConfigBytes)
	if err != nil {
		if isMissing(err) {
			return nil, reposFile + " not found, so no repo is mapped"
		}
		return nil, reposFile + " unreadable: " + err.Error()
	}
	view := ParseRepos(data)
	if view.Err != nil {
		return nil, reposFile + " invalid: " + view.Err.Error()
	}
	if entry, mapped := MatchRepo(view.Repos, cur, env.Home); mapped {
		return nil, "repo already mapped to store " + entry.Store
	}
	candidates := CandidateStores(reg)
	if mapTo == "" {
		if len(candidates) == 0 {
			return nil, "no registered store to choose"
		}
		return nil, ""
	}
	if !slices.Contains(candidates, mapTo) {
		return nil, "store " + mapTo + " is not registered or found under the specter root"
	}
	entry := repoEntryFor(cur, env.Home, mapTo)
	return plan.Edit{
		Path: path,
		Edit: func(old []byte, exists bool) ([]byte, error) {
			if !exists {
				return nil, errors.New(reposFile + " not found")
			}
			return AddRepoMapping(old, entry)
		},
		Rows: []plan.Change{{Effect: plan.EffectOverwrite, Path: path, Note: "map the current repo to store " + mapTo}},
	}, ""
}

// repoEntryFor prefers the origin remote; without one it maps by path, `~`-relative under HOME.
func repoEntryFor(cur CurrentRepo, home, store string) RepoEntry {
	if cur.Remote != "" {
		return RepoEntry{Remote: cur.Remote, Store: store}
	}
	if rel, err := filepath.Rel(home, cur.Path); err == nil && !outsideHome(home, cur.Path) && rel != "." {
		return RepoEntry{Path: "~/" + filepath.ToSlash(rel), Store: store}
	}
	return RepoEntry{Path: cur.Path, Store: store}
}
