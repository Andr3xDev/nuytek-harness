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
)

// validatedMsg carries one store's validation back to Root; at is the instant the run was requested.
type validatedMsg struct {
	store string
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
	for _, s := range st.facts.Registered {
		sv := st.storeValidation(s.ID)
		if sv.running {
			continue
		}
		sv.running = true
		cmds = append(cmds, r.validateCmd(validate, s.ID, time.Now()))
	}
	return tea.Batch(cmds...)
}

func (r *Root) validateCmd(validate func(context.Context, string) (specter.Validation, error), id string, at time.Time) tea.Cmd {
	ctx, slots := r.ctx, r.validateSlots
	return func() tea.Msg {
		// Blocking is safe: a holder always ends, its ctx is the root one.
		slots <- struct{}{}
		defer func() { <-slots }()
		v, err := validate(ctx, id)
		return validatedMsg{store: id, at: at, v: v, err: err}
	}
}

func (r *Root) openFastInit() tea.Cmd {
	svc := r.svc
	planFn := func(ctx context.Context, names []string) (plan.Plan, error) {
		mapTo := ""
		if len(names) > 0 {
			mapTo = names[0]
		}
		return svc.SpecterPlanFastInit(ctx, mapTo)
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
		return svc.SpecterPlanMaintenance(ctx)
	})
	r.specterReport = true
	return r.specterFlow.start()
}

// handleSpecterKey serves the Specter panel; an open action flow owns the keys until it ends.
func (r *Root) handleSpecterKey(msg tea.KeyPressMsg) tea.Cmd {
	k, st := r.svc.keys, &r.specter
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
	case key.Matches(msg, k.Up):
		st.scroll.by(-1)
	case key.Matches(msg, k.Down):
		st.scroll.by(1)
	case key.Matches(msg, k.Validate):
		return r.validateStores()
	case key.Matches(msg, k.FastInit) && st.writable() && r.svc.SpecterPlanFastInit != nil:
		return r.openFastInit()
	case key.Matches(msg, k.Maintenance) && st.writable() && r.svc.SpecterPlanMaintenance != nil:
		return r.openMaintenance()
	}
	return nil
}

// specterKeys lists what the Specter panel answers to; a key whose action is unavailable is not claimed.
func (r *Root) specterKeys() []key.Binding {
	k, st := r.svc.keys, &r.specter
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
	return append(keys, k.Refresh, k.Move, k.Back)
}

// specterPanel shows the open action flow, with the drift report above a maintenance preview, or the screen.
func (r *Root) specterPanel(width, h int) []string {
	th := r.svc.th
	f := r.specterFlow
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
