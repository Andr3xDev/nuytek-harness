package ui

import (
	"context"
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const (
	outputMargin      = 2 // cells the pane leaves free on each axis, so the outer borders stay visible
	maxOutputWidth    = 72
	runningFloorRows  = 3 // content rows a running pane keeps while little output has arrived
	maxOutputLines    = 2000
	closeHintText     = "Press esc to close."
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
	title      string
	dryRun     bool
	events     <-chan tea.Msg
	cancel     context.CancelFunc // stops the apply; its goroutine then reports applyDone
	cancelling bool
	lines      []outLine  // capped at maxOutputLines, oldest dropped first
	wrapped    [][]string // styled and wrapped lines[:len(wrapped)] at wrappedWidth
	wrapWidth  int
	done       bool
	scroll     scroller
	follow     bool
}

func newRun(title string, dryRun bool) *run {
	return &run{title: title, dryRun: dryRun, follow: true}
}

func (r *run) push(lines ...outLine) {
	r.lines = append(r.lines, lines...)
	if drop := len(r.lines) - maxOutputLines; drop > 0 {
		r.lines = slices.Delete(r.lines, 0, drop)
		r.wrapped = r.wrapped[min(drop, len(r.wrapped)):]
	}
}

// requestCancel stops the apply once; the pane stays open until applyDone arrives.
func (r *run) requestCancel() {
	r.cancelling = true
	r.cancel()
}

func (r *run) add(e plan.Event) {
	switch e.Phase {
	case plan.PhaseStart:
		r.push(outLine{outStep, fmt.Sprintf("[%d/%d] %s", e.Index, e.Total, e.Step)})
	case plan.PhaseLine:
		r.push(outLine{outText, "    " + e.Text})
	case plan.PhaseDone:
		r.push(outLine{outDone, "    done"})
	case plan.PhaseSkipped:
		text := "    skipped"
		if r.dryRun {
			text += " - " + notExecutedText
		} else if e.Text != "" {
			text += " - " + e.Text
		}
		r.push(outLine{outNote, text})
	case plan.PhaseWarned:
		r.push(outLine{outWarn, "    warned: " + e.Text})
	case plan.PhaseFailed:
		r.push(outLine{outFail, "    failed: " + e.Text})
	}
}

func (r *run) finish(res plan.Result) {
	r.done = true
	r.push(outLine{outText, ""})
	switch {
	case res.Failed != nil:
		r.push(
			outLine{outFail, "Failed at step: " + res.Failed.Step},
			outLine{outFail, fmt.Sprintf("error: %v", res.Failed.Err)})
	case r.dryRun:
		r.push(outLine{outNote, "Dry-run finished: nothing was written or executed."})
	default:
		r.push(outLine{outDone, "Finished."})
	}
	switch {
	case r.dryRun:
	case res.Stamp != "":
		r.push(outLine{outText, "snapshot: " + res.Stamp})
	default:
		r.push(outLine{outNote, noSnapshotText})
	}
	if len(res.Warned) > 0 {
		r.push(outLine{outWarn, fmt.Sprintf("Warnings (%d):", len(res.Warned))})
		for _, w := range res.Warned {
			r.push(outLine{outWarn, "  " + w})
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
	switch {
	case r.done:
		return runTitle + " - " + title
	case r.cancelling:
		return runTitle + " - " + title + " (cancelling...)"
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
	if w != r.wrapWidth {
		r.wrapped, r.wrapWidth = nil, w
	}
	for len(r.wrapped) < len(r.lines) {
		l := r.lines[len(r.wrapped)]
		r.wrapped = append(r.wrapped, styleAll(th.forOut(l.kind), wrap(l.text, w)))
	}
	lines := slices.Concat(r.wrapped...)
	if r.follow {
		r.scroll.offset = len(lines)
	}
	return r.scroll.window(lines, h)
}

// outputPopup keeps the targets visible around the pane. Its height follows the content up to the
// body height; a running pane keeps a small floor so it does not jitter, a finished one ends with the close hint.
func outputPopup(th theme, r *run, w, bodyH int) popup {
	popupW := min(w-outputMargin, maxOutputWidth)
	inner := max(bodyH-outputMargin-borderCells, 0)
	if !r.done {
		lines := r.view(th, contentWidth(popupW), inner)
		popupH := min(max(len(lines), runningFloorRows), inner) + borderCells
		return popup{box: renderBox(th, r.boxTitle(), lines, popupW, popupH, true), w: popupW, h: popupH}
	}
	lines := r.view(th, contentWidth(popupW), max(inner-1, 0))
	lines = append(lines, th.muted.Render(closeHintText))
	popupH := min(len(lines), inner) + borderCells
	return popup{box: renderBox(th, r.boxTitle(), lines, popupW, popupH, true), w: popupW, h: popupH}
}
