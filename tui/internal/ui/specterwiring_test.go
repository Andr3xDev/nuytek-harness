package ui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const (
	typedRemote = "git@example.test:org/new.git"
	typedStore  = "gamma"
)

func typeText(r *Root, s string) {
	for _, c := range s {
		press(r, textKey(string(c)))
	}
}

// readyFacts is baseFacts with a Specter repo whose creation tools all work.
func readyFacts() specter.Facts {
	f := baseFacts()
	f.IsRepo = true
	f.Folders = []specter.StoreFolder{{ID: storeAlpha}}
	f.Git = specter.ToolInfo{Found: true}
	f.Gh = specter.GhInfo{Found: true, Authenticated: true}
	return f
}

func TestSpecterScreen_BlockedFastInitShowsCannotRunAndNoPlan(t *testing.T) {
	fakes := &specterFakes{fastInitErr: specter.ErrBlocked{Reason: "openspec not found on PATH"}}
	r := screenRoot(t, readyFacts(), fakes)

	press(r, textKey("f"))
	deliver(r, press(r, enterKey))
	got := view(r)

	if !strings.Contains(got, "Cannot run: openspec not found on PATH") {
		t.Errorf("blocked reason not shown:\n%s", got)
	}
	if strings.Contains(got, registerDirRow) || r.specterFlow.canApply() {
		t.Errorf("a plan opened for a blocked action:\n%s", got)
	}
}

func TestSpecterScreen_CreateStoreAsksIdThenPreviewsPlan(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, readyFacts(), fakes)

	press(r, textKey("n"))
	typeText(r, typedStore)
	deliver(r, press(r, enterKey))

	if !reflect.DeepEqual(fakes.createdStores, []string{typedStore}) {
		t.Errorf("planned ids = %v, want [%s]", fakes.createdStores, typedStore)
	}
	if !strings.Contains(view(r), registerDirRow) {
		t.Errorf("preview missing the planned steps:\n%s", view(r))
	}
}

func TestSpecterScreen_CreateStoreFailureIsShownAndOpensNoPlan(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"blocked", specter.ErrBlocked{Reason: "openspec not found on PATH"}, "Cannot run: openspec not found on PATH"},
		{"plain error", errors.New("store id \"gamma\" is already used"), "already used"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := screenRoot(t, readyFacts(), &specterFakes{createStoreErr: tt.err})

			press(r, textKey("n"))
			typeText(r, typedStore)
			deliver(r, press(r, enterKey))
			got := view(r)

			if !strings.Contains(got, tt.want) {
				t.Errorf("%q not shown:\n%s", tt.want, got)
			}
			if strings.Contains(got, registerDirRow) || r.specterFlow.canApply() {
				t.Errorf("a plan opened for a failed create:\n%s", got)
			}
		})
	}
}

func TestSpecterScreen_EmptyPromptIsRejectedAndEscCancels(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, readyFacts(), fakes)

	press(r, textKey("n"))
	press(r, enterKey)
	if r.specterPrompt == nil || len(fakes.createdStores) != 0 {
		t.Fatalf("empty id was accepted (prompt open %v, planned %v)", r.specterPrompt != nil, fakes.createdStores)
	}
	press(r, escKey)

	if r.specterPrompt != nil || r.inputActive() {
		t.Error("esc left the prompt open")
	}
}

func TestSpecterScreen_AddMappingAsksRepoThenStore(t *testing.T) {
	tests := []struct {
		name  string
		typed string
		want  specter.RepoEntry
	}{
		{"remote", typedRemote, specter.RepoEntry{Remote: typedRemote, Store: storeAlpha}},
		{"path", "~/work/app", specter.RepoEntry{Path: "~/work/app", Store: storeAlpha}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakes := &specterFakes{}
			r := screenRoot(t, readyFacts(), fakes)

			press(r, textKey("a"))
			typeText(r, tt.typed)
			press(r, enterKey)
			press(r, textKey(" "))
			deliver(r, press(r, enterKey))

			if !reflect.DeepEqual(fakes.addedMappings, []specter.RepoEntry{tt.want}) {
				t.Errorf("planned %+v, want %+v", fakes.addedMappings, tt.want)
			}
			if !strings.Contains(view(r), registerDirRow) {
				t.Errorf("preview missing the planned steps:\n%s", view(r))
			}
		})
	}
}

func TestSpecterScreen_ChangeMappingOpensPreview(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, readyFacts(), fakes)

	press(r, textKey("c"))
	typeText(r, typedRemote)
	press(r, enterKey)
	press(r, textKey(" "))
	deliver(r, press(r, enterKey))

	if want := []specter.RepoEntry{{Remote: typedRemote, Store: storeAlpha}}; !reflect.DeepEqual(fakes.changedMappings, want) {
		t.Errorf("changed %+v, want %+v", fakes.changedMappings, want)
	}
	if !strings.Contains(view(r), registerDirRow) {
		t.Errorf("change did not open a preview with the planned steps:\n%s", view(r))
	}
}

func TestSpecterScreen_RemoveMappingOpensPreview(t *testing.T) {
	fakes := &specterFakes{}
	r := screenRoot(t, readyFacts(), fakes)

	press(r, textKey("x"))
	typeText(r, typedRemote)
	deliver(r, press(r, enterKey))

	if want := []specter.RepoEntry{{Remote: typedRemote}}; !reflect.DeepEqual(fakes.removedMappings, want) {
		t.Errorf("removed %+v, want %+v", fakes.removedMappings, want)
	}
	if !strings.Contains(view(r), registerDirRow) {
		t.Errorf("remove did not open a preview with the planned steps:\n%s", view(r))
	}
}

func TestSpecterScreen_MappingWithoutRegisteredStoreIsBlockedByThePlanner(t *testing.T) {
	facts := readyFacts()
	facts.Registered = nil
	fakes := &specterFakes{}
	r := screenRoot(t, facts, fakes)

	press(r, textKey("a"))
	typeText(r, typedRemote)
	deliver(r, press(r, enterKey))

	if len(fakes.addedMappings) != 1 || fakes.addedMappings[0].Store != "" {
		t.Errorf("planner calls = %+v, want one call with no store so it can name the blocker", fakes.addedMappings)
	}
}

func TestSpecterScreen_MissingRootGuidance(t *testing.T) {
	tests := []struct {
		name string
		edit func(*specter.Facts)
		want string
	}{
		{"fast init available", func(*specter.Facts) {}, "Press f to create it from the template"},
		{"gh missing", func(f *specter.Facts) { f.Gh = specter.GhInfo{} }, "gh not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := readyFacts()
			facts.RootExists, facts.IsRepo = false, false
			tt.edit(&facts)

			got := view(screenRoot(t, facts, &specterFakes{}))

			if !strings.Contains(got, "Root not found") || !strings.Contains(got, tt.want) {
				t.Errorf("guidance lacks %q:\n%s", tt.want, got)
			}
			if strings.Contains(got, "git clone") || strings.Contains(got, "NUYTEK_SPECTER_REPO") {
				t.Errorf("clone hint shown:\n%s", got)
			}
		})
	}
}

func TestSpecterScreen_NotARepoNoticeOnlyWithoutReposYAML(t *testing.T) {
	for _, isRepo := range []bool{false, true} {
		facts := readyFacts()
		facts.IsRepo = isRepo

		got := strings.Contains(view(screenRoot(t, facts, &specterFakes{})), notSpecterRepoText)

		if got == isRepo {
			t.Errorf("IsRepo=%v: notice shown = %v", isRepo, got)
		}
	}
}

func TestSpecterScreen_EmptyStateHintOnlyForRepoWithoutStores(t *testing.T) {
	tests := []struct {
		name    string
		folders []specter.StoreFolder
		want    bool
	}{
		{"fresh repo", nil, true},
		{"unregistered store folder counts as a store", []specter.StoreFolder{{ID: "loose"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facts := readyFacts()
			facts.Folders = tt.folders

			got := strings.Contains(view(screenRoot(t, facts, &specterFakes{})), "No stores yet")

			if got != tt.want {
				t.Errorf("hint shown = %v, want %v", got, tt.want)
			}
		})
	}
}

// renderedManifest gives target 0 one component per name, deployed at .claude/<name>, and lists a file in
// targets/rendered-files.txt for each listed name; the reinstall offer must follow that list, not fixed names.
func renderedManifest(t *testing.T, r *Root, listed []string, unlisted ...string) {
	t.Helper()
	var lines []string
	var components []manifest.Component
	for _, name := range append(slices.Clone(listed), unlisted...) {
		components = append(components, manifest.Component{Name: name, Target: ".claude/" + name})
		if slices.Contains(listed, name) {
			lines = append(lines, ".claude/"+name+"/file.md")
		}
	}
	list := filepath.Join(r.svc.Env.Repo, "targets", "rendered-files.txt")
	if err := os.MkdirAll(filepath.Dir(list), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(list, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.targets[0].Manifest = manifest.Manifest{Components: components}
}

func TestConfigPanel_RootChangeOffersReinstallLimitedToRenderedComponents(t *testing.T) {
	r, _, home := configRoot(t, &configFakes{}, nil)
	renderedManifest(t, r, []string{"agents", "commands", "fresh"}, "hooks")

	editRow(r, 0, home)
	press(r, enterKey)
	if !r.configPanel.offer {
		t.Fatalf("no reinstall offer after a root change:\n%s", plain(r.render()))
	}
	if got := reinstallNames(r.svc.Env, r.targets[0]); !reflect.DeepEqual(got, []string{"agents", "commands", "fresh"}) {
		t.Errorf("reinstall components = %v, want the components with a listed file (including one no fixed list knows) and not hooks", got)
	}

	press(r, textKey("y"))

	if r.configFlow == nil || r.configFlow.title != reinstallTitle || r.configPanel.offer {
		t.Errorf("accepting the offer did not open the reinstall flow (flow %v)", r.configFlow)
	}
}

func TestConfigPanel_DecliningReinstallOffersNothingMore(t *testing.T) {
	r, _, home := configRoot(t, &configFakes{}, nil)
	renderedManifest(t, r, []string{"agents"})

	editRow(r, 0, home)
	press(r, enterKey)
	press(r, escKey)

	if r.configPanel.offer || r.configFlow != nil {
		t.Errorf("declined offer left state: offer %v, flow %v", r.configPanel.offer, r.configFlow)
	}
}

func TestConfigPanel_SavingTheSameRootOffersNothing(t *testing.T) {
	r, _, _ := configRoot(t, &configFakes{}, nil)
	renderedManifest(t, r, []string{"agents"})

	editRow(r, 0, specter.DefaultRoot(r.svc.Env.Home))
	press(r, enterKey)

	if r.configPanel.offer {
		t.Error("offer shown although the resolved root did not change")
	}
}
