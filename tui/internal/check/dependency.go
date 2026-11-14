package check

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"
)

const DefaultProbeTimeout = 3 * time.Second

var versionPattern = regexp.MustCompile(`v?\d+(\.\d+)+`)

// versionArgs lists tools whose version flag is not --version.
var versionArgs = map[string][]string{"engram": {"version"}}

func Dependencies(id string, tools func() []string, timeout time.Duration) Check {
	return Check{ID: id, Run: func(ctx context.Context, env Env) []Result {
		names := tools()
		results := make([]Result, len(names))
		var wg sync.WaitGroup
		for i, name := range names {
			wg.Go(func() {
				// A panicking probe must not take down the other tools' results.
				defer func() {
					if p := recover(); p != nil {
						results[i] = Result{CheckID: id, Subject: name, Status: StatusUnknown, Message: fmt.Sprintf("probe panicked: %v", p)}
					}
				}()
				status, message := probeDependency(ctx, env.Exec, name, timeout)
				results[i] = Result{CheckID: id, Subject: name, Status: status, Message: message}
			})
		}
		wg.Wait()
		return results
	}}
}

func probeDependency(ctx context.Context, exec Exec, name string, timeout time.Duration) (Status, string) {
	if _, err := exec.LookPath(name); err != nil {
		return StatusError, fmt.Sprintf("%s not found on PATH", name)
	}
	args, ok := versionArgs[name]
	if !ok {
		args = []string{"--version"}
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, _ := exec.Output(probeCtx, name, args...)
	switch {
	case ctx.Err() != nil:
		return StatusUnknown, fmt.Sprintf("%s version probe cancelled", name)
	case errors.Is(probeCtx.Err(), context.DeadlineExceeded):
		return StatusWarning, fmt.Sprintf("%s version probe timed out after %s", name, timeout)
	}
	// A failed probe still counts as installed: only the version is unknown.
	version := versionPattern.Find(out)
	if version == nil {
		return StatusOK, "found, version unknown"
	}
	return StatusOK, "found, version " + string(version)
}
