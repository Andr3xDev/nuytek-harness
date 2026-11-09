package check

import "context"

type Check struct {
	ID  string
	Run func(ctx context.Context, env Env) []Result
}

// DependenciesID tags the check results that dashboard.Compute turns into missing dependencies.
const DependenciesID = "dependencies"
