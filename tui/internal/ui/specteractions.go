package ui

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const (
	fastInitTitle    = "Specter fast init"
	fastInitPrompt   = "Map this repo to a store (optional)"
	maintenanceTitle = "Specter maintenance"
	driftReportTitle = "Drift report"
	noDriftText      = "No drift."

	createStoreTitle   = "Create store"
	createStorePrompt  = "New store id"
	addMappingTitle    = "Add repo mapping"
	changeMappingTitle = "Change repo mapping"
	removeMappingTitle = "Remove repo mapping"
	mappingPrompt      = "Repo remote or path"
	mappingStorePrompt = "Store for the mapping"
)

// validatedMsg carries one store's validation back to Root; at is the instant the run was requested and
// root the saved Specter root it ran against, so a result from before a root change is dropped.
type validatedMsg struct {
	store string
	root  string
	at    time.Time
	v     specter.Validation
	err   error
}

func (s *specterState) storeValidation(id string) *storeValidation {
	if s.validation == nil {
		s.validation = map[string]*storeValidation{}
	}
	if s.validation[id] == nil {
		s.validation[id] = &storeValidation{}
	}
	return s.validation[id]
}

func (s *specterState) finishValidation(msg validatedMsg) {
	sv := s.storeValidation(msg.store)
	sv.running, sv.err = false, msg.err
	if msg.err == nil {
		sv.records = msg.v.ChangeRecords(msg.at)
	}
}

// writable is true when the root allows the write actions (a root outside HOME is not supported).
func (s *specterState) writable() bool { return s.loaded && !s.facts.Root.OutsideHome }

// validateStores starts one background validation per registered store that is not already validating.
func (r *Root) validateStores() tea.Cmd {
	validate, st := r.svc.SpecterValidate, &r.specter
	if validate == nil || !st.loaded {
		return nil
	}
	var cmds []tea.Cmd
	root := r.svc.Env.Saved.SpecterRoot
	for _, s := range st.facts.Registered {
		sv := st.storeValidation(s.ID)
		if sv.running {
			continue
		}
		sv.running = true
		cmds = append(cmds, r.validateCmd(validate, s.ID, root, time.Now()))
	}
	return tea.Batch(cmds...)
}

func (r *Root) validateCmd(validate func(context.Context, string) (specter.Validation, error), id, root string, at time.Time) tea.Cmd {
	ctx, slots := r.ctx, r.validateSlots
	return func() tea.Msg {
		// Blocking is safe: a holder always ends, its ctx is the root one.
		slots <- struct{}{}
		defer func() { <-slots }()
		v, err := validate(ctx, id)
		return validatedMsg{store: id, root: root, at: at, v: v, err: err}
	}
}

func (r *Root) openFastInit() tea.Cmd {
	svc := r.svc
	planFn := func(ctx context.Context, names []string) (plan.Plan, error) {
		mapTo := ""
		if len(names) > 0 {
			mapTo = names[0]
		}
		return svc.SpecterPlanFastInit(ctx, r.svc.Env, mapTo)
	}
	candidates := specter.CandidateStores(r.specter.facts.Registration)
	if len(candidates) == 0 {
		r.specterFlow = newDirectFlow(svc, fastInitTitle, nothingToDoText, planFn)
	} else {
		f := newSelectFlow(svc, fastInitTitle, fastInitPrompt, nothingToDoText, planFn)
		f.limit, f.optional = 1, true
		choices := make([]choice, len(candidates))
		for i, id := range candidates {
			choices[i] = choice{name: id}
		}
		f.setChoices(choices)
		r.specterFlow = f
	}
	r.specterReport = false
	return r.specterFlow.start()
}

func (r *Root) openMaintenance() tea.Cmd {
	svc := r.svc
	r.specterFlow = newDirectFlow(svc, maintenanceTitle, nothingToDoText, func(ctx context.Context, _ []string) (plan.Plan, error) {
		return svc.SpecterPlanMaintenance(ctx, r.svc.Env)
	})
	r.specterReport = true
	return r.specterFlow.start()
}

// specterPrompt is the text field of a Specter action; submit gets the trimmed, non-empty value.
type specterPrompt struct {
	configState
	label  string
	submit func(raw string) tea.Cmd
}

func (r *Root) openSpecterPrompt(label string, submit func(raw string) tea.Cmd) tea.Cmd {
	p := &specterPrompt{configState: newConfigState(), label: label, submit: submit}
	r.specterPrompt = p
	return p.open("")
}

func (r *Root) handleSpecterPromptKey(msg tea.KeyPressMsg) tea.Cmd {
	k, p := r.svc.keys, r.specterPrompt
	switch {
	case key.Matches(msg, k.Cancel):
		r.specterPrompt = nil
	case key.Matches(msg, k.Accept):
		raw := strings.TrimSpace(p.input.Value())
		if raw == "" {
			p.err = "enter a value"
			return nil
		}
		r.specterPrompt = nil
		return p.submit(raw)
	default:
		p.err = ""
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		return cmd
	}
	return nil
}

func (r *Root) openSpecterFlow(f *flow) tea.Cmd {
	r.specterFlow, r.specterReport = f, false
	return f.start()
}

func (r *Root) openCreateStore() tea.Cmd {
	svc := r.svc
	return r.openSpecterPrompt(createStorePrompt, func(id string) tea.Cmd {
		return r.openSpecterFlow(newDirectFlow(svc, createStoreTitle, nothingToDoText, func(ctx context.Context, _ []string) (plan.Plan, error) {
			return svc.SpecterPlanCreateStore(ctx, svc.Env, id)
		}))
	})
}

func (r *Root) openRemoveMapping() tea.Cmd {
	svc := r.svc
	return r.openSpecterPrompt(mappingPrompt, func(raw string) tea.Cmd {
		return r.openSpecterFlow(newDirectFlow(svc, removeMappingTitle, nothingToDoText, func(ctx context.Context, _ []string) (plan.Plan, error) {
			return svc.SpecterPlanRemoveMapping(ctx, svc.Env, mappingEntry(raw))
		}))
	})
}

// openStoreMapping asks for the repo, then for the store: add and change differ only in the planner.
func (r *Root) openStoreMapping(title string, planFor func(ctx context.Context, entry specter.RepoEntry, store string) (plan.Plan, error)) tea.Cmd {
	svc := r.svc
	return r.openSpecterPrompt(mappingPrompt, func(raw string) tea.Cmd {
		planFn := func(ctx context.Context, names []string) (plan.Plan, error) {
			store := ""
			if len(names) > 0 {
				store = names[0]
			}
			return planFor(ctx, mappingEntry(raw), store)
		}
		registered := r.specter.facts.Registered
		if len(registered) == 0 {
			// The planner names the missing store as the blocker.
			return r.openSpecterFlow(newDirectFlow(svc, title, nothingToDoText, planFn))
		}
		f := newSelectFlow(svc, title, mappingStorePrompt, nothingToDoText, planFn)
		f.limit = 1
		choices := make([]choice, len(registered))
		for i, s := range registered {
			choices[i] = choice{name: s.ID}
		}
		f.setChoices(choices)
		return r.openSpecterFlow(f)
	})
}

// mappingEntry reads a `/` or `~` prefix as a repo path and anything else as a remote.
func mappingEntry(raw string) specter.RepoEntry {
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "~") {
		return specter.RepoEntry{Path: raw}
	}
	return specter.RepoEntry{Remote: raw}
}

// handleSpecterKey serves the Specter panel; an open action flow owns the keys until it ends.
func (r *Root) handleSpecterKey(msg tea.KeyPressMsg) tea.Cmd {
	svc, k, st := r.svc, r.svc.keys, &r.specter
	if r.specterPrompt != nil {
		return r.handleSpecterPromptKey(msg)
	}
	if f := r.specterFlow; f != nil {
		if f.stage == stageSelect && key.Matches(msg, k.Cancel) {
			r.specterFlow = nil
			return nil
		}
		cmd := f.Update(msg)
		if f.finished() {
			r.specterFlow = nil
		}
		return cmd
	}
	switch {
	case key.Matches(msg, k.Back):
		r.focus = focusSidebar
	case key.Matches(msg, k.Refresh):
		return r.loadSpecter()
	case k.step(msg) != 0:
		st.scroll.by(k.step(msg))
	case key.Matches(msg, k.Validate):
		return r.validateStores()
	case key.Matches(msg, k.FastInit) && st.writable() && r.svc.SpecterPlanFastInit != nil:
		return r.openFastInit()
	case key.Matches(msg, k.Maintenance) && st.writable() && r.svc.SpecterPlanMaintenance != nil:
		return r.openMaintenance()
	case key.Matches(msg, k.NewStore) && st.writable() && svc.SpecterPlanCreateStore != nil:
		return r.openCreateStore()
	case key.Matches(msg, k.MapAdd) && st.writable() && svc.SpecterPlanAddMapping != nil:
		return r.openStoreMapping(addMappingTitle, func(ctx context.Context, entry specter.RepoEntry, store string) (plan.Plan, error) {
			entry.Store = store
			return svc.SpecterPlanAddMapping(ctx, svc.Env, entry)
		})
	case key.Matches(msg, k.MapChange) && st.writable() && svc.SpecterPlanChangeMapping != nil:
		return r.openStoreMapping(changeMappingTitle, func(ctx context.Context, entry specter.RepoEntry, store string) (plan.Plan, error) {
			return svc.SpecterPlanChangeMapping(ctx, svc.Env, entry, store)
		})
	case key.Matches(msg, k.MapRemove) && st.writable() && svc.SpecterPlanRemoveMapping != nil:
		return r.openRemoveMapping()
	}
	return nil
}

// specterKeys lists what the Specter panel answers to; a key whose action is unavailable is not claimed.
func (r *Root) specterKeys() []key.Binding {
	svc, k, st := r.svc, r.svc.keys, &r.specter
	if r.specterPrompt != nil {
		return []key.Binding{described(k.Accept, "continue"), described(k.Cancel, "cancel")}
	}
	if f := r.specterFlow; f != nil {
		keys := f.Keys()
		if f.stage == stageSelect {
			keys = append(keys, k.Cancel)
		}
		return keys
	}
	var keys []key.Binding
	if st.loaded && r.svc.SpecterValidate != nil {
		keys = append(keys, k.Validate)
	}
	if st.writable() && r.svc.SpecterPlanFastInit != nil {
		keys = append(keys, k.FastInit)
	}
	if st.writable() && r.svc.SpecterPlanMaintenance != nil {
		keys = append(keys, k.Maintenance)
	}
	if st.writable() {
		for _, a := range []struct {
			binding key.Binding
			wired   bool
		}{
			{k.NewStore, svc.SpecterPlanCreateStore != nil},
			{k.MapAdd, svc.SpecterPlanAddMapping != nil},
			{k.MapChange, svc.SpecterPlanChangeMapping != nil},
			{k.MapRemove, svc.SpecterPlanRemoveMapping != nil},
		} {
			if a.wired {
				keys = append(keys, a.binding)
			}
		}
	}
	return append(keys, k.Refresh, k.Move, k.Back)
}

// specterPanel shows the open action flow, with the drift report above a maintenance preview, or the screen.
func (r *Root) specterPanel(width, h int) []string {
	th := r.svc.th
	f := r.specterFlow
	if p := r.specterPrompt; p != nil {
		in := p.input
		in.SetWidth(max(width-len(configInputPad), 1))
		lines := []string{th.section.Render(p.label), configInputPad + in.View()}
		if p.err != "" {
			lines = append(lines, styleAll(th.err, wrap(p.err, width))...)
		}
		return clip(lines, h)
	}
	if f == nil {
		return r.specter.scroll.window(specterScreen(th, &r.specter, width), h)
	}
	var head []string
	if r.specterReport {
		head = append([]string{th.section.Render(driftReportTitle)}, driftReportLines(th, r.specter.facts, width)...)
		head = append(head, "")
	}
	return clip(append(head, strings.Split(f.View(width, h-len(head)), "\n")...), h)
}

func driftReportLines(th theme, f specter.Facts, width int) []string {
	drift := specter.Drift(f)
	if len(drift) == 0 {
		return []string{th.muted.Render(noDriftText)}
	}
	return resultLines(th, drift, width)
}
