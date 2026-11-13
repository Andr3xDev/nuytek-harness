package specter

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const (
	reposFile     = "repos.yaml"
	storeListKeys = 3 // active, archived, specs
)

var errRelativeRoot = errors.New("root is not an absolute path")

// ChangeInfo is one active change; LastModified is the raw ISO value, parsed by ParseLastModified.
type ChangeInfo struct {
	Name         string `json:"name"`
	LastModified string `json:"lastModified"`
}

// StoreStats counts one registered store; Err is set when any of its list calls failed.
type StoreStats struct {
	Active, Archived, Specs int
	Changes                 []ChangeInfo // active changes only
	Err                     error
}

// ParseLastModified reads the RFC3339 value (fractional seconds allowed) of `openspec list --json`.
func ParseLastModified(value string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339, value)
	return t, err == nil
}

// Facts is everything the screen and the dashboard line need; checks and views are pure functions of it.
type Facts struct {
	Registration
	OpenspecVersion string
	Repos           ReposView
	Stores          map[string]StoreStats
	References      map[string][]string // store id -> `references:` of its openspec/config.yaml
	CurrentRepo     CurrentRepo
	RootRefs        []RootRef
	Home            string // expands `~` in repos.yaml paths
	// PathExists holds only definite answers (Lstat found it or reported it missing) for repos.yaml `path:` entries
	// and registered store roots, so Checks and Drift stay pure; an absent key means "not checked".
	PathExists map[string]bool
}

// Load does all the IO once. It never runs git and runs openspec only when it is on PATH.
func Load(ctx context.Context, env check.Env) Facts {
	reg := LoadRegistration(ctx, env)
	facts := Facts{
		Registration: reg,
		Repos:        loadRepos(env.FS, reg.Root),
		CurrentRepo:  DetectRepo(env),
		RootRefs:     LoadRootRefs(env, reg.Root),
		References:   map[string][]string{},
		Home:         env.Home,
	}
	facts.PathExists = probePaths(env.FS, facts)
	if !reg.Openspec.Found {
		return facts
	}
	facts.OpenspecVersion = probeVersion(ctx, env)
	facts.Stores = loadStoreStats(ctx, env.Exec, reg.Registered)
	for _, s := range reg.Registered {
		if refs := readReferences(env.FS, s.Root); len(refs) > 0 {
			facts.References[s.ID] = refs
		}
	}
	return facts
}

func loadRepos(fsys check.FS, root Root) ReposView {
	if root.Relative {
		return ReposView{Err: errRelativeRoot}
	}
	data, err := readCapped(fsys, filepath.Join(root.Path, reposFile), maxConfigBytes)
	if err != nil {
		return ReposView{Err: err}
	}
	return ParseRepos(data)
}

func probePaths(fsys check.FS, f Facts) map[string]bool {
	paths := make([]string, 0, len(f.Registered)+len(f.Repos.Repos))
	for _, s := range f.Registered {
		paths = append(paths, s.Root)
	}
	for _, e := range f.Repos.Repos {
		if e.Path != "" {
			paths = append(paths, entryPath(e, f.Home))
		}
	}
	exists := make(map[string]bool, len(paths))
	for _, p := range paths {
		_, err := fsys.Lstat(p)
		switch {
		case err == nil:
			exists[p] = true
		case isMissing(err):
			exists[p] = false
		}
	}
	return exists
}

func probeVersion(ctx context.Context, env check.Env) string {
	res := check.ProbeResult(ctx, env.Exec, checkOpenspec, openspecBinary, check.DefaultProbeTimeout)
	if res.Status != check.StatusOK {
		return ""
	}
	return res.Message
}

func readReferences(fsys check.FS, storeRoot string) []string {
	data, err := readCapped(fsys, filepath.Join(storeRoot, "openspec", "config.yaml"), maxConfigBytes)
	if err != nil {
		return nil
	}
	var cfg struct {
		References []string `yaml:"references"`
	}
	if yaml.Unmarshal(data, &cfg) != nil {
		return nil
	}
	return cfg.References
}

// loadStoreStats runs the three list calls of every store, at most MaxConcurrentOpenspec at a time.
func loadStoreStats(ctx context.Context, exec check.Exec, stores []StoreInfo) map[string]StoreStats {
	type listing struct {
		Changes []ChangeInfo `json:"changes"`
		Specs   []struct{}   `json:"specs"`
	}
	results := make([][storeListKeys]struct {
		list listing
		err  error
	}, len(stores))
	queries := [storeListKeys][]string{
		{"--json"},
		{"--json", "--archived"},
		{"--specs", "--json"},
	}
	slots := make(chan struct{}, MaxConcurrentOpenspec)
	var wg sync.WaitGroup
	for i, s := range stores {
		for q, flags := range queries {
			wg.Go(func() {
				select {
				case slots <- struct{}{}:
					defer func() { <-slots }()
				case <-ctx.Done():
				}
				// select picks randomly when both cases are ready, so re-check after it.
				if err := ctx.Err(); err != nil {
					results[i][q].err = err
					return
				}
				callCtx, cancel := context.WithTimeout(ctx, OpenspecQueryTimeout)
				defer cancel()
				args := append([]string{"list", "--store", s.ID}, flags...)
				results[i][q].err = openspecJSON(callCtx, exec, &results[i][q].list, false, args...)
			})
		}
	}
	wg.Wait()

	stats := make(map[string]StoreStats, len(stores))
	for i, s := range stores {
		r := results[i]
		stat := StoreStats{Active: len(r[0].list.Changes), Changes: r[0].list.Changes, Archived: len(r[1].list.Changes), Specs: len(r[2].list.Specs)}
		stat.Err = errors.Join(r[0].err, r[1].err, r[2].err)
		stats[s.ID] = stat
	}
	return stats
}
