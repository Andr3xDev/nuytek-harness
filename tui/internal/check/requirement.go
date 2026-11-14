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
	Unanalyzed             string // when set, reported as unknown with this message and no other check runs
}

func Requirements(id string, extract func(env Env) ([]Requirement, error)) Check {
	return Check{ID: id, Run: func(_ context.Context, env Env) []Result {
		reqs, err := extract(env)
		if err != nil {
			return []Result{{CheckID: id, Subject: id, Status: StatusUnknown, Message: err.Error()}}
		}
		results := make([]Result, 0, len(reqs))
		for _, req := range reqs {
			r := verifyRequirement(env, req)
			r.CheckID, r.Subject = id, req.Subject
			results = append(results, r)
		}
		return results
	}}
}

func verifyRequirement(env Env, req Requirement) Result {
	if req.Unanalyzed != "" {
		return Result{Status: StatusUnknown, Message: req.Unanalyzed}
	}
	if req.Command != "" {
		if _, err := env.Exec.LookPath(req.Command); err != nil {
			return Result{
				Status:     StatusError,
				Message:    fmt.Sprintf("command %s not found on PATH", req.Command),
				Suggestion: fmt.Sprintf("install %s or remove the reference", req.Command),
			}
		}
		return Result{Status: StatusOK}
	}
	info, err := env.FS.Lstat(req.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return Result{Status: StatusError, Message: fmt.Sprintf("%s not found", req.Path), Suggestion: "run install for the component that deploys it"}
	case err != nil:
		return Result{Status: StatusUnknown, Message: err.Error()}
	case req.MustExec && info.Mode()&OwnerExecBit == 0:
		return Result{Status: StatusError, Message: fmt.Sprintf("%s is not executable", req.Path)}
	}
	return Result{Status: StatusOK}
}
