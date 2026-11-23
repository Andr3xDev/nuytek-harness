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

// Both plans are snapshot-free and carry an empty Confirm: they only register stores and add one repos.yaml entry.

// PlanFastInit creates the root from the template when it is missing; otherwise it registers every unregistered
// store folder and, when mapTo names a store, maps the current repo to it. Unmet preconditions return ErrBlocked.
func PlanFastInit(ctx context.Context, env check.Env, mapTo string) (plan.Plan, error) {
	reg := LoadRegistration(ctx, env)
	switch {
	case reg.Root.OutsideHome:
		return plan.Plan{}, ErrBlocked{noteOutsideHome}
	case !reg.RootExists && reg.FoldersErr == nil:
		return planTemplateInit(ctx, env, reg.Root)
	case !reg.Openspec.Found:
		return plan.Plan{}, ErrBlocked{errOpenspecMissing.Error()}
	}
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
	return addMappingStep(path, repoEntryFor(cur, env.Home, mapTo), "map the current repo to store "+mapTo), ""
}

func addMappingStep(path string, entry RepoEntry, note string) plan.Edit {
	return reposEditStep(path, note, func(old []byte) ([]byte, error) { return AddRepoMapping(old, entry) })
}

func reposEditStep(path, note string, edit func(old []byte) ([]byte, error)) plan.Edit {
	return plan.Edit{
		Path: path,
		Edit: func(old []byte, exists bool) ([]byte, error) {
			if !exists {
				return nil, errors.New(reposFile + " not found")
			}
			return edit(old)
		},
		Rows: []plan.Change{{Effect: plan.EffectOverwrite, Path: path, Note: note}},
	}
}

// mappingRegistration loads the registration and blocks when mappings cannot be edited.
func mappingRegistration(ctx context.Context, env check.Env) (Registration, error) {
	reg := LoadRegistration(ctx, env)
	if blocker := registerBlocker(reg); blocker != "" {
		return reg, ErrBlocked{blocker}
	}
	return reg, nil
}

// matchedMapping resolves entry against repos.yaml the way MatchRepo does and returns the file's own entry,
// so a mapping that the edit could not find is refused at plan time.
func matchedMapping(env check.Env, reg Registration, entry RepoEntry) (RepoEntry, error) {
	path := filepath.Join(reg.Root.Path, reposFile)
	data, err := readCapped(env.FS, path, maxConfigBytes)
	if err != nil {
		return RepoEntry{}, ErrBlocked{reposFile + " unreadable: " + err.Error()}
	}
	view := ParseRepos(data)
	if view.Err != nil {
		return RepoEntry{}, ErrBlocked{reposFile + " invalid: " + view.Err.Error()}
	}
	cur := CurrentRepo{Found: true, Remote: entry.Remote}
	if entry.Path != "" {
		cur.Path = entryPath(entry, env.Home)
	}
	matched, found := MatchRepo(view.Repos, cur, env.Home)
	if !found {
		return RepoEntry{}, ErrBlocked{"repo mapping not found in " + reposFile}
	}
	if _, _, _, _, err := findRepoEntry(data, matched); errors.Is(err, errReposFlowList) {
		return RepoEntry{}, ErrBlocked{err.Error()}
	}
	return matched, nil
}

func requireRegistered(reg Registration, store string) error {
	if len(reg.Registered) == 0 {
		return ErrBlocked{"no registered store to map to"}
	}
	if _, registered := registeredAt(reg, store); !registered {
		return fmt.Errorf("store %q is not registered", store)
	}
	return nil
}

// PlanAddMapping plans one repos.yaml edit that maps entry; no registered store blocks it.
func PlanAddMapping(ctx context.Context, env check.Env, entry RepoEntry) (plan.Plan, error) {
	reg, err := mappingRegistration(ctx, env)
	if err != nil {
		return plan.Plan{}, err
	}
	if err := requireRegistered(reg, entry.Store); err != nil {
		return plan.Plan{}, err
	}
	step := addMappingStep(filepath.Join(reg.Root.Path, reposFile), entry, "map to store "+entry.Store)
	return plan.New("Add repo mapping", nil, []plan.Step{step}), nil
}

// PlanChangeMapping plans one repos.yaml edit that points the mapping matched by entry's remote or path at store.
func PlanChangeMapping(ctx context.Context, env check.Env, entry RepoEntry, store string) (plan.Plan, error) {
	reg, err := mappingRegistration(ctx, env)
	if err != nil {
		return plan.Plan{}, err
	}
	if err := requireRegistered(reg, store); err != nil {
		return plan.Plan{}, err
	}
	matched, err := matchedMapping(env, reg, entry)
	if err != nil {
		return plan.Plan{}, err
	}
	step := reposEditStep(filepath.Join(reg.Root.Path, reposFile), "point the mapping to store "+store,
		func(old []byte) ([]byte, error) { return ChangeRepoStore(old, matched, store) })
	return plan.New("Change repo mapping", nil, []plan.Step{step}), nil
}

// PlanRemoveMapping plans one repos.yaml edit that drops the mapping matched by entry's remote or path.
func PlanRemoveMapping(ctx context.Context, env check.Env, entry RepoEntry) (plan.Plan, error) {
	reg, err := mappingRegistration(ctx, env)
	if err != nil {
		return plan.Plan{}, err
	}
	matched, err := matchedMapping(env, reg, entry)
	if err != nil {
		return plan.Plan{}, err
	}
	step := reposEditStep(filepath.Join(reg.Root.Path, reposFile), "remove the repo mapping",
		func(old []byte) ([]byte, error) { return RemoveRepoMapping(old, matched) })
	return plan.New("Remove repo mapping", nil, []plan.Step{step}), nil
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
