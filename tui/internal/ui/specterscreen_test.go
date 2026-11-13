package ui

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	storeAlpha     = "alpha-store"
	changeEdited   = "change-edited"
	changeInvalid  = "change-invalid"
	changeClean    = "change-clean"
	registerDirRow = "register-beta-dir"
)

// specterFakes replaces the OS-facing Deps of the Specter screen; none of them touches openspec, HOME or exec.
type specterFakes struct {
	validateCalls   int
	validateCtxErr  []error
	validation      specter.Validation
	validateErr     error
	fastInitMapTo   []string
	maintenanceRuns int
}

func (f *specterFakes) deps(facts specter.Facts) Deps {
	registerPlan := plan.Plan{Steps: []plan.Step{plan.Copy{Rows: []plan.Change{{Effect: plan.EffectRun, Path: registerDirRow}}}}}
	return Deps{
		SpecterLoad: func(context.Context) specter.Facts { return facts },
		SpecterValidate: func(ctx context.Context, id string) (specter.Validation, error) {
			f.validateCalls++
			f.validateCtxErr = append(f.validateCtxErr, ctx.Err())
			return f.validation, f.validateErr
		},
		SpecterPlanFastInit: func(_ context.Context, mapTo string) (plan.Plan, error) {
			f.fastInitMapTo = append(f.fastInitMapTo, mapTo)
			return registerPlan, nil
		},
		SpecterPlanMaintenance: func(context.Context) (plan.Plan, error) {
			f.maintenanceRuns++
			return registerPlan, nil
		},
	}
}

func baseFacts() specter.Facts {
	return specter.Facts{
		Registration: specter.Registration{
			Root:         specter.Root{Path: "/home/u/dev/specter", Source: specter.RootSourceDefault},
			CloneCommand: "git clone URL /home/u/dev/specter",
			RootExists:   true,
			Openspec:     specter.ToolInfo{Found: true},
			Registered:   []specter.StoreInfo{{ID: storeAlpha, Root: "/home/u/dev/specter/alpha"}},
		},
		OpenspecVersion: "9.8.7",
		Repos: specter.ReposView{
			Repos:  []specter.RepoEntry{{Remote: "git@example.test:org/widget.git", Store: storeAlpha}},
			Stores: map[string]specter.StoreMapping{storeAlpha: {Engram: []string{"widget-memory"}}},
		},
		Stores: map[string]specter.StoreStats{storeAlpha: {
			Active: 13, Archived: 11, Specs: 17,
			Changes: []specter.ChangeInfo{
				{Name: changeClean, LastModified: "2000-01-01T00:00:00Z"},
				{Name: changeEdited, LastModified: "2999-01-01T00:00:00Z"},
				{Name: changeInvalid, LastModified: "2000-01-01T00:00:00Z"},
			},
		}},
		Home: "/home/u",
	}
}

// screenRoot has the Specter panel focused with facts already loaded.
func screenRoot(t *testing.T, facts specter.Facts, fakes *specterFakes) *Root {
	t.Helper()
	tgt := availableTarget()
	r := NewRoot([]target.Target{tgt}, fakes.deps(facts), nil)
	r.Update(tea.WindowSizeMsg{Width: 160, Height: 200})
	selectSpecter(r)
	r.Update(specterLoadedMsg{gen: r.specter.gen, facts: facts})
	press(r, enterKey)
	return r
}

// deliver runs cmd and feeds every message back to r, the way the Bubble Tea loop would.
func deliver(r *Root, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case nil:
	case tea.BatchMsg:
		for _, c := range msg {
			deliver(r, c)
		}
	case runRequest:
	default:
		_, next := r.Update(msg)
		deliver(r, next)
	}
}

func view(r *Root) string { return plain(r.render()) }

func lineWith(s, sub string) string {
	for _, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			return l
		}
	}
	return ""
}

func lineIndex(s, sub string) int {
	for i, l := range strings.Split(s, "\n") {
		if strings.Contains(l, sub) {
			return i
		}
	}
	return -1
}

func assertRendered(t *testing.T, got string, results []check.Result) {
	t.Helper()
	if len(results) == 0 {
		t.Fatal("fixture produced no results")
	}
	for _, res := range results {
		line := lineWith(got, res.Message)
		if line == "" {
			t.Errorf("message %q not rendered", res.Message)
			continue
		}
		if !strings.Contains(line, statusGlyph(res.Status)) {
			t.Errorf("line %q lacks glyph %q of status %s", line, statusGlyph(res.Status), res.Status)
		}
		if res.Suggestion != "" && !strings.Contains(lineWith(got, res.Suggestion), "Suggestion") {
			t.Errorf("suggestion %q not rendered on a Suggestion line", res.Suggestion)
		}
	}
}

func TestSpecterScreen_RootBlockShowsPathAndSource(t *testing.T) {
	facts := baseFacts()
	facts.Root = specter.Root{Path: "/home/u/custom-root", Source: specter.RootSourceEnv}
	got := view(screenRoot(t, facts, &specterFakes{}))

	if !strings.Contains(got, "/home/u/custom-root") || !strings.Contains(got, string(specter.RootSourceEnv)) {
		t.Fatalf("root path and source missing:\n%s", got)
	}
}

func TestSpecterScreen_MissingRootShowsCloneCommand(t *testing.T) {
	facts := baseFacts()
	facts.RootExists = false
	got := view(screenRoot(t, facts, &specterFakes{}))

	if !strings.Contains(got, facts.CloneCommand) {
		t.Fatalf("clone command %q missing:\n%s", facts.CloneCommand, got)
	}
}

func TestSpecterScreen_RootBlockListsConfigsWithCopyAndNotFound(t *testing.T) {
	facts := baseFacts()
	facts.RootRefs = []specter.RootRef{
		{Label: "guard hook", Copy: specter.CopyDeployed, Found: true, References: true},
		{Label: "SDD rules"},
	}
	got := view(screenRoot(t, facts, &specterFakes{}))

	if line := lineWith(got, "guard hook"); !strings.Contains(line, string(specter.CopyDeployed)) {
		t.Errorf("guard hook row %q lacks the copy read", line)
	}
	if line := lineWith(got, "SDD rules"); !strings.Contains(line, "not found") {
		t.Errorf("SDD rules row %q lacks \"not found\"", line)
	}
}

func TestSpecterScreen_OverviewShowsStoresMappingsAndCounts(t *testing.T) {
	got := view(screenRoot(t, baseFacts(), &specterFakes{}))

	for _, want := range []string{"/home/u/dev/specter/alpha", "git@example.test:org/widget.git", "widget-memory", "13", "11", "17"} {
		if !strings.Contains(got, want) {
			t.Errorf("overview lacks %q:\n%s", want, got)
		}
	}
}

func TestSpecterScreen_UnreadableReposShowsUnknownAndKeepsCounts(t *testing.T) {
	facts := baseFacts()
	facts.Repos = specter.ReposView{Err: errors.New("bad yaml")}
	got := view(screenRoot(t, facts, &specterFakes{}))

	if !strings.Contains(got, "unknown") || !strings.Contains(got, "17") || !strings.Contains(got, "/home/u/dev/specter/alpha") {
		t.Fatalf("want unknown mapping with stores and counts still rendered:\n%s", got)
	}
}

func TestSpecterScreen_FailedStoreQueryShowsUnknownCounts(t *testing.T) {
	facts := baseFacts()
	facts.Stores = map[string]specter.StoreStats{storeAlpha: {Err: errors.New("list failed")}}
	got := view(screenRoot(t, facts, &specterFakes{}))

	if !strings.Contains(got, "unknown") || !strings.Contains(got, "widget-memory") {
		t.Fatalf("want unknown counts with the mapping still rendered:\n%s", got)
	}
}

func TestSpecterScreen_ChecksRenderGlyphMessageAndSuggestion(t *testing.T) {
	facts := baseFacts()
	facts.Repos.Stores["beta-store"] = specter.StoreMapping{}
	facts.Repos.Repos = nil
	got := view(screenRoot(t, facts, &specterFakes{}))

	assertRendered(t, got, specter.Checks(facts))
}

func TestSpecterScreen_SkippedChecksShowReason(t *testing.T) {
	facts := baseFacts()
	facts.Openspec = specter.ToolInfo{}
	facts.RegisteredErr = errors.New("openspec not found on PATH")
	results := specter.Checks(facts)
	if !hasStatus(results, check.StatusSkipped) {
		t.Fatal("fixture has no skipped check")
	}

	assertRendered(t, view(screenRoot(t, facts, &specterFakes{})), results)
}

func hasStatus(results []check.Result, s check.Status) bool {
	for _, r := range results {
		if r.Status == s {
			return true
		}
	}
	return false
}

func TestSpecterScreen_DriftEntriesShowMessageAndCommand(t *testing.T) {
	facts := baseFacts()
	facts.Folders = []specter.StoreFolder{{Dir: "/home/u/dev/specter/gamma", ID: "gamma-store"}}
	facts.Registered = append(facts.Registered, specter.StoreInfo{ID: "gamma-store", Root: "/elsewhere/gamma"})
	got := view(screenRoot(t, facts, &specterFakes{}))

	drift := specter.Drift(facts)
	assertRendered(t, got, drift)
	if !strings.Contains(got, "/specter") {
		t.Fatal("no /specter command rendered for the drift entry")
	}
}

func validFixture() specter.Validation {
	return specter.Validation{Items: []specter.ValidationItem{
		{ID: changeClean, Type: "change", Valid: true},
		{ID: changeEdited, Type: "change", Valid: true},
		{ID: changeInvalid, Type: "change", Valid: false, Issues: []specter.Issue{{Level: "ERROR", Message: "missing scenario section"}}},
	}}
}

func validatedMark(r *Root, change string) bool {
	return strings.Contains(lineWith(view(r), change), statusGlyph(check.StatusOK))
}

func TestSpecterScreen_OpeningAndRefreshNeverValidate(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, baseFacts(), fakes)

	deliver(r, press(r, textKey("r")))

	if fakes.validateCalls != 0 {
		t.Fatalf("validate ran %d times without v", fakes.validateCalls)
	}
	if validatedMark(r, changeClean) {
		t.Fatal("a change reads validated before any validation")
	}
}

func TestSpecterScreen_ValidateMarksOnlyValidUneditedChanges(t *testing.T) {
	fakes := &specterFakes{validation: validFixture()}
	r := screenRoot(t, baseFacts(), fakes)

	deliver(r, press(r, textKey("v")))

	if fakes.validateCalls != 1 {
		t.Fatalf("validate calls = %d, want 1", fakes.validateCalls)
	}
	if !validatedMark(r, changeClean) {
		t.Error("valid unedited change is not marked validated")
	}
	if validatedMark(r, changeEdited) {
		t.Error("change modified after the run started is marked validated")
	}
	if validatedMark(r, changeInvalid) {
		t.Error("invalid change is marked validated")
	}
	if !strings.Contains(view(r), "missing scenario section") {
		t.Error("issue of the invalid change not shown")
	}
}

func TestSpecterScreen_ValidateIgnoredWhileThatStoreValidates(t *testing.T) {
	fakes := &specterFakes{validation: validFixture()}
	r := screenRoot(t, baseFacts(), fakes)

	first := press(r, textKey("v"))
	deliver(r, press(r, textKey("v")))
	deliver(r, first)

	if fakes.validateCalls != 1 {
		t.Fatalf("validate calls = %d, want 1 (second v ignored while running)", fakes.validateCalls)
	}
	deliver(r, press(r, textKey("v")))
	if fakes.validateCalls != 2 {
		t.Fatalf("validate calls = %d after the run ended, want 2", fakes.validateCalls)
	}
}

func TestSpecterScreen_ValidationUsesRootContextAndQuitIsNotRefused(t *testing.T) {
	fakes := &specterFakes{validation: validFixture()}
	r := screenRoot(t, baseFacts(), fakes)

	cmd := press(r, textKey("v"))
	if !isQuit(press(r, textKey("q"))) {
		t.Fatal("q during validation did not quit")
	}
	deliver(r, cmd)

	if len(fakes.validateCtxErr) != 1 || fakes.validateCtxErr[0] == nil {
		t.Fatalf("validation ctx errors = %v, want a context cancelled with the root", fakes.validateCtxErr)
	}
}

func TestSpecterScreen_ValidatedIndicatorSurvivesRefresh(t *testing.T) {
	fakes := &specterFakes{validation: validFixture()}
	facts := baseFacts()
	r := screenRoot(t, facts, fakes)
	deliver(r, press(r, textKey("v")))

	deliver(r, press(r, textKey("r")))
	r.Update(specterLoadedMsg{gen: r.specter.gen, facts: facts})

	if !validatedMark(r, changeClean) {
		t.Fatal("refresh dropped the session validation record")
	}
}

func TestSpecterScreen_ValidationFailureShowsErrorAndNoIndicator(t *testing.T) {
	fakes := &specterFakes{validateErr: errors.New("openspec exploded")}
	r := screenRoot(t, baseFacts(), fakes)

	deliver(r, press(r, textKey("v")))

	if !strings.Contains(view(r), "openspec exploded") || validatedMark(r, changeClean) {
		t.Fatalf("want the error shown and no validated change:\n%s", view(r))
	}
}

func facts2Candidates() specter.Facts {
	facts := baseFacts()
	facts.Folders = []specter.StoreFolder{{Dir: "/home/u/dev/specter/beta", ID: "beta-store"}}
	return facts
}

func TestSpecterScreen_FastInitSelectsOneOptionalStoreThenPlans(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, facts2Candidates(), fakes)

	press(r, textKey("f"))
	keys, _ := r.keybarKeys()
	if !claims(keys, "space") || claims(keys, "a") {
		t.Fatalf("selection keys = %v, want toggle without all/none", keys)
	}
	if !strings.Contains(view(r), "beta-store") {
		t.Fatal("candidate store not listed")
	}

	press(r, textKey(" "))
	deliver(r, press(r, enterKey))

	if len(fakes.fastInitMapTo) != 1 || fakes.fastInitMapTo[0] != storeAlpha {
		t.Fatalf("PlanFastInit mapTo = %v, want [%s]", fakes.fastInitMapTo, storeAlpha)
	}
	if !strings.Contains(view(r), registerDirRow) {
		t.Fatalf("preview missing:\n%s", view(r))
	}
}

func TestSpecterScreen_FastInitWithoutPickPlansWithEmptyMapping(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, facts2Candidates(), fakes)

	press(r, textKey("f"))
	deliver(r, press(r, enterKey))

	if len(fakes.fastInitMapTo) != 1 || fakes.fastInitMapTo[0] != "" {
		t.Fatalf("PlanFastInit mapTo = %v, want [\"\"]", fakes.fastInitMapTo)
	}
}

func TestSpecterScreen_FastInitWithoutCandidatesPlansAtOnce(t *testing.T) {
	facts := baseFacts()
	facts.Registered = nil
	fakes := &specterFakes{}
	r := screenRoot(t, facts, fakes)

	deliver(r, press(r, textKey("f")))

	if len(fakes.fastInitMapTo) != 1 || fakes.fastInitMapTo[0] != "" {
		t.Fatalf("PlanFastInit mapTo = %v, want one call with no mapping", fakes.fastInitMapTo)
	}
}

func TestSpecterScreen_PreviewEnterAppliesWithoutConfirmAndEscCancels(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, baseFacts(), fakes)
	deliver(r, press(r, textKey("m")))

	if cmd := press(r, enterKey); !isRunRequest(cmd) {
		t.Fatal("enter in the preview did not request a run")
	}

	deliver(r, press(r, textKey("m")))
	if cmd := press(r, escKey); isRunRequest(cmd) || strings.Contains(view(r), registerDirRow) {
		t.Fatal("esc applied or left the preview open")
	}
}

func TestSpecterScreen_MaintenanceShowsDriftReportAbovePreview(t *testing.T) {
	facts := baseFacts()
	facts.Folders = []specter.StoreFolder{{Dir: "/home/u/dev/specter/gamma", ID: "gamma-store"}}
	facts.Registered = append(facts.Registered, specter.StoreInfo{ID: "gamma-store", Root: "/elsewhere/gamma"})
	drift := specter.Drift(facts)
	if len(drift) == 0 {
		t.Fatal("fixture has no drift")
	}
	fakes := &specterFakes{}
	r := screenRoot(t, facts, fakes)

	deliver(r, press(r, textKey("m")))

	got := view(r)
	if fakes.maintenanceRuns != 1 {
		t.Fatalf("PlanMaintenance calls = %d, want 1", fakes.maintenanceRuns)
	}
	if d, p := lineIndex(got, drift[0].Message), lineIndex(got, registerDirRow); d < 0 || p < 0 || d > p {
		t.Fatalf("drift line %d, preview line %d; want the drift report above the preview:\n%s", d, p, got)
	}
}

func TestSpecterScreen_ActionsHiddenWhenRootOutsideHome(t *testing.T) {
	facts := baseFacts()
	facts.Root = specter.Root{Path: "/srv/specter", Source: specter.RootSourceEnv, OutsideHome: true}
	fakes := &specterFakes{}
	r := screenRoot(t, facts, fakes)

	deliver(r, press(r, textKey("f")))
	deliver(r, press(r, textKey("m")))

	keys, _ := r.keybarKeys()
	if len(fakes.fastInitMapTo) != 0 || fakes.maintenanceRuns != 0 || claims(keys, "f") || claims(keys, "m") {
		t.Fatalf("actions available outside HOME: fastInit %v, maintenance %d, keys %v", fakes.fastInitMapTo, fakes.maintenanceRuns, keys)
	}
	if !strings.Contains(view(r), "not supported") {
		t.Fatalf("outside-HOME statement missing:\n%s", view(r))
	}
}

func TestSpecterScreen_KeybarOffersSpecterKeysWithMaintenanceHelp(t *testing.T) {
	r := screenRoot(t, baseFacts(), &specterFakes{})

	keys, _ := r.keybarKeys()
	for _, k := range []string{"v", "f", "m", "r", "esc"} {
		if !claims(keys, k) {
			t.Errorf("Specter keybar lacks %q: %v", k, keys)
		}
	}
	for _, b := range keys {
		if len(b.Keys()) > 0 && b.Keys()[0] == "m" && b.Help().Desc != "maintenance" {
			t.Errorf("m is described %q, want maintenance", b.Help().Desc)
		}
	}
}

var modeHelp = regexp.MustCompile(`\bm\s+mode\b`)

func TestSpecterScreen_HelpShowsMaintenanceNotWizardMode(t *testing.T) {
	r := screenRoot(t, baseFacts(), &specterFakes{})
	press(r, textKey("?"))

	got := view(r)
	if !strings.Contains(got, "maintenance") || modeHelp.MatchString(got) {
		t.Fatalf("help on the Specter screen must show m maintenance and not m mode:\n%s", got)
	}
}
