package state

type State struct {
	RepoPath    string
	SpecterRoot string
	Targets     map[string]TargetState
}

type TargetState struct {
	BackupsDir string
}
