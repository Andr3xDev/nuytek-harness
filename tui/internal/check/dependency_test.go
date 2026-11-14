package check_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const testProbeTimeout = 200 * time.Millisecond

func depsCheck(tools ...string) check.Check {
	return check.Dependencies(check.DependenciesID, func() []string { return tools }, testProbeTimeout)
}

func TestDependencies_StatusPerTool(t *testing.T) {
	exec := &fakeExec{
		onPath: map[string]bool{"jq": true, "odd": true, "broken": true, "slow": true},
		outputs: map[string]string{
			"jq":  "jq-1.7.1\n",
			"odd": "no digits here",
		},
		probeErrs: map[string]error{"broken": errors.New("exit status 2")},
		hang:      map[string]bool{"slow": true},
	}

	got := runCheck(t, depsCheck("jq", "ghost", "odd", "broken", "slow"), newHome(t, exec))

	tests := []struct {
		tool        string
		want        check.Status
		wantMessage string
	}{
		{"jq", check.StatusOK, "1.7.1"},
		{"ghost", check.StatusError, "ghost"},
		{"odd", check.StatusOK, "unknown"},
		{"broken", check.StatusOK, "unknown"},
		{"slow", check.StatusWarning, "slow"},
	}
	if len(got) != len(tests) {
		t.Fatalf("got %d results, want %d: %+v", len(got), len(tests), got)
	}
	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			r := resultFor(t, got, tt.tool)
			if r.Status != tt.want {
				t.Errorf("status = %q, want %q (%+v)", r.Status, tt.want, r)
			}
			if r.CheckID != check.DependenciesID {
				t.Errorf("CheckID = %q, want %q", r.CheckID, check.DependenciesID)
			}
			if !strings.Contains(r.Message, tt.wantMessage) {
				t.Errorf("message %q lacks %q", r.Message, tt.wantMessage)
			}
		})
	}
}

func TestDependencies_ResultsKeepInputOrder(t *testing.T) {
	exec := &fakeExec{
		onPath:  map[string]bool{"a": true, "b": true, "c": true},
		outputs: map[string]string{"a": "1.0", "b": "2.0", "c": "3.0"},
	}
	tools := []string{"c", "a", "ghost", "b"}

	got := runCheck(t, depsCheck(tools...), newHome(t, exec))

	var subjects []string
	for _, r := range got {
		subjects = append(subjects, r.Subject)
	}
	if !slices.Equal(subjects, tools) {
		t.Fatalf("subjects = %v, want %v", subjects, tools)
	}
}

func TestDependencies_VersionArgs(t *testing.T) {
	exec := &fakeExec{
		onPath:  map[string]bool{"engram": true, "node": true},
		outputs: map[string]string{"engram": "engram 3.6.0", "node": "v22.1.0"},
	}

	runCheck(t, depsCheck("engram", "node"), newHome(t, exec))

	if got := exec.argsOf("engram"); !slices.Equal(got, []string{"version"}) {
		t.Errorf("engram args = %v, want [version]", got)
	}
	if got := exec.argsOf("node"); !slices.Equal(got, []string{"--version"}) {
		t.Errorf("node args = %v, want [--version]", got)
	}
}

// barrierExec makes every probe wait until all of them have started, which only happens when probes run concurrently.
type barrierExec struct {
	fakeExec
	remaining  atomic.Int32
	allStarted chan struct{}
}

func (b *barrierExec) Output(ctx context.Context, _ string, _ ...string) ([]byte, error) {
	if b.remaining.Add(-1) == 0 {
		close(b.allStarted)
	}
	select {
	case <-b.allStarted:
		return []byte("1.0.0"), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestDependencies_ProbesRunConcurrently(t *testing.T) {
	tools := []string{"s1", "s2", "s3", "s4", "s5"}
	exec := &barrierExec{fakeExec: fakeExec{onPath: map[string]bool{}}, allStarted: make(chan struct{})}
	exec.remaining.Store(int32(len(tools)))
	for _, name := range tools {
		exec.onPath[name] = true
	}
	c := check.Dependencies(check.DependenciesID, func() []string { return tools }, 5*time.Second)

	got := runCheck(t, c, newHome(t, exec))

	if len(got) != len(tools) {
		t.Fatalf("got %d results, want %d", len(got), len(tools))
	}
	for _, r := range got {
		if r.Status != check.StatusOK {
			t.Errorf("%s: status %q, want ok: probes are not concurrent (%+v)", r.Subject, r.Status, r)
		}
	}
}

type panicExec struct{ fakeExec }

func (p *panicExec) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "boom" {
		panic("kaboom")
	}
	return p.fakeExec.Output(ctx, name, args...)
}

func TestDependencies_PanickingProbeIsIsolated(t *testing.T) {
	exec := &panicExec{fakeExec{
		onPath:  map[string]bool{"boom": true, "jq": true},
		outputs: map[string]string{"jq": "jq-1.7.1"},
	}}

	got := runCheck(t, depsCheck("boom", "jq"), newHome(t, exec))

	boom := resultFor(t, got, "boom")
	if boom.Status != check.StatusUnknown || !strings.Contains(boom.Message, "probe panicked: kaboom") {
		t.Errorf("boom = %+v, want unknown with \"probe panicked: kaboom\"", boom)
	}
	if jq := resultFor(t, got, "jq"); jq.Status != check.StatusOK {
		t.Errorf("jq = %+v, want ok", jq)
	}
}

func TestDependencies_ParentCancelIsNotTimeout(t *testing.T) {
	exec := &fakeExec{onPath: map[string]bool{"slow": true}, hang: map[string]bool{"slow": true}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := depsCheck("slow").Run(ctx, newHome(t, exec))

	r := resultFor(t, got, "slow")
	if r.Status != check.StatusUnknown || strings.Contains(r.Message, "timed out") {
		t.Errorf("slow = %+v, want unknown and not a timeout", r)
	}
}

func TestDependencies_MissingToolSuggestsInstall(t *testing.T) {
	exec := &fakeExec{onPath: map[string]bool{"jq": true}, outputs: map[string]string{"jq": "jq-1.7.1"}}

	got := runCheck(t, depsCheck("ghost", "jq"), newHome(t, exec))

	if s := resultFor(t, got, "ghost").Suggestion; !strings.Contains(s, "install ghost") {
		t.Errorf("ghost suggestion = %q, want it to say to install ghost", s)
	}
	if s := resultFor(t, got, "jq").Suggestion; s != "" {
		t.Errorf("jq suggestion = %q, want none for a found tool", s)
	}
}

func TestProbeResult_BuildsFullResult(t *testing.T) {
	exec := &fakeExec{onPath: map[string]bool{"jq": true}, outputs: map[string]string{"jq": "jq-1.7.1"}}

	found := check.ProbeResult(context.Background(), exec, "doctor", "jq", testProbeTimeout)
	missing := check.ProbeResult(context.Background(), exec, "doctor", "ghost", testProbeTimeout)

	if found.CheckID != "doctor" || found.Subject != "jq" || found.Status != check.StatusOK || !strings.Contains(found.Message, "1.7.1") {
		t.Errorf("found = %+v, want ok doctor/jq with version 1.7.1", found)
	}
	if missing.Status != check.StatusError || missing.Subject != "ghost" || missing.Suggestion == "" {
		t.Errorf("missing = %+v, want error ghost with a suggestion", missing)
	}
}
