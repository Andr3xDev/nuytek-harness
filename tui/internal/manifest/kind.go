package manifest

type Kind string

const (
	KindDir   Kind = "dir"
	KindDirX  Kind = "dirx"
	KindFile  Kind = "file"
	KindFileX Kind = "filex"
	// KindSpecter has no source or target: it is not deployed from the repo.
	KindSpecter Kind = "specter"
)

func (k Kind) Deploys() bool { return k != KindSpecter }

func (k Kind) IsDir() bool     { return k == KindDir || k == KindDirX }
func (k Kind) IsData() bool    { return k.IsDir() || k == KindFile || k == KindFileX }
func (k Kind) NeedsExec() bool { return k == KindDirX || k == KindFileX }
