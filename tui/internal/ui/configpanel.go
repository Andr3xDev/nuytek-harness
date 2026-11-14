package ui

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/config"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
)

const (
	specterRootLabel = "Specter root"
	sourceDefault    = "default"
	sourceSaved      = "saved"
	configInputPad   = "    "
	selfUpdateTitle  = "Update the TUI"
)

// configSaved carries the state that SaveState accepted; Root adopts it as the saved config.
type configSaved struct{ state state.State }

// settingRow is one editable setting. edit validates raw and returns the state to save; reset returns
// the state with the value cleared. Both leave the current saved state untouched.
type settingRow struct {
	label     string
	effective string
	source    string
	edit      func(raw string) (state.State, error)
	reset     func() state.State
}

// configState is the Config panel: a row cursor and the one text field shared by every row.
type configState struct {
	cursor  int
	editing bool
	input   textinput.Model
	err     string
}

func newConfigState() configState {
	in := textinput.New()
	in.Prompt = ""
	return configState{input: in}
}

func (c *configState) open(value string) tea.Cmd {
	c.editing, c.err = true, ""
	c.input.SetValue(value)
	c.input.CursorEnd()
	return c.input.Focus()
}

func (c *configState) close() {
	c.editing, c.err = false, ""
	c.input.Blur()
}

func sourceOf(saved string) string {
	if saved == "" {
		return sourceDefault
	}
	return sourceSaved
}

// withSetting applies set to a copy of saved whose Targets map is not shared with the original.
func withSetting(saved state.State, set func(*state.State)) state.State {
	next := saved
	next.Targets = make(map[string]state.TargetState, len(saved.Targets)+1)
	maps.Copy(next.Targets, saved.Targets)
	set(&next)
	return next
}

// settingRows is the Specter root plus one backups row per available target with a configurable home store.
func (r *Root) settingRows() []settingRow {
	env := r.svc.Env
	setRoot := func(path string) state.State {
		return withSetting(env.Saved, func(s *state.State) { s.SpecterRoot = path })
	}
	rows := []settingRow{{
		label:     specterRootLabel,
		effective: specter.ResolveRoot(env.Saved.SpecterRoot, env.Home).Path,
		source:    sourceOf(env.Saved.SpecterRoot),
		edit: func(raw string) (state.State, error) {
			path, err := config.ValidateSpecterRoot(env.FS, env.Home, raw)
			if err != nil {
				return state.State{}, err
			}
			return setRoot(path), nil
		},
		reset: func() state.State { return setRoot("") },
	}}
	for _, t := range r.targets {
		if !available(t) || t.Spec.DefaultBackupsRel == "" {
			continue
		}
		spec := t.Spec
		saved := env.Saved.Targets[spec.ID].BackupsDir
		setDir := func(dir string) state.State {
			return withSetting(env.Saved, func(s *state.State) {
				ts := s.Targets[spec.ID]
				ts.BackupsDir = dir
				s.Targets[spec.ID] = ts
			})
		}
		rows = append(rows, settingRow{
			label:     spec.Name + " backups",
			effective: spec.HomeStore(env.Home, saved),
			source:    sourceOf(saved),
			edit: func(raw string) (state.State, error) {
				path, err := config.ValidateBackupsDir(env.Home, raw, spec.OwnTree(env.Home))
				if err != nil {
					return state.State{}, err
				}
				return setDir(path), nil
			},
			reset: func() state.State { return setDir("") },
		})
	}
	return rows
}

// handleConfigKey serves the Config panel; an open field owns every key except those it hands back.
func (r *Root) handleConfigKey(msg tea.KeyPressMsg) tea.Cmd {
	k, c := r.svc.keys, &r.configPanel
	if f := r.configFlow; f != nil {
		cmd := f.Update(msg)
		if f.finished() {
			r.configFlow = nil
		}
		return cmd
	}
	rows := r.settingRows()
	row := rows[c.cursor]
	if c.editing {
		switch {
		case key.Matches(msg, k.Cancel):
			c.close()
		case key.Matches(msg, k.Accept):
			next, err := row.edit(c.input.Value())
			if err != nil {
				c.err = err.Error()
				return nil
			}
			return r.saveConfig(next)
		default:
			c.err = ""
			var cmd tea.Cmd
			c.input, cmd = c.input.Update(msg)
			return cmd
		}
		return nil
	}
	switch {
	case key.Matches(msg, k.Back):
		r.focus = focusSidebar
	case k.step(msg) != 0:
		c.cursor, c.err = max(min(c.cursor+k.step(msg), len(rows)-1), 0), ""
	case key.Matches(msg, k.Edit):
		return c.open(row.effective)
	case key.Matches(msg, k.Reset):
		return r.saveConfig(row.reset())
	case key.Matches(msg, k.UpdateTUI) && r.svc.ConfigPlanUpdate != nil:
		r.configFlow = newDirectFlow(r.svc, selfUpdateTitle, nothingToDoText, func(ctx context.Context, _ []string) (plan.Plan, error) {
			return r.svc.ConfigPlanUpdate(ctx, r.svc.Env)
		})
		return r.configFlow.start()
	}
	return nil
}

// applySavedConfig adopts a saved state and reloads everything that reads paths from it.
func (r *Root) applySavedConfig(saved state.State) tea.Cmd {
	saved.Targets = maps.Clone(saved.Targets)
	r.svc.Env.Saved = saved
	cmds := []tea.Cmd{r.loadSpecter()}
	for i := range r.targets {
		cmds = append(cmds, r.compute(i, true))
	}
	return tea.Batch(append(cmds, r.broadcast(configChanged{}))...)
}

// inputActive is true while the Config field is open, so shortcut letters are typed, not run.
func (r *Root) inputActive() bool {
	return r.entry == entryConfig && r.focus == focusPanel && r.configPanel.editing
}

// saveConfig keeps the field open and applies nothing when SaveState fails.
func (r *Root) saveConfig(next state.State) tea.Cmd {
	c := &r.configPanel
	if save := r.svc.SaveState; save != nil {
		if err := save(next); err != nil {
			c.err = err.Error()
			return nil
		}
	}
	c.close()
	return func() tea.Msg { return configSaved{state: next} }
}

func (r *Root) configKeys() []key.Binding {
	k := r.svc.keys
	if r.configFlow != nil {
		return r.configFlow.Keys()
	}
	if r.configPanel.editing {
		return []key.Binding{described(k.Accept, "save"), described(k.Cancel, "cancel")}
	}
	keys := []key.Binding{k.Edit, k.Reset}
	if r.svc.ConfigPlanUpdate != nil {
		keys = append(keys, k.UpdateTUI)
	}
	return append(keys, k.Move, k.Back)
}

func (r *Root) configLines(width, h int) []string {
	if r.configFlow != nil {
		return strings.Split(r.configFlow.View(width, h), "\n")
	}
	th, c, home := r.svc.th, &r.configPanel, r.svc.Env.Home
	rows := r.settingRows()
	labelWidth := 0
	for _, row := range rows {
		labelWidth = max(labelWidth, len(row.label))
	}
	var lines []string
	for i, row := range rows {
		text := fmt.Sprintf("%-*s  %s  (%s)", labelWidth, row.label, config.PathLabel(home, row.effective), row.source)
		lines = append(lines, selectableRow(th, text, i == c.cursor, width))
		if i != c.cursor {
			continue
		}
		if c.editing {
			in := c.input
			in.SetWidth(max(width-len(configInputPad), 1))
			lines = append(lines, configInputPad+in.View())
		}
		if c.err != "" {
			lines = append(lines, styleAll(th.err, wrap(c.err, width))...)
		}
	}
	return lines
}
