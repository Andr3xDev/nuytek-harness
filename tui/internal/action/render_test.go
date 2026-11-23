package action_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
)

const (
	listedRel     = "tech.md"
	sourceText    = "root {{SPECTER_ROOT}}\n"
	defaultRooted = "root ~/dev/specter\n"
	customRooted  = "root ~/work/specs\n"
)

// renderFixture holds one listed agent file and one unlisted one in a dir component.
func renderFixture(t *testing.T, deployed string) (*fixture, manifest.Component) {
	t.Helper()
	f := newFixture(t)
	f.writeRepo("targets/rendered-files.txt", ".claude/agents/"+listedRel+"\n", 0o644)
	f.writeRepo("agents/"+listedRel, sourceText, 0o644)
	f.writeHome(".claude/agents/"+listedRel, deployed, 0o644)
	agents := pullable(manifest.Component{Name: "agents", Kind: manifest.KindDir, Source: "agents", Target: ".claude/agents"})
	return f, agents
}

func customRootState(f *fixture) state.State {
	return state.State{SpecterRoot: filepath.Join(f.home, "work/specs")}
}

func TestPlanInstall_RendersListedFiles(t *testing.T) {
	t.Run("deployed copy rendered with the saved root needs no copy", func(t *testing.T) {
		f, agents := renderFixture(t, customRooted)
		env := f.env()
		env.Saved = customRootState(f)

		p, err := action.PlanInstall(context.Background(), env, f.target(agents), []string{"agents"})

		requireNoError(t, err)
		if copies := stepsOf[plan.Copy](p); len(copies) != 0 {
			t.Fatalf("copy steps = %+v, want none: rendered copy is up to date", copies)
		}
	})

	t.Run("root changed copies the file with a transform that renders the saved root", func(t *testing.T) {
		f, agents := renderFixture(t, defaultRooted)
		env := f.env()
		env.Saved = customRootState(f)

		p, err := action.PlanInstall(context.Background(), env, f.target(agents), []string{"agents"})

		requireNoError(t, err)
		cp := onlyStep[plan.Copy](t, p)
		if len(cp.Files) != 1 || cp.Files[0] != listedRel {
			t.Fatalf("copy files = %v, want [%s]", cp.Files, listedRel)
		}
		if cp.Transform == nil {
			t.Fatal("copy has no transform")
		}
		if got := string(cp.Transform(listedRel, []byte(sourceText))); got != customRooted {
			t.Errorf("transform output = %q, want %q", got, customRooted)
		}
	})
}

// Unrender restores only the current root, so a file rendered for another root would put a stale path in the repo.
func TestPlanPull_SkipsListedFilesRenderedForAnotherRoot(t *testing.T) {
	tests := map[string]string{
		"another root":        "root ~/old/place\nedited\n",
		"placeholder left in": "root {{SPECTER_ROOT}}\nedited\n",
	}
	for name, deployed := range tests {
		t.Run(name, func(t *testing.T) {
			f, agents := renderFixture(t, deployed)
			f.writeRepo("agents/plain.md", "plain\n", 0o644)
			f.writeHome(".claude/agents/plain.md", "plain edited\n", 0o644)
			env := f.env()
			env.Saved = customRootState(f)

			p, err := action.PlanPull(context.Background(), env, f.target(agents), []string{"agents"})

			requireNoError(t, err)
			cp := onlyStep[plan.Copy](t, p)
			if len(cp.Files) != 1 || cp.Files[0] != "plain.md" {
				t.Fatalf("copy files = %v, want only the unlisted plain.md", cp.Files)
			}
			skipped := false
			for _, row := range cp.Rows {
				if row.Path == listedRel && row.Effect == plan.EffectSkip && strings.Contains(row.Note, "reinstall first") {
					skipped = true
				}
			}
			if !skipped {
				t.Errorf("rows = %+v, want %s skipped with a \"reinstall first\" note", cp.Rows, listedRel)
			}
		})
	}
}

func TestPlanPull_UnrendersDeployedSide(t *testing.T) {
	t.Run("unedited rendered file is not pulled", func(t *testing.T) {
		f, agents := renderFixture(t, customRooted)
		env := f.env()
		env.Saved = customRootState(f)

		p, err := action.PlanPull(context.Background(), env, f.target(agents), []string{"agents"})

		requireNoError(t, err)
		for _, cp := range stepsOf[plan.Copy](p) {
			if len(cp.Files) != 0 {
				t.Fatalf("copy files = %v, want none: only the rendered root differs", cp.Files)
			}
		}
	})

	t.Run("edited file is pulled with a transform that restores the placeholder", func(t *testing.T) {
		f, agents := renderFixture(t, customRooted+"new line\n")
		env := f.env()
		env.Saved = customRootState(f)

		p, err := action.PlanPull(context.Background(), env, f.target(agents), []string{"agents"})

		requireNoError(t, err)
		cp := onlyStep[plan.Copy](t, p)
		if len(cp.Files) != 1 || cp.Files[0] != listedRel {
			t.Fatalf("copy files = %v, want [%s]", cp.Files, listedRel)
		}
		if cp.Transform == nil {
			t.Fatal("copy has no transform")
		}
		if got, want := string(cp.Transform(listedRel, []byte(customRooted+"new line\n"))), sourceText+"new line\n"; got != want {
			t.Errorf("transform output = %q, want %q", got, want)
		}
	})
}
