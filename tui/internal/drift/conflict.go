package drift

// ErrPathConflict reports an entry whose type blocks a copy; nothing is written.
type ErrPathConflict struct {
	Path  string
	Found string // type of the blocking entry: file, directory, symlink or special file; empty when unknown
}

func (e ErrPathConflict) Error() string { return "path conflict: " + e.Path }
