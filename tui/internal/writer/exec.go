package writer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

// killedPipeGrace bounds the wait for output pipes after the process group is killed.
const killedPipeGrace = time.Second

// maxPendingLine bounds memory for output without a newline.
const maxPendingLine = 64 << 10

func execute(ctx context.Context, e plan.Exec, line func(string)) plan.StepResult {
	if len(e.Argv) == 0 {
		return fromError(errors.New("exec: empty command"))
	}
	runCtx := ctx
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		runCtx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(runCtx, e.Argv[0], e.Argv[1:]...)
	// A shell leaves its children holding the pipes, so the whole process group must die.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = killedPipeGrace
	out := &lineWriter{emit: line}
	cmd.Stdout, cmd.Stderr = out, out

	err := cmd.Run()
	out.flush()
	switch {
	case err == nil, errors.Is(err, exec.ErrWaitDelay):
		// ErrWaitDelay means the command exited 0 but a child kept the pipes open.
		return plan.StepResult{Status: plan.StepDone}
	case ctx.Err() != nil:
		return plan.StepResult{Status: plan.StepFailed, Err: ctx.Err()}
	case runCtx.Err() != nil:
		status := plan.StepFailed
		if e.Warn {
			status = plan.StepWarned
		}
		return plan.StepResult{Status: status, Err: fmt.Errorf("%s timed out after %s", e.Describe(), e.Timeout)}
	case e.Warn:
		return plan.StepResult{Status: plan.StepWarned, Err: fmt.Errorf("%s: %w", e.Describe(), err)}
	}
	return plan.StepResult{Status: plan.StepFailed, Err: fmt.Errorf("%s: %w", e.Describe(), err)}
}

// lineWriter splits written bytes into lines; stdout and stderr share one so exec copies with a single goroutine.
type lineWriter struct {
	emit func(string)
	buf  bytes.Buffer
}

func (l *lineWriter) Write(p []byte) (int, error) {
	l.buf.Write(p)
	for {
		i := bytes.IndexByte(l.buf.Bytes(), '\n')
		if i < 0 {
			if l.buf.Len() >= maxPendingLine {
				l.flush()
			}
			return len(p), nil
		}
		l.emit(strings.TrimSuffix(string(l.buf.Next(i + 1)[:i]), "\r"))
	}
}

func (l *lineWriter) flush() {
	if l.buf.Len() > 0 {
		l.emit(strings.TrimSuffix(l.buf.String(), "\r"))
		l.buf.Reset()
	}
}
