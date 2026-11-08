package app_test

import (
	"context"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/app"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/domain"
)

func okCheck(id string) app.Check {
	return app.Check{ID: id, Run: func(context.Context, app.CheckEnv) []domain.CheckResult {
		return []domain.CheckResult{{CheckID: id, Subject: id, Status: domain.StatusOK}}
	}}
}

func TestRunChecks_PanicIsIsolatedAndOrderKept(t *testing.T) {
	boom := app.Check{ID: "boom", Run: func(context.Context, app.CheckEnv) []domain.CheckResult { panic("kaboom") }}

	got := app.RunChecks(context.Background(), app.CheckEnv{}, []app.Check{okCheck("a"), boom, okCheck("c")})

	if len(got) != 3 {
		t.Fatalf("got %d results, want 3: %v", len(got), got)
	}
	if got[0].CheckID != "a" || got[0].Status != domain.StatusOK {
		t.Fatalf("got[0] = %+v, want ok result of check a", got[0])
	}
	if got[1].CheckID != "boom" || got[1].Status != domain.StatusError {
		t.Fatalf("got[1] = %+v, want error result for check boom", got[1])
	}
	if got[2].CheckID != "c" || got[2].Status != domain.StatusOK {
		t.Fatalf("got[2] = %+v, want ok result of check c", got[2])
	}
}
