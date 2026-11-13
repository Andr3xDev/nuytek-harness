package specter

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"syscall"

	"go.yaml.in/yaml/v3"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const (
	openspecBinary    = "openspec"
	maxStoreYAMLBytes = 64 << 10
)

var (
	errOpenspecMissing = errors.New("openspec not found on PATH")
	storeIDPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
)

type ToolInfo struct {
	Found bool
	Err   error
}

type StoreInfo struct {
	ID   string `json:"id"`
	Root string `json:"root"`
}

// StoreFolder is a directory under the root that holds a store; ID is empty when store.yaml has none
// or an unsafe one (kept in BadID), since the id reaches an argv and the tracked repos.yaml.
type StoreFolder struct {
	Dir   string
	ID    string
	BadID string
}

type Registration struct {
	Root          Root
	CloneCommand  string
	RootExists    bool
	Openspec      ToolInfo
	Registered    []StoreInfo
	RegisteredErr error // set when the registry is unknown
	Folders       []StoreFolder
	FoldersErr    error // set when the root or a store.yaml could not be read, so Folders may be incomplete
}

// LoadRegistration makes one `openspec store list --json` call and never runs git.
func LoadRegistration(ctx context.Context, env check.Env) Registration {
	reg := Registration{Root: ResolveRoot(env.LookupEnv, env.Home)}
	reg.CloneCommand = CloneCommand(env.LookupEnv, reg.Root)
	reg.RootExists, reg.Folders, reg.FoldersErr = scanFolders(env.FS, reg.Root)

	if _, err := env.Exec.LookPath(openspecBinary); err != nil {
		reg.Openspec = ToolInfo{Err: err}
		reg.RegisteredErr = errOpenspecMissing
		return reg
	}
	reg.Openspec.Found = true

	var list struct {
		Stores []StoreInfo `json:"stores"`
	}
	ctx, cancel := context.WithTimeout(ctx, OpenspecQueryTimeout)
	defer cancel()
	if err := openspecJSON(ctx, env.Exec, &list, false, "store", "list", "--json"); err != nil {
		reg.RegisteredErr = err
		return reg
	}
	reg.Registered = list.Stores
	return reg
}

// isMissing separates "not there" from "cannot tell"; only the first may be reported as absent.
func isMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

func scanFolders(fsys check.FS, root Root) (exists bool, folders []StoreFolder, firstErr error) {
	if root.Relative {
		return false, nil, nil
	}
	entries, err := fsys.ReadDir(root.Path)
	if err != nil {
		if isMissing(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root.Path, e.Name())
		id, found, err := readStoreID(fsys, filepath.Join(dir, ".openspec-store", "store.yaml"))
		if err != nil && firstErr == nil {
			firstErr = err
		}
		if !found {
			continue
		}
		folder := StoreFolder{Dir: dir, ID: id}
		if id != "" && !storeIDPattern.MatchString(id) {
			folder.ID, folder.BadID = "", id
		}
		folders = append(folders, folder)
	}
	return true, folders, firstErr
}

// readStoreID reports found=false for an absent, oversized or non-regular store.yaml; err only for unreadable ones.
func readStoreID(fsys check.FS, path string) (id string, found bool, err error) {
	data, err := readCapped(fsys, path, maxStoreYAMLBytes)
	if err != nil {
		if isMissing(err) || errors.Is(err, errNotRegular) || errors.Is(err, errTooLarge) {
			return "", false, nil
		}
		return "", false, err
	}
	var meta struct {
		ID string `yaml:"id"`
	}
	_ = yaml.Unmarshal(data, &meta) // an unparsable store.yaml has no usable id, same as a missing one
	return meta.ID, true, nil
}
