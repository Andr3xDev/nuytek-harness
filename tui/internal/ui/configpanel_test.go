package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/config"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dryrun"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	typedPath      = "/data/q?i"
	backupsRow     = 1
	updateConfirm  = "Pull the latest harness and rebuild the TUI?"
	updatePullStep = "pull --ff-only"
	updateBuild    = "bootstrap-tui.sh"
)

type configFakes struct {
	saves        []state.State
	saveErr      error
	specterLoads int
}

func (f *configFakes) save(s state.State) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	f.saves = append(f.saves, s)
	return nil
}

// configRoot has the Config panel focused over one target whose backups dir is configurable, its Backups tab loaded.
func configRoot(t *testing.T, f *configFakes, saved func(home string) state.State) (*Root, *backupsTab, string) {
	t.Helper()
	home := t.TempDir()
	tg := target.Target{Spec: target.Spec{
		ID: "t", Name: "T", DefaultBackupsRel: ".bk", OwnTreeRel: ".claude",
		Checks: availableTarget().Spec.Checks,
		BackupDirs: func(home, _, homeStore string) []backup.Dir {
			return []backup.Dir{{Path: homeStore, Label: config.PathLabel(home, homeStore), Kind: backup.StoreHome, Base: home}}
		},
	}}
	env := check.Env{FS: osfs.FS{}, Home: home, Repo: t.TempDir()}
	if saved != nil {
		env.Saved = saved(home)
	}
	deps := Deps{
		Env:       env,
		SaveState: f.save,
		SpecterLoad: func(context.Context, check.Env) specter.Facts {
			f.specterLoads++
			return specter.Facts{}
		},
	}
	r := NewRoot([]target.Target{tg}, deps, nil)
	r.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	var bt *backupsTab
	for _, tb := range r.tabs[0] {
		if b, ok := tb.(*backupsTab); ok {
			bt = b
		}
	}
	if bt == nil {
		t.Fatal("no Backups tab")
	}
	pump(t, r, bt.load())
	enterConfig(r)
	return r, bt, home
}

func enterConfig(r *Root) {
	press(r, upKey)
	press(r, enterKey)
}

// editRow opens the field on the given row and replaces its prefilled value.
func editRow(r *Root, row int, value string) {
	for range row {
		press(r, downKey)
	}
	press(r, textKey("e"))
	r.configPanel.input.SetValue(value)
}

func TestConfigPanel_ShortcutLettersTypeIntoTheOpenField(t *testing.T) {
	r, _, _ := configRoot(t, &configFakes{}, nil)
	editRow(r, 0, "")

	for _, c := range typedPath {
		if cmd := press(r, textKey(string(c))); isQuit(cmd) {
			t.Fatalf("key %q quit the TUI while the field was open", c)
		}
	}

	if got := r.configPanel.input.Value(); got != typedPath {
		t.Errorf("field = %q, want %q", got, typedPath)
	}
	if r.showHelp || r.wizard != nil || !r.configPanel.editing {
		t.Errorf("showHelp = %v, wizard open = %v, editing = %v; want no help, no wizard, field still open", r.showHelp, r.wizard != nil, r.configPanel.editing)
	}
}

func TestConfigPanel_PasteGoesIntoTheOpenField(t *testing.T) {
	r, _, _ := configRoot(t, &configFakes{}, nil)
	editRow(r, 0, "")

	r.Update(tea.PasteMsg{Content: typedPath})

	if got := r.configPanel.input.Value(); got != typedPath {
		t.Errorf("field = %q, want the pasted %q", got, typedPath)
	}
}

func TestPullTab_ListsOffTheUILoopAndDropsStaleResults(t *testing.T) {
	svc := newRoot(t, availableTarget()).svc
	p := newPullTab(svc, &targetState{}, availableTarget())

	first := p.Activate(availableTarget())
	if first == nil || p.listed {
		t.Fatalf("Activate cmd = %v, listed = %v; want a background cmd and no listing yet", first, p.listed)
	}
	p.Activate(availableTarget())

	p.Update(pullListed{p: p, epoch: 1, names: []string{"stale"}})
	if p.listed {
		t.Fatal("a superseded result was applied")
	}
	p.Update(pullListed{p: p, epoch: 2, names: []string{"agents"}})
	if !p.listed || len(p.pick.choices) != 1 || p.pick.choices[0].name != "agents" {
		t.Errorf("listed = %v, choices = %v; want the current result applied", p.listed, p.pick.choices)
	}
}

func TestConfigPanel_CtrlCQuitsWhileEditing(t *testing.T) {
	r, _, _ := configRoot(t, &configFakes{}, nil)
	editRow(r, 0, "")

	if !isQuit(press(r, ctrlC)) {
		t.Fatal("ctrl+c did not quit while the field was open")
	}
}

func TestConfigPanel_SaveAppliesToEnvAndReloadsSpecterAndBackups(t *testing.T) {
	f := &configFakes{}
	r, bt, home := configRoot(t, f, nil)
	newDir := filepath.Join(home, "bk")
	if err := os.MkdirAll(filepath.Join(newDir, newerStamp), 0o755); err != nil {
		t.Fatal(err)
	}
	loadsBefore := f.specterLoads

	editRow(r, backupsRow, newDir)
	pump(t, r, press(r, enterKey))

	if got := r.svc.Env.Saved.Targets["t"].BackupsDir; got != newDir || len(f.saves) != 1 {
		t.Errorf("Env.Saved backups dir = %q, saves = %d; want %q, 1", got, len(f.saves), newDir)
	}
	if f.specterLoads != loadsBefore+1 {
		t.Errorf("Specter loads = %d, want %d: a save must reload Specter", f.specterLoads, loadsBefore+1)
	}
	if view := bt.View(80, 20); !strings.Contains(view, newerStamp) {
		t.Errorf("Backups view = %q, want the snapshot from the newly configured dir", view)
	}
	if r.configPanel.editing {
		t.Error("field still open after a successful save")
	}
}

func TestConfigPanel_FailedSaveKeepsFieldOpenAndAppliesNothing(t *testing.T) {
	f := &configFakes{saveErr: errors.New("disk full")}
	r, _, home := configRoot(t, f, nil)
	loadsBefore := f.specterLoads

	editRow(r, backupsRow, filepath.Join(home, "bk"))
	pump(t, r, press(r, enterKey))

	if !r.configPanel.editing || !strings.Contains(view(r), "disk full") {
		t.Errorf("editing = %v, want the field open with the save error shown", r.configPanel.editing)
	}
	if len(r.svc.Env.Saved.Targets) != 0 || f.specterLoads != loadsBefore {
		t.Errorf("Saved targets = %v, Specter loads = %d; want nothing applied", r.svc.Env.Saved.Targets, f.specterLoads-loadsBefore)
	}
}

func TestConfigPanel_ResetClearsTheSavedValue(t *testing.T) {
	f := &configFakes{}
	r, _, _ := configRoot(t, f, func(home string) state.State {
		return state.State{Targets: map[string]state.TargetState{"t": {BackupsDir: filepath.Join(home, "bk")}}}
	})
	press(r, downKey)

	pump(t, r, press(r, textKey("r")))

	if len(f.saves) != 1 || f.saves[0].Targets["t"].BackupsDir != "" {
		t.Errorf("saves = %+v, want one save with the backups dir empty", f.saves)
	}
	if got := r.svc.Env.Saved.Targets["t"].BackupsDir; got != "" {
		t.Errorf("Env.Saved backups dir = %q, want empty", got)
	}
}

func TestConfigPanel_RejectedPathKeepsFieldOpenWithReason(t *testing.T) {
	f := &configFakes{}
	r, _, _ := configRoot(t, f, nil)

	editRow(r, 0, "relative/dir")
	pump(t, r, press(r, enterKey))

	if !r.configPanel.editing || r.configPanel.err == "" || !strings.Contains(view(r), r.configPanel.err) {
		t.Errorf("editing = %v, err = %q; want the field open with the reason rendered", r.configPanel.editing, r.configPanel.err)
	}
	if len(f.saves) != 0 {
		t.Errorf("saves = %d, want none for a rejected path", len(f.saves))
	}
}

func updateRoot(t *testing.T, deps Deps) *Root {
	t.Helper()
	r := NewRoot([]target.Target{availableTarget()}, deps, nil)
	r.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	enterConfig(r)
	return r
}

func execStep(argv ...string) plan.Step { return plan.Exec{Argv: argv} }

func selfUpdatePlan() plan.Plan {
	p := plan.New("Update the TUI", nil, []plan.Step{
		execStep("env", "GIT_TERMINAL_PROMPT=0", "git", "-C", "/repo", "pull", "--ff-only"),
		execStep("bash", "/repo/scripts/bootstrap-tui.sh"),
	})
	p.Confirm = updateConfirm
	p.Notes = []string{"Updates the harness repo at /repo", "restart the TUI afterwards to run the new build"}
	return p
}

func TestConfigPanel_UpdateKeyOpensTheSelfUpdateFlow(t *testing.T) {
	r := updateRoot(t, Deps{ConfigPlanUpdate: func(context.Context, check.Env) (plan.Plan, error) { return selfUpdatePlan(), nil }})

	pump(t, r, press(r, textKey("u")))

	got := view(r)
	for _, want := range []string{updatePullStep, updateBuild, "/repo", "restart"} {
		if !strings.Contains(got, want) {
			t.Fatalf("preview lacks %q:\n%s", want, got)
		}
	}
	pump(t, r, press(r, enterKey))
	if got := view(r); !strings.Contains(got, updateConfirm) {
		t.Fatalf("enter did not open the confirmation:\n%s", got)
	}
}

func TestConfigPanel_UpdatePlannerErrorShowsInPreviewWithoutConfirm(t *testing.T) {
	r := updateRoot(t, Deps{ConfigPlanUpdate: func(context.Context, check.Env) (plan.Plan, error) {
		return plan.Plan{}, errors.New("harness repo has uncommitted changes")
	}})

	pump(t, r, press(r, textKey("u")))

	if got := view(r); !strings.Contains(got, "harness repo has uncommitted changes") || strings.Contains(got, updateConfirm) {
		t.Errorf("want the planner error and no confirmation:\n%s", got)
	}
	if isRunRequest(press(r, textKey("y"))) || isRunRequest(press(r, enterKey)) {
		t.Error("a refused update offered a run")
	}
}

func TestConfigPanel_UpdateUnderDryRunListsBothStepsNotExecuted(t *testing.T) {
	r := updateRoot(t, Deps{
		DryRun:           true,
		Runner:           dryrun.New(),
		ConfigPlanUpdate: func(context.Context, check.Env) (plan.Plan, error) { return selfUpdatePlan(), nil },
	})
	pump(t, r, press(r, textKey("u")))
	pump(t, r, press(r, enterKey))

	pump(t, r, press(r, textKey("y")))

	got := view(r)
	if !strings.Contains(got, updatePullStep) || !strings.Contains(got, updateBuild) || strings.Count(got, notExecutedText) < 2 {
		t.Errorf("want both update steps reported %q:\n%s", notExecutedText, got)
	}
}
