package check_test

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

func TestRun_PanicIsIsolatedAndOrderKept(t *testing.T) {
	boom := check.Check{ID: "boom", Run: func(context.Context, check.Env) []check.Result { panic("kaboom") }}

	got := check.Run(context.Background(), check.Env{}, []check.Check{okCheck("a"), boom, okCheck("c")})

	if len(got) != 3 {
		t.Fatalf("got %d results, want 3: %v", len(got), got)
	}
	if got[0].CheckID != "a" || got[0].Status != check.StatusOK {
		t.Fatalf("got[0] = %+v, want ok result of check a", got[0])
	}
	if got[1].CheckID != "boom" || got[1].Status != check.StatusError {
		t.Fatalf("got[1] = %+v, want error result for check boom", got[1])
	}
	if got[2].CheckID != "c" || got[2].Status != check.StatusOK {
		t.Fatalf("got[2] = %+v, want ok result of check c", got[2])
	}
}
