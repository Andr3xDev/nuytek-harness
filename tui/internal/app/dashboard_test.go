package app_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/adapters/fsx"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/app"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/domain"
)

const evaluatorWaitLimit = 5 * time.Second

func resultOf(id string, status domain.Status) domain.CheckResult {
	return domain.CheckResult{CheckID: id, Subject: id, Status: status}
}

func fixedCheck(id string, results ...domain.CheckResult) app.Check {
	return app.Check{ID: id, Run: func(context.Context, app.CheckEnv) []domain.CheckResult { return results }}
}

func targetWith(checks ...app.Check) app.Target {
	return app.Target{Spec: app.TargetSpec{ID: "t"}, Checks: checks}
}

func TestComputeDashboard_Counts(t *testing.T) {
	tests := []struct {
		name   string
		checks []app.Check
		want   map[domain.Status]int
	}{
		{"mixed", []app.Check{fixedCheck("a",
			resultOf("a", domain.StatusOK), resultOf("a", domain.StatusOK),
			resultOf("a", domain.StatusWarning), resultOf("a", domain.StatusError))},
			map[domain.Status]int{domain.StatusOK: 2, domain.StatusWarning: 1, domain.StatusError: 1}},
		{"all ok", []app.Check{fixedCheck("a", resultOf("a", domain.StatusOK))},
			map[domain.Status]int{domain.StatusOK: 1}},
		{"empty", nil, map[domain.Status]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := app.ComputeDashboard(context.Background(), app.CheckEnv{}, targetWith(tt.checks...))
			for _, s := range []domain.Status{domain.StatusOK, domain.StatusWarning, domain.StatusError, domain.StatusUnknown} {
				if got.Counts[s] != tt.want[s] {
					t.Fatalf("Counts[%s] = %d, want %d (all: %v)", s, got.Counts[s], tt.want[s], got.Counts)
				}
			}
		})
	}
}

func TestComputeDashboard_TargetErrorSkipsAllWork(t *testing.T) {
	boom := errors.New("manifest broken")
	target := targetWith(fixedCheck("a", resultOf("a", domain.StatusError)))
	target.Err = boom

	got := app.ComputeDashboard(context.Background(), app.CheckEnv{}, target)

	if !errors.Is(got.Err, boom) {
		t.Fatalf("Err = %v, want %v", got.Err, boom)
	}
	if got.Counts == nil || len(got.Counts) != 0 || len(got.Results) != 0 {
		t.Fatalf("got Counts %v Results %v, want empty non-nil counts and no results", got.Counts, got.Results)
	}
}

func TestComputeDashboard_DriftDoesNotChangeCounts(t *testing.T) {
	target := targetWith(fixedCheck("a", resultOf("a", domain.StatusOK)))
	target.Manifest = domain.Manifest{Components: []domain.Component{{Name: "x", Kind: domain.KindFile}}}
	target.Spec.Evaluators = map[domain.Kind]app.Evaluator{
		domain.KindFile: func(context.Context, app.CheckEnv, domain.Component) domain.ComponentStatus {
			return domain.ComponentStatus{Name: "x", State: domain.StateDrift}
		},
	}

	got := app.ComputeDashboard(context.Background(), app.CheckEnv{}, target)

	if got.Counts[domain.StatusOK] != 1 || got.Counts[domain.StatusWarning] != 0 || got.Counts[domain.StatusError] != 0 {
		t.Fatalf("Counts = %v, want only 1 ok", got.Counts)
	}
}

func TestDashboard_Issues(t *testing.T) {
	warn, err1, err2, ok := resultOf("w", domain.StatusWarning), resultOf("e1", domain.StatusError), resultOf("e2", domain.StatusError), resultOf("o", domain.StatusOK)

	t.Run("errors first and ok excluded", func(t *testing.T) {
		got := app.Dashboard{Results: []domain.CheckResult{warn, err1, ok, err2}}.Issues()
		want := []domain.CheckResult{err1, err2, warn}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("empty when all ok", func(t *testing.T) {
		if got := (app.Dashboard{Results: []domain.CheckResult{ok}}).Issues(); len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})
}

func TestComputeDashboard_ComponentStatuses(t *testing.T) {
	evaluate := func(state domain.ComponentState) app.Evaluator {
		return func(_ context.Context, _ app.CheckEnv, c domain.Component) domain.ComponentStatus {
			return domain.ComponentStatus{Name: c.Name, State: state}
		}
	}
	target := app.Target{Spec: app.TargetSpec{Evaluators: map[domain.Kind]app.Evaluator{
		domain.KindFile: evaluate(domain.StateInstalled),
		domain.KindDir:  evaluate(domain.StateDrift),
	}}, Manifest: domain.Manifest{Components: []domain.Component{
		{Name: "first", Kind: domain.KindFile},
		{Name: "second", Kind: domain.KindDir},
		{Name: "third", Kind: domain.KindFileX},
	}}}

	got := app.ComputeDashboard(context.Background(), app.CheckEnv{}, target).Components

	want := []domain.ComponentStatus{
		{Name: "first", State: domain.StateInstalled},
		{Name: "second", State: domain.StateDrift},
		{Name: "third", State: domain.StateUnknown},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i].Name != want[i].Name || got[i].State != want[i].State {
			t.Fatalf("got[%d] = %+v, want name %q state %q", i, got[i], want[i].Name, want[i].State)
		}
	}
}

func TestComputeDashboard_PanickingEvaluatorBecomesUnknown(t *testing.T) {
	target := app.Target{Spec: app.TargetSpec{Evaluators: map[domain.Kind]app.Evaluator{
		domain.KindFile: func(_ context.Context, _ app.CheckEnv, c domain.Component) domain.ComponentStatus {
			if c.Name == "bad" {
				panic("kaboom")
			}
			return domain.ComponentStatus{Name: c.Name, State: domain.StateInstalled}
		},
	}}, Manifest: domain.Manifest{Components: []domain.Component{
		{Name: "good", Kind: domain.KindFile},
		{Name: "bad", Kind: domain.KindFile},
	}}}

	got := app.ComputeDashboard(context.Background(), app.CheckEnv{}, target).Components

	if len(got) != 2 || got[0].State != domain.StateInstalled {
		t.Fatalf("got %+v, want good installed first", got)
	}
	if got[1].Name != "bad" || got[1].State != domain.StateUnknown || !strings.Contains(got[1].Reason, "kaboom") {
		t.Fatalf("got[1] = %+v, want unknown status named bad with the panic in Reason", got[1])
	}
}

func TestComputeDashboard_EvaluationIsBoundedAndParallel(t *testing.T) {
	const componentCount = app.MaxConcurrentEvaluations * 3
	var running, peak atomic.Int32
	entered := make(chan struct{}, componentCount)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseAll)
	evaluate := func(_ context.Context, _ app.CheckEnv, c domain.Component) domain.ComponentStatus {
		now := running.Add(1)
		for p := peak.Load(); now > p && !peak.CompareAndSwap(p, now); p = peak.Load() {
		}
		entered <- struct{}{}
		<-release
		running.Add(-1)
		return domain.ComponentStatus{Name: c.Name, State: domain.StateInstalled}
	}
	m := domain.Manifest{}
	for i := range componentCount {
		m.Components = append(m.Components, domain.Component{Name: fmt.Sprint("c", i), Kind: domain.KindFile})
	}
	target := app.Target{Spec: app.TargetSpec{Evaluators: map[domain.Kind]app.Evaluator{domain.KindFile: evaluate}}, Manifest: m}

	done := make(chan app.Dashboard)
	go func() { done <- app.ComputeDashboard(context.Background(), app.CheckEnv{}, target) }()

	var finished app.Dashboard
	for range app.MaxConcurrentEvaluations {
		select {
		case <-entered:
		case finished = <-done:
			t.Fatalf("ComputeDashboard returned before %d evaluations ran concurrently: %+v", app.MaxConcurrentEvaluations, finished)
		case <-time.After(evaluatorWaitLimit):
			t.Fatalf("fewer than %d evaluations ran concurrently", app.MaxConcurrentEvaluations)
		}
	}
	releaseAll()
	got := <-done

	if p := peak.Load(); p != app.MaxConcurrentEvaluations {
		t.Fatalf("peak concurrent evaluations = %d, want %d", p, app.MaxConcurrentEvaluations)
	}
	if len(got.Components) != componentCount {
		t.Fatalf("got %d component statuses, want %d", len(got.Components), componentCount)
	}
}

func TestComputeDashboard_FeedsMissingDepsAndLastBackup(t *testing.T) {
	backups := t.TempDir()
	if err := os.Mkdir(filepath.Join(backups, "20240301-120000"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := app.Target{
		Spec: app.TargetSpec{BackupDirs: func(home, repo string) []app.BackupDir {
			return []app.BackupDir{{Path: filepath.Join(home, filepath.Base(backups)), Label: "claude"}}
		}},
		Manifest: domain.Manifest{Components: []domain.Component{{Name: "hooks", Kind: domain.KindDirX, Needs: []string{"jq"}}}},
		Checks: []app.Check{fixedCheck(app.DependenciesCheckID,
			domain.CheckResult{CheckID: app.DependenciesCheckID, Subject: "jq", Status: domain.StatusError},
			domain.CheckResult{CheckID: "other", Subject: "ghost", Status: domain.StatusError})},
	}
	env := app.CheckEnv{FS: fsx.FS{}, Home: filepath.Dir(backups)}

	got := app.ComputeDashboard(context.Background(), env, target)

	wantDeps := []domain.MissingDep{{Tool: "jq", Dependents: []string{"hooks"}}}
	if !reflect.DeepEqual(got.MissingDeps, wantDeps) {
		t.Fatalf("MissingDeps = %v, want %v", got.MissingDeps, wantDeps)
	}
	if got.LastBackup == nil || got.LastBackup.Label != "claude" {
		t.Fatalf("LastBackup = %+v, want label claude", got.LastBackup)
	}
}
