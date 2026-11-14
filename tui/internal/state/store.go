package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	stateDirMode  = 0o755
	stateFileMode = 0o644
	corruptSuffix = ".corrupt"
	// Nanoseconds keep two corrupt files moved aside in quick succession apart.
	corruptStampLayout = "20060102T150405.000000000"
)

const (
	keyRepoPath    = "repo_path"
	keySpecterRoot = "specter_root"
	keyTargets     = "targets"
	keyBackupsDir  = "backups_dir"
)

type fileFormat struct {
	RepoPath    string                      `toml:"repo_path"`
	SpecterRoot string                      `toml:"specter_root"`
	Targets     map[string]targetFileFormat `toml:"targets"`
}

type targetFileFormat struct {
	BackupsDir string `toml:"backups_dir"`
}

// Store is the only writer of files in the TUI.
type Store struct{ path string }

func NewStore(path string) *Store { return &Store{path: path} }

// Load returns defaults for a missing file; a corrupt file also yields a warning and is left untouched
// until the next Save moves it aside.
func (s *Store) Load() (State, string) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, ""
	}
	if err != nil {
		return State{}, s.warning(err)
	}
	var f fileFormat
	if err := toml.Unmarshal(data, &f); err != nil {
		return State{}, s.warning(err)
	}
	st := State{RepoPath: f.RepoPath, SpecterRoot: f.SpecterRoot}
	for id, t := range f.Targets {
		if st.Targets == nil {
			st.Targets = map[string]TargetState{}
		}
		st.Targets[id] = TargetState{BackupsDir: t.BackupsDir}
	}
	return st, ""
}

func (s *Store) warning(err error) string {
	return fmt.Sprintf("state file %s: fix or delete it (%v)", s.path, err)
}

// Save writes via a temp file in the same directory, then renames, so readers never see a partial file.
// It merges into the existing file so keys it does not know survive; an empty value removes its key, and a
// target table left with no keys is dropped. A corrupt file is renamed to state.toml.corrupt-<timestamp>, then replaced.
func (s *Store) Save(st State) error {
	doc, err := s.readDoc()
	if err != nil {
		return err
	}
	setOrDelete(doc, keyRepoPath, st.RepoPath)
	setOrDelete(doc, keySpecterRoot, st.SpecterRoot)
	targets, _ := doc[keyTargets].(map[string]any)
	if targets == nil {
		targets = map[string]any{}
	}
	for id, t := range st.Targets {
		table, _ := targets[id].(map[string]any)
		if table == nil {
			table = map[string]any{}
		}
		setOrDelete(table, keyBackupsDir, t.BackupsDir)
		if len(table) == 0 {
			delete(targets, id)
		} else {
			targets[id] = table
		}
	}
	if len(targets) == 0 {
		delete(doc, keyTargets)
	} else {
		doc[keyTargets] = targets
	}
	data, err := toml.Marshal(doc)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, stateDirMode); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(stateFileMode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}

// readDoc returns the file as a generic map; a missing file is an empty document and any other read error
// aborts. A corrupt file is moved aside to a timestamped corruptSuffix name first, so replacing it never
// destroys what the user wrote, nor an earlier backup.
func (s *Store) readDoc() (map[string]any, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	doc := map[string]any{}
	if toml.Unmarshal(data, &doc) == nil {
		return doc, nil
	}
	aside := s.path + corruptSuffix + "-" + time.Now().UTC().Format(corruptStampLayout)
	if err := os.Rename(s.path, aside); err != nil {
		return nil, fmt.Errorf("keep corrupt state file: %w", err)
	}
	return map[string]any{}, nil
}

func setOrDelete(m map[string]any, key, value string) {
	if value == "" {
		delete(m, key)
		return
	}
	m[key] = value
}

type DiscardStore struct{ real *Store }

// Discard wraps s for dry-run: Load reads the real store, Save writes nothing.
func Discard(s *Store) DiscardStore { return DiscardStore{real: s} }

func (d DiscardStore) Load() (State, string) { return d.real.Load() }

func (d DiscardStore) Save(State) error { return nil }
