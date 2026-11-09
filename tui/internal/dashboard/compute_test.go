package dashboard_test

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

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const evaluatorWaitLimit = 5 * time.Second

func TestCompute_Counts(t *testing.T) {
	tests := []struct {
		name   string
		checks []check.Check
		want   map[check.Status]int
	}{
		{"mixed", []check.Check{fixedCheck("a",
			resultOf("a", check.StatusOK), resultOf("a", check.StatusOK),
			resultOf("a", check.StatusWarning), resultOf("a", check.StatusError))},
			map[check.Status]int{check.StatusOK: 2, check.StatusWarning: 1, check.StatusError: 1}},
		{"all ok", []check.Check{fixedCheck("a", resultOf("a", check.StatusOK))},
			map[check.Status]int{check.StatusOK: 1}},
		{"empty", nil, map[check.Status]int{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dashboard.Compute(context.Background(), check.Env{}, targetWith(tt.checks...))
			for _, s := range []check.Status{check.StatusOK, check.StatusWarning, check.StatusError, check.StatusUnknown} {
				if got.Counts[s] != tt.want[s] {
					t.Fatalf("Counts[%s] = %d, want %d (all: %v)", s, got.Counts[s], tt.want[s], got.Counts)
				}
			}
		})
	}
}

func TestCompute_TargetErrorSkipsAllWork(t *testing.T) {
	boom := errors.New("manifest broken")
	tg := targetWith(fixedCheck("a", resultOf("a", check.StatusError)))
	tg.Err = boom

	got := dashboard.Compute(context.Background(), check.Env{}, tg)

	if !errors.Is(got.Err, boom) {
		t.Fatalf("Err = %v, want %v", got.Err, boom)
	}
	if got.Counts == nil || len(got.Counts) != 0 || len(got.Results) != 0 {
		t.Fatalf("got Counts %v Results %v, want empty non-nil counts and no results", got.Counts, got.Results)
	}
}

func TestCompute_DriftDoesNotChangeCounts(t *testing.T) {
	tg := targetWith(fixedCheck("a", resultOf("a", check.StatusOK)))
	tg.Manifest = manifest.Manifest{Components: []manifest.Component{{Name: "x", Kind: manifest.KindFile}}}
	tg.Spec.Evaluators = map[manifest.Kind]drift.Evaluator{
		manifest.KindFile: func(context.Context, check.Env, manifest.Component) drift.Status {
			return drift.Status{Name: "x", State: drift.StateDrift}
		},
	}

	got := dashboard.Compute(context.Background(), check.Env{}, tg)

	if got.Counts[check.StatusOK] != 1 || got.Counts[check.StatusWarning] != 0 || got.Counts[check.StatusError] != 0 {
		t.Fatalf("Counts = %v, want only 1 ok", got.Counts)
	}
}

func TestCompute_ComponentStatuses(t *testing.T) {
	evaluate := func(state drift.State) drift.Evaluator {
		return func(_ context.Context, _ check.Env, c manifest.Component) drift.Status {
			return drift.Status{Name: c.Name, State: state}
		}
	}
	tg := target.Target{Spec: target.Spec{Evaluators: map[manifest.Kind]drift.Evaluator{
		manifest.KindFile: evaluate(drift.StateInstalled),
		manifest.KindDir:  evaluate(drift.StateDrift),
	}}, Manifest: manifest.Manifest{Components: []manifest.Component{
		{Name: "first", Kind: manifest.KindFile},
		{Name: "second", Kind: manifest.KindDir},
		{Name: "third", Kind: manifest.KindFileX},
	}}}

	got := dashboard.Compute(context.Background(), check.Env{}, tg).Components

	want := []drift.Status{
		{Name: "first", State: drift.StateInstalled},
		{Name: "second", State: drift.StateDrift},
		{Name: "third", State: drift.StateUnknown},
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

func TestCompute_PanickingEvaluatorBecomesUnknown(t *testing.T) {
	tg := target.Target{Spec: target.Spec{Evaluators: map[manifest.Kind]drift.Evaluator{
		manifest.KindFile: func(_ context.Context, _ check.Env, c manifest.Component) drift.Status {
			if c.Name == "bad" {
				panic("kaboom")
			}
			return drift.Status{Name: c.Name, State: drift.StateInstalled}
		},
	}}, Manifest: manifest.Manifest{Components: []manifest.Component{
		{Name: "good", Kind: manifest.KindFile},
		{Name: "bad", Kind: manifest.KindFile},
	}}}

	got := dashboard.Compute(context.Background(), check.Env{}, tg).Components

	if len(got) != 2 || got[0].State != drift.StateInstalled {
		t.Fatalf("got %+v, want good installed first", got)
	}
	if got[1].Name != "bad" || got[1].State != drift.StateUnknown || !strings.Contains(got[1].Reason, "kaboom") {
		t.Fatalf("got[1] = %+v, want unknown status named bad with the panic in Reason", got[1])
	}
}

func TestCompute_EvaluationIsBoundedAndParallel(t *testing.T) {
	const componentCount = dashboard.MaxConcurrentEvaluations * 3
	var running, peak atomic.Int32
	entered := make(chan struct{}, componentCount)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseAll)
	evaluate := func(_ context.Context, _ check.Env, c manifest.Component) drift.Status {
		now := running.Add(1)
		for p := peak.Load(); now > p && !peak.CompareAndSwap(p, now); p = peak.Load() {
		}
		entered <- struct{}{}
		<-release
		running.Add(-1)
		return drift.Status{Name: c.Name, State: drift.StateInstalled}
	}
	m := manifest.Manifest{}
	for i := range componentCount {
		m.Components = append(m.Components, manifest.Component{Name: fmt.Sprint("c", i), Kind: manifest.KindFile})
	}
	tg := target.Target{Spec: target.Spec{Evaluators: map[manifest.Kind]drift.Evaluator{manifest.KindFile: evaluate}}, Manifest: m}

	done := make(chan dashboard.Dashboard)
	go func() { done <- dashboard.Compute(context.Background(), check.Env{}, tg) }()

	var finished dashboard.Dashboard
	for range dashboard.MaxConcurrentEvaluations {
		select {
		case <-entered:
		case finished = <-done:
			t.Fatalf("Compute returned before %d evaluations ran concurrently: %+v", dashboard.MaxConcurrentEvaluations, finished)
		case <-time.After(evaluatorWaitLimit):
			t.Fatalf("fewer than %d evaluations ran concurrently", dashboard.MaxConcurrentEvaluations)
		}
	}
	releaseAll()
	got := <-done

	if p := peak.Load(); p != dashboard.MaxConcurrentEvaluations {
		t.Fatalf("peak concurrent evaluations = %d, want %d", p, dashboard.MaxConcurrentEvaluations)
	}
	if len(got.Components) != componentCount {
		t.Fatalf("got %d component statuses, want %d", len(got.Components), componentCount)
	}
}

func TestCompute_FeedsMissingDepsAndLastBackup(t *testing.T) {
	backups := t.TempDir()
	if err := os.Mkdir(filepath.Join(backups, "20240301-120000"), 0o755); err != nil {
		t.Fatal(err)
	}
	tg := target.Target{
		Spec: target.Spec{BackupDirs: func(home, repo string) []backup.Dir {
			return []backup.Dir{{Path: filepath.Join(home, filepath.Base(backups)), Label: "claude"}}
		}},
		Manifest: manifest.Manifest{Components: []manifest.Component{{Name: "hooks", Kind: manifest.KindDirX, Needs: []string{"jq"}}}},
		Checks: []check.Check{fixedCheck(check.DependenciesID,
			check.Result{CheckID: check.DependenciesID, Subject: "jq", Status: check.StatusError},
			check.Result{CheckID: "other", Subject: "ghost", Status: check.StatusError})},
	}
	env := check.Env{FS: osfs.FS{}, Home: filepath.Dir(backups)}

	got := dashboard.Compute(context.Background(), env, tg)

	wantDeps := []dashboard.MissingDep{{Tool: "jq", Dependents: []string{"hooks"}}}
	if !reflect.DeepEqual(got.MissingDeps, wantDeps) {
		t.Fatalf("MissingDeps = %v, want %v", got.MissingDeps, wantDeps)
	}
	if got.LastBackup == nil || got.LastBackup.Label != "claude" {
		t.Fatalf("LastBackup = %+v, want label claude", got.LastBackup)
	}
}
