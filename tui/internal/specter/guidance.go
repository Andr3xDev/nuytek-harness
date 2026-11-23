package specter

// FastInitBlocker is the first unmet create-from-template condition that the loaded facts can show, or "".
// It runs nothing: the parent directory and the name clash are only checked when the plan is built.
// It mirrors templateOwner's checks.
func FastInitBlocker(f Facts) string {
	switch {
	case f.Root.OutsideHome:
		return noteOutsideHome
	case !f.Git.Found:
		return gitBinary + " not found on PATH"
	case !f.Openspec.Found:
		return openspecBinary + " not found on PATH"
	case !f.Gh.Found:
		return ghBinary + " not found on PATH"
	case !f.Gh.Authenticated:
		return "gh not authenticated; run gh auth login"
	}
	return ""
}
