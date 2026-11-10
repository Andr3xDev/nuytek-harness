package ui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	outputMargin      = 2 // cells the pane leaves free on each axis, so the outer borders stay visible
	runInProgressText = "a run is in progress"
	notExecutedText   = "not executed"
	noSnapshotText    = "snapshot: none (nothing existed)"
	runTitle          = "Output"
)

// Messages of one apply: the Cmd starts the goroutine, then the model re-arms waitEvent until applyDone.
type (
	applyStarted struct{ events <-chan tea.Msg }
	applyEvent   struct{ event plan.Event }
	applyDone    struct{ result plan.Result }
)

func startApply(ctx context.Context, runner plan.Runner, p plan.Plan) tea.Cmd {
	return func() tea.Msg {
		events := make(chan tea.Msg)
		go func() {
			defer close(events)
			result := plan.Apply(ctx, runner, p, func(e plan.Event) { events <- applyEvent{event: e} })
			events <- applyDone{result: result}
		}()
		return applyStarted{events: events}
	}
}

func waitEvent(events <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg { return <-events }
}

type outKind int

const (
	outStep outKind = iota
	outText
	outDone
	outWarn
	outFail
	outNote
)

type outLine struct {
	kind outKind
	text string
}

// run is the output pane state; its text stays raw and is wrapped (and so sanitized) when drawn.
type run struct {
	title  string
	dryRun bool
	events <-chan tea.Msg
	lines  []outLine
	done   bool
	scroll scroller
	follow bool
}

func newRun(title string, dryRun bool) *run {
	return &run{title: title, dryRun: dryRun, follow: true}
}

func (r *run) add(e plan.Event) {
	switch e.Phase {
	case plan.PhaseStart:
		r.lines = append(r.lines, outLine{outStep, fmt.Sprintf("[%d/%d] %s", e.Index, e.Total, e.Step)})
	case plan.PhaseLine:
		r.lines = append(r.lines, outLine{outText, "    " + e.Text})
	case plan.PhaseDone:
		r.lines = append(r.lines, outLine{outDone, "    done"})
	case plan.PhaseSkipped:
		text := "    skipped"
		if r.dryRun {
			text += " - " + notExecutedText
		} else if e.Text != "" {
			text += " - " + e.Text
		}
		r.lines = append(r.lines, outLine{outNote, text})
	case plan.PhaseWarned:
		r.lines = append(r.lines, outLine{outWarn, "    warned: " + e.Text})
	case plan.PhaseFailed:
		r.lines = append(r.lines, outLine{outFail, "    failed: " + e.Text})
	}
}

func (r *run) finish(res plan.Result) {
	r.done = true
	r.lines = append(r.lines, outLine{outText, ""})
	switch {
	case res.Failed != nil:
		r.lines = append(r.lines,
			outLine{outFail, "Failed at step: " + res.Failed.Step},
			outLine{outFail, fmt.Sprintf("error: %v", res.Failed.Err)})
	case r.dryRun:
		r.lines = append(r.lines, outLine{outNote, "Dry-run finished: nothing was written or executed."})
	default:
		r.lines = append(r.lines, outLine{outDone, "Finished."})
	}
	switch {
	case r.dryRun:
	case res.Stamp != "":
		r.lines = append(r.lines, outLine{outText, "snapshot: " + res.Stamp})
	default:
		r.lines = append(r.lines, outLine{outNote, noSnapshotText})
	}
	if len(res.Warned) > 0 {
		r.lines = append(r.lines, outLine{outWarn, fmt.Sprintf("Warnings (%d):", len(res.Warned))})
		for _, w := range res.Warned {
			r.lines = append(r.lines, outLine{outWarn, "  " + w})
		}
	}
}

func (r *run) scrollBy(delta int) {
	if delta < 0 {
		r.follow = false
	}
	r.scroll.by(delta)
}

func (r *run) boxTitle() string {
	title := sanitizeLine(r.title)
	if r.done {
		return runTitle + " - " + title + " (esc to close)"
	}
	return runTitle + " - " + title + " (running)"
}

func (t theme) forOut(k outKind) lipgloss.Style {
	switch k {
	case outStep:
		return t.section
	case outDone:
		return t.ok
	case outWarn:
		return t.warning
	case outFail:
		return t.err
	case outNote:
		return t.muted
	default:
		return t.plain
	}
}

func (r *run) view(th theme, w, h int) []string {
	var lines []string
	for _, l := range r.lines {
		lines = append(lines, styleAll(th.forOut(l.kind), wrap(l.text, w))...)
	}
	if r.follow {
		r.scroll.offset = len(lines)
	}
	return r.scroll.window(lines, h)
}

// outputPopup sizes the pane to the body, leaving the targets visible around it.
func outputPopup(th theme, r *run, w, bodyH int) popup {
	popupW, popupH := w-outputMargin, bodyH-outputMargin
	lines := r.view(th, contentWidth(popupW), popupH-borderCells)
	return popup{box: renderBox(th, r.boxTitle(), lines, popupW, popupH, true), w: popupW, h: popupH}
}
