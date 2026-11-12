package osexec_test

import (
	"context"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/osexec"
)

func skipWithoutSh(t *testing.T, e osexec.Exec) {
	t.Helper()
	if _, err := e.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
}

func TestRun_SplitsStreamsAndKeepsExitCode(t *testing.T) {
	e := osexec.Exec{}
	skipWithoutSh(t, e)

	got, err := e.Run(context.Background(), "sh", "-c", "echo out; echo err >&2; exit 3")

	if err != nil {
		t.Fatalf("non-zero exit must not be a Run error, got %v", err)
	}
	if string(got.Stdout) != "out\n" || string(got.Stderr) != "err\n" || got.ExitCode != 3 {
		t.Errorf("got stdout %q stderr %q code %d, want \"out\\n\" \"err\\n\" 3", got.Stdout, got.Stderr, got.ExitCode)
	}
}

func TestRun_CapsEachStreamAtMaxRunOutputBytes(t *testing.T) {
	e := osexec.Exec{}
	skipWithoutSh(t, e)

	got, err := e.Run(context.Background(), "sh", "-c", "head -c 2000000 /dev/zero; head -c 2000000 /dev/zero >&2")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Stdout) != osexec.MaxRunOutputBytes || len(got.Stderr) != osexec.MaxRunOutputBytes {
		t.Errorf("captured stdout %d stderr %d, want cap %d each", len(got.Stdout), len(got.Stderr), osexec.MaxRunOutputBytes)
	}
}

func TestRun_CtxDeadlineKillsCommandAndReturnsError(t *testing.T) {
	e := osexec.Exec{}
	skipWithoutSh(t, e)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := e.Run(ctx, "sh", "-c", "sleep 5")

	if err == nil {
		t.Fatal("want an error when ctx expires")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Run took %s, want it to return soon after the deadline", elapsed)
	}
}

func TestRun_MissingCommandIsError(t *testing.T) {
	_, err := osexec.Exec{}.Run(context.Background(), "definitely-not-a-command-xyz")

	if err == nil {
		t.Fatal("want an error when the command cannot start")
	}
}
