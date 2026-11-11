package claude

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	pluginsSettingsRel  = ".claude/settings.json"
	installedPluginsRel = ".claude/plugins/installed_plugins.json"
	knownMarketplaceRel = ".claude/plugins/known_marketplaces.json"
	pluginsSourceRel    = "claude/claude-settings.json"
)

const removeMatchingDescribe = "remove matching files"

var cavecrewGlobs = []string{
	".claude/plugins/cache/caveman/caveman/*/agents/cavecrew-*.md",
	".claude/plugins/cache/caveman/caveman/*/plugins/caveman/agents/cavecrew-*.md",
	".claude/plugins/marketplaces/caveman/agents/cavecrew-*.md",
	".claude/plugins/marketplaces/caveman/plugins/caveman/agents/cavecrew-*.md",
}

func pluginsComponent() manifest.Component {
	return manifest.Component{Name: "plugins", Kind: "plugins", Source: pluginsSourceRel, Target: pluginsSettingsRel}
}

func describeAll(steps []plan.Step) []string {
	out := make([]string, len(steps))
	for i, s := range steps {
		out[i] = s.Describe()
	}
	return out
}

func noEnv(string) (string, bool) { return "", false }

func TestPluginOps(t *testing.T) {
	const (
		bothMarketplaces = `{"extraKnownMarketplaces": {
			"mk-present": {"source": {"source": "github", "repo": "o/present"}},
			"mk-absent": {"source": {"source": "github", "repo": "o/absent"}}}}`
		urlMarketplace  = `{"extraKnownMarketplaces": {"mk-url": {"source": {"source": "git", "url": "https://x/y.git"}}}}`
		bothPlugins     = `{"enabledPlugins": {"p-present@m": true, "p-absent@m": true}}`
		disabledPlugin  = `{"enabledPlugins": {"p-off@m": false}}`
		declaredOne     = `{"enabledPlugins": {"p@m": true}}`
		knownPresent    = `{"mk-present": {}}`
		installedPlugin = `{"plugins": {"p-present@m": []}}`
	)
	tests := []struct {
		name      string
		settings  string
		known     *string
		installed *string
		want      []string
		wantHard  []string
		wantNote  bool
	}{
		{"marketplace present is updated", bothMarketplaces, ptr(knownPresent), nil,
			[]string{"claude plugin marketplace add o/absent", "claude plugin marketplace update mk-present"},
			[]string{"claude plugin marketplace add o/absent", "claude plugin marketplace update mk-present"}, false},
		{"marketplace without repo uses url", urlMarketplace, nil, nil,
			[]string{"claude plugin marketplace add https://x/y.git"},
			[]string{"claude plugin marketplace add https://x/y.git"}, false},
		{"plugin present is updated and absent is installed", bothPlugins, nil, ptr(installedPlugin),
			[]string{"claude plugin install p-absent@m", "claude plugin update p-present@m -y"},
			[]string{"claude plugin install p-absent@m", "claude plugin update p-present@m -y"}, false},
		{"installed undeclared plugin gets no command in install", declaredOne, nil, ptr(`{"plugins": {"mine@local": [], "p@m": []}}`),
			[]string{"claude plugin update p@m -y"},
			[]string{"claude plugin uninstall mine@local", "claude plugin update p@m -y"}, false},
		{"known undeclared marketplace is removed only in hard", declaredOne, ptr(`{"mk-old": {}}`), nil,
			[]string{"claude plugin install p@m"},
			[]string{"claude plugin marketplace remove mk-old", "claude plugin install p@m"}, false},
		{"hard orders uninstalls before marketplace removes before install steps", bothPlugins + "", ptr(`{"mk-old": {}}`), ptr(`{"plugins": {"mine@local": [], "p-present@m": []}}`),
			[]string{"claude plugin install p-absent@m", "claude plugin update p-present@m -y"},
			[]string{"claude plugin uninstall mine@local", "claude plugin marketplace remove mk-old", "claude plugin install p-absent@m", "claude plugin update p-present@m -y"}, false},
		{"disabled absent plugin is not installed", disabledPlugin, nil, nil, nil, nil, false},
		{"disabled installed plugin is not updated nor uninstalled", disabledPlugin, nil, ptr(`{"plugins": {"p-off@m": []}}`), nil, nil, false},
		{"unparseable known file is empty with a note", `{"extraKnownMarketplaces": {"mk": {"source": {"repo": "o/r"}}}}`, ptr(`{"mk": `), nil,
			[]string{"claude plugin marketplace add o/r"},
			[]string{"claude plugin marketplace add o/r"}, true},
		{"unparseable installed file is empty with a note", declaredOne, nil, ptr(`{"plugins": `),
			[]string{"claude plugin install p@m"},
			[]string{"claude plugin install p@m"}, true},
		{"missing files are empty without a note", declaredOne, nil, nil,
			[]string{"claude plugin install p@m"},
			[]string{"claude plugin install p@m"}, false},
		{"marketplace without repo or url is skipped with a note", `{"extraKnownMarketplaces": {"mk-none": {"source": {"source": "git"}}}}`, nil, nil,
			nil, nil, true},
		{"leading-dash removal key is never passed to the command", `{"enabledPlugins": {"p@m": true, "-q@m": false}, "extraKnownMarketplaces": {"-mk": {"source": {"repo": "o/r"}}}}`, ptr(`{"--scope=project": {}, "-mk": {}}`), ptr(`{"plugins": {"--scope=project": [], "p@m": []}}`),
			[]string{"claude plugin update p@m -y"},
			[]string{"claude plugin update p@m -y"}, true},
		{"leading-dash plugin id is skipped with a note", `{"enabledPlugins": {"--scope=project": true, "p@m": true}}`, nil, nil,
			[]string{"claude plugin install p@m"},
			[]string{"claude plugin install p@m"}, true},
	}
	for _, tt := range tests {
		for _, hard := range []bool{false, true} {
			name := tt.name
			want := tt.want
			planPart := installPlugins
			if hard {
				name += "/hard"
				want = tt.wantHard
				planPart = hardInstallPlugins
			}
			t.Run(name, func(t *testing.T) {
				env := newEnv(t)
				env.LookupEnv = noEnv
				writeFile(t, env.Repo, pluginsSourceRel, tt.settings)
				if tt.known != nil {
					writeFile(t, env.Home, knownMarketplaceRel, *tt.known)
				}
				if tt.installed != nil {
					writeFile(t, env.Home, installedPluginsRel, *tt.installed)
				}
				writeFile(t, env.Home, ".claude/plugins/marketplaces/caveman/agents/cavecrew-x.md", "x")

				part, err := planPart(context.Background(), env, pluginsComponent())

				if err != nil {
					t.Fatal(err)
				}
				got := describeAll(part.Steps)
				wantAll := append(slices.Clone(want), removeMatchingDescribe)
				if !slices.Equal(got, wantAll) {
					t.Fatalf("steps =\n%v\nwant\n%v", got, wantAll)
				}
				requirePluginStepShape(t, env, part)
				if (len(part.Notes) > 0) != tt.wantNote {
					t.Errorf("notes = %v, want note: %v", part.Notes, tt.wantNote)
				}
				notes := strings.Join(part.Notes, " ")
				if hard && strings.Contains(notes, "unparseable") && !strings.Contains(notes, "not removed") {
					t.Errorf("hard note %v must say undeclared plugins were not removed", part.Notes)
				}
			})
		}
	}
}

func requirePluginStepShape(t *testing.T, env check.Env, part plan.Part) {
	t.Helper()
	last, ok := part.Steps[len(part.Steps)-1].(plan.RemoveMatching)
	if !ok {
		t.Fatalf("last step is %T, want plan.RemoveMatching", part.Steps[len(part.Steps)-1])
	}
	var wantGlobs []string
	for _, g := range cavecrewGlobs {
		wantGlobs = append(wantGlobs, filepath.Join(plan.EscapeGlob(env.Home), g))
	}
	if !slices.Equal(last.Globs, wantGlobs) {
		t.Errorf("globs = %v, want %v", last.Globs, wantGlobs)
	}
	for _, step := range part.Steps[:len(part.Steps)-1] {
		exec, ok := step.(plan.Exec)
		if !ok {
			t.Fatalf("step %q is %T, want plan.Exec", step.Describe(), step)
		}
		if !exec.Warn || exec.Timeout != PluginCommandTimeout {
			t.Errorf("%q: Warn=%v Timeout=%v, want true and %v", step.Describe(), exec.Warn, exec.Timeout, PluginCommandTimeout)
		}
		isRemoval := strings.HasPrefix(step.Describe(), "claude plugin uninstall") || strings.HasPrefix(step.Describe(), "claude plugin marketplace remove")
		if isRemoval {
			want := []plan.Change{{Effect: plan.EffectDelete, Path: step.Describe(), Unrestorable: true}}
			if rows := exec.Changes(); !slices.Equal(rows, want) {
				t.Errorf("%q: rows = %+v, want %+v", step.Describe(), rows, want)
			}
		}
	}
}

func TestPlugins_CavecrewRemovalOnlyWhenAFileMatches(t *testing.T) {
	env := newEnv(t)
	env.LookupEnv = noEnv
	writeFile(t, env.Repo, pluginsSourceRel, `{"enabledPlugins": {"p@m": true}}`)

	part, err := installPlugins(context.Background(), env, pluginsComponent())

	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(describeAll(part.Steps), removeMatchingDescribe) {
		t.Errorf("steps = %v, want no cavecrew removal when nothing matches", describeAll(part.Steps))
	}
}

func TestPlugins_CavecrewGlobsEscapeHomeMetacharacters(t *testing.T) {
	env := newEnv(t)
	env.LookupEnv = noEnv
	env.Home = filepath.Join(env.Home, "h[1]*?")
	writeFile(t, env.Repo, pluginsSourceRel, `{"enabledPlugins": {"p@m": true}}`)
	writeFile(t, env.Home, ".claude/plugins/marketplaces/caveman/agents/cavecrew-x.md", "x")

	part, err := installPlugins(context.Background(), env, pluginsComponent())

	if err != nil {
		t.Fatal(err)
	}
	last, ok := part.Steps[len(part.Steps)-1].(plan.RemoveMatching)
	if !ok {
		t.Fatalf("last step is %T, want plan.RemoveMatching", part.Steps[len(part.Steps)-1])
	}
	if matches, _ := filepath.Glob(last.Globs[2]); len(matches) != 1 {
		t.Errorf("glob %q matches %v, want the one cavecrew file", last.Globs[2], matches)
	}
}

func TestPlugins_SkipFlagGivesNoStepsAndSkipped(t *testing.T) {
	for name, planPart := range map[string]plan.PartFunc{"install": installPlugins, "hard": hardInstallPlugins} {
		t.Run(name, func(t *testing.T) {
			env := newEnv(t)
			env.LookupEnv = func(k string) (string, bool) { return "1", k == "NUYTEK_SKIP_PLUGINS" }
			writeFile(t, env.Repo, pluginsSourceRel, `{"enabledPlugins": {"p@m": true}}`)

			part, err := planPart(context.Background(), env, pluginsComponent())

			if err != nil {
				t.Fatal(err)
			}
			if len(part.Steps) != 0 {
				t.Errorf("steps = %v, want none", describeAll(part.Steps))
			}
			want := []plan.Skip{{Name: "plugins", Reason: "NUYTEK_SKIP_PLUGINS=1"}}
			if !slices.Equal(part.Skipped, want) {
				t.Errorf("Skipped = %+v, want %+v", part.Skipped, want)
			}
		})
	}
}

func TestEvaluatePlugins(t *testing.T) {
	const enabled = `{"enabledPlugins": {"zeta@zeta": true, "engram@engram": true, "alpha@alpha": true}}`
	tests := []struct {
		name       string
		settings   *string
		installed  *string
		want       drift.State
		wantReason string
	}{
		{"all installed", ptr(enabled), ptr(`{"version": 2, "plugins": {"zeta@zeta": [], "engram@engram": [], "alpha@alpha": [], "extra@x": []}}`), drift.StateInstalled, ""},
		{"one missing", ptr(enabled), ptr(`{"plugins": {"zeta@zeta": [], "alpha@alpha": []}}`), drift.StateDrift, "missing: engram@engram"},
		{"several missing sorted and comma joined", ptr(enabled), ptr(`{"plugins": {"zeta@zeta": []}}`), drift.StateDrift, "missing: alpha@alpha,engram@engram"},
		{"installed file unparseable", ptr(enabled), ptr(`{"plugins": `), drift.StateUnknown, ""},
		{"disabled plugin is not required", ptr(`{"enabledPlugins": {"a@x": true, "b@x": false}}`), ptr(`{"plugins": {"a@x": []}}`), drift.StateInstalled, ""},
		{"installed file absent", ptr(`{"enabledPlugins": {"b@x": true, "a@x": true, "c@x": false}}`), nil, drift.StateDrift, "missing: a@x,b@x"},
		{"settings absent", nil, ptr(`{"plugins": {}}`), drift.StateNotInstalled, ""},
		{"only disabled plugins", ptr(`{"enabledPlugins": {"b@x": false}}`), nil, drift.StateNotInstalled, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newEnv(t)
			if tt.settings != nil {
				writeFile(t, env.Home, pluginsSettingsRel, *tt.settings)
			}
			if tt.installed != nil {
				writeFile(t, env.Home, installedPluginsRel, *tt.installed)
			}
			c := manifest.Component{Name: "plugins", Kind: "plugins", Source: "claude/claude-settings.json", Target: pluginsSettingsRel}

			got := evaluatePlugins(context.Background(), env, c)

			requireState(t, got, tt.want)
			if got.Name != "plugins" {
				t.Errorf("Name = %q, want plugins", got.Name)
			}
			if tt.wantReason != "" && got.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", got.Reason, tt.wantReason)
			}
			if tt.want == drift.StateUnknown && got.Reason == "" {
				t.Error("unknown state must carry the parse reason")
			}
		})
	}
}
