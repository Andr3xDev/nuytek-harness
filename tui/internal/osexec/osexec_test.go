package osexec_test

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/osexec"
)

func TestOutput_CapsCapturedBytes(t *testing.T) {
	e := osexec.Exec{}
	if _, err := e.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	got, err := e.Output(context.Background(), "sh", "-c", "head -c 200000 /dev/zero")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != osexec.MaxOutputBytes {
		t.Fatalf("captured %d bytes, want cap %d", len(got), osexec.MaxOutputBytes)
	}
}
