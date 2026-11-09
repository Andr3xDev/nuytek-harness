package check

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
)

// OwnerExecBit is the mode bit the installer sets for executables.
const OwnerExecBit fs.FileMode = 0o100

// Requirement is a path that must exist (and be executable when MustExec) or a command that must resolve on PATH.
type Requirement struct {
	Subject, Path, Command string
	MustExec               bool
}

func Requirements(id string, extract func(env Env) ([]Requirement, error)) Check {
	return Check{ID: id, Run: func(_ context.Context, env Env) []Result {
		reqs, err := extract(env)
		if err != nil {
			return []Result{{CheckID: id, Subject: id, Status: StatusUnknown, Message: err.Error()}}
		}
		results := make([]Result, 0, len(reqs))
		for _, req := range reqs {
			status, message := verifyRequirement(env, req)
			results = append(results, Result{CheckID: id, Subject: req.Subject, Status: status, Message: message})
		}
		return results
	}}
}

func verifyRequirement(env Env, req Requirement) (Status, string) {
	if req.Command != "" {
		if _, err := env.Exec.LookPath(req.Command); err != nil {
			return StatusError, fmt.Sprintf("command %s not found on PATH", req.Command)
		}
		return StatusOK, ""
	}
	info, err := env.FS.Lstat(req.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return StatusError, fmt.Sprintf("%s not found", req.Path)
	case err != nil:
		return StatusUnknown, err.Error()
	case req.MustExec && info.Mode()&OwnerExecBit == 0:
		return StatusError, fmt.Sprintf("%s is not executable", req.Path)
	}
	return StatusOK, ""
}
