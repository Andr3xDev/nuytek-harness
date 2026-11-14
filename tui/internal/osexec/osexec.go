package osexec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

var _ check.Exec = Exec{}

// MaxOutputBytes caps captured output; version probes need a few lines.
const MaxOutputBytes = 64 * 1024

// MaxRunOutputBytes caps each stream of Run; a validate report of a large store exceeds MaxOutputBytes.
const MaxRunOutputBytes = 1 << 20

const killWaitDelay = 500 * time.Millisecond

type Exec struct{}

func (Exec) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (Exec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	buf := cappedBuffer{max: MaxOutputBytes}
	cmd := newCommand(ctx, name, args)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.Bytes(), err
}

// Run reports a non-zero exit as ExitCode with a nil error. A signal kill not caused by ctx yields ExitCode -1 with a nil error; exec.ErrWaitDelay surfaces as in Output.
func (Exec) Run(ctx context.Context, name string, args ...string) (check.ExecResult, error) {
	stdout, stderr := cappedBuffer{max: MaxRunOutputBytes}, cappedBuffer{max: MaxRunOutputBytes}
	cmd := newCommand(ctx, name, args)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := check.ExecResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var exitErr *exec.ExitError
	// A kill from ctx also surfaces as ExitError; that is a cancellation, not an exit code.
	if errors.As(err, &exitErr) && ctx.Err() == nil {
		res.ExitCode, err = exitErr.ExitCode(), nil
	}
	return res, err
}

func newCommand(ctx context.Context, name string, args []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	// Without it a grandchild holding the pipe open blocks Run long after ctx is cancelled.
	cmd.WaitDelay = killWaitDelay
	return cmd
}

// cappedBuffer drops bytes past max but reports them as written so the child never blocks.
// The buffer is a named field: embedding it would promote ReadFrom, which io.Copy prefers and which bypasses the cap.
type cappedBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.max - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (b *cappedBuffer) Bytes() []byte { return b.buf.Bytes() }
