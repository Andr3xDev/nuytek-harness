package manifest

type Kind string

const (
	KindDir   Kind = "dir"
	KindDirX  Kind = "dirx"
	KindFile  Kind = "file"
	KindFileX Kind = "filex"
)

func (k Kind) IsDir() bool     { return k == KindDir || k == KindDirX }
func (k Kind) NeedsExec() bool { return k == KindDirX || k == KindFileX }
