package osexec

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

var _ check.Exec = Exec{}

// MaxOutputBytes caps captured output; version probes need a few lines.
const MaxOutputBytes = 64 * 1024

const killWaitDelay = 500 * time.Millisecond

// Exec is the os/exec-backed check.Exec.
type Exec struct{}

func (Exec) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (Exec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	var buf cappedBuffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	// Without it a grandchild holding the pipe open blocks Run long after ctx is cancelled.
	cmd.WaitDelay = killWaitDelay
	err := cmd.Run()
	return buf.Bytes(), err
}

// cappedBuffer drops bytes past MaxOutputBytes but reports them as written so the child never blocks.
// The buffer is a named field: embedding it would promote ReadFrom, which io.Copy prefers and which bypasses the cap.
type cappedBuffer struct{ buf bytes.Buffer }

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := MaxOutputBytes - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (b *cappedBuffer) Bytes() []byte { return b.buf.Bytes() }
