package state

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	stateDirMode  = 0o755
	stateFileMode = 0o644
)

type fileFormat struct {
	RepoPath string `toml:"repo_path"`
}

// Store is the only writer of files in the TUI.
type Store struct{ path string }

func NewStore(path string) *Store { return &Store{path: path} }

// Load returns defaults for a missing file; a corrupt file also yields a warning and is left untouched.
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
	return State{RepoPath: f.RepoPath}, ""
}

func (s *Store) warning(err error) string {
	return fmt.Sprintf("state file %s: fix or delete it (%v)", s.path, err)
}

// Save writes via a temp file in the same directory, then renames, so readers never see a partial file.
// It replaces a corrupt file that Load only warned about: the caller passes Save an explicit repo.
func (s *Store) Save(st State) error {
	data, err := toml.Marshal(fileFormat{RepoPath: st.RepoPath})
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

type DiscardStore struct{ real *Store }

// Discard wraps s for dry-run: Load reads the real store, Save writes nothing.
func Discard(s *Store) DiscardStore { return DiscardStore{real: s} }

func (d DiscardStore) Load() (State, string) { return d.real.Load() }

func (d DiscardStore) Save(State) error { return nil }
