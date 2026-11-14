package main

import (
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/dryrun"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/writer"
)

const (
	stdinFd  = 0
	stdoutFd = 1
)

func terminalGate(isTerminal func(fd int) bool) bool {
	stdinTerminal, stdoutTerminal := isTerminal(stdinFd), isTerminal(stdoutFd)
	return stdinTerminal && stdoutTerminal
}

func wireIO(dry bool, fileStore *state.Store, now func() time.Time) (plan.Runner, stateStore) {
	if dry {
		return dryrun.New(), state.Discard(fileStore)
	}
	return writer.New(now), fileStore
}
