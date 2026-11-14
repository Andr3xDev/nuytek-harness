package ui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/action"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	backupsTitle     = "Backups"
	noBackupsText    = "No backups"
	emptyFiles       = "The snapshot holds no files."
	nothingToRestore = "Nothing to restore."
	nothingToBackUp  = "Nothing to back up."
	bytesPerUnit     = 1024
)

type backupRow struct {
	entry backup.Entry
	size  int64
}

// backupsLoaded and filesLoaded carry background results; epoch drops one that a reload superseded.
type backupsLoaded struct {
	b     *backupsTab
	epoch int
	rows  []backupRow
}

type filesLoaded struct {
	b     *backupsTab
	entry backup.Entry
	files []string
}

type fileView struct {
	entry  backup.Entry
	files  []string
	loaded bool
	scroll scroller
}

type backupsTab struct {
	svc    *services
	t      target.Target
	rows   []backupRow
	loaded bool
	cursor int
	epoch  int
	files  *fileView       // set while a snapshot is open
	flow   *flow           // set while a restore or delete flow runs
	marks  map[string]bool // backup paths marked for deletion
	notice string
}

func newBackupsTab(svc *services, t target.Target) *backupsTab {
	return &backupsTab{svc: svc, t: t, marks: map[string]bool{}}
}

func (b *backupsTab) Title() string { return backupsTitle }

func (b *backupsTab) Activate(t target.Target) tea.Cmd {
	b.t = t
	return b.load()
}

// load lists both stores and sums the sizes off the UI loop.
func (b *backupsTab) load() tea.Cmd {
	b.epoch++
	epoch, env, t := b.epoch, b.svc.Env, b.t
	return func() tea.Msg {
		var rows []backupRow
		for _, e := range backup.List(env.FS, t.Spec.Stores(env)) {
			rows = append(rows, backupRow{entry: e, size: backup.Size(env.FS, e)})
		}
		return backupsLoaded{b: b, epoch: epoch, rows: rows}
	}
}

func (b *backupsTab) Update(msg tea.Msg) (tab, tea.Cmd) {
	switch msg := msg.(type) {
	case backupsLoaded:
		if msg.b == b && msg.epoch == b.epoch {
			b.rows, b.loaded = msg.rows, true
			b.cursor = max(min(b.cursor, len(b.rows)-1), 0)
			b.dropVanishedMarks()
		}
	case filesLoaded:
		if msg.b == b && b.files != nil && b.files.entry.Path == msg.entry.Path {
			b.files.files, b.files.loaded = msg.files, true
		}
	case runDismissed:
		b.flow, b.files = nil, nil
		clear(b.marks)
		if b.loaded {
			return b, b.load()
		}
	case configChanged:
		if b.loaded {
			return b, b.load()
		}
	case plannedMsg:
		if b.flow != nil {
			return b, b.flow.Update(msg)
		}
	case tea.KeyPressMsg:
		b.notice = ""
		return b, b.handleKey(msg)
	}
	return b, nil
}

func (b *backupsTab) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := b.svc.keys
	if b.flow != nil {
		cmd := b.flow.Update(msg)
		if b.flow.finished() {
			b.flow = nil
		}
		return cmd
	}
	if b.files != nil {
		switch {
		case k.step(msg) != 0:
			b.files.scroll.by(k.step(msg))
		case key.Matches(msg, k.Cancel):
			b.files = nil
		}
		return nil
	}
	switch {
	case k.step(msg) != 0:
		b.cursor = max(min(b.cursor+k.step(msg), len(b.rows)-1), 0)
	case key.Matches(msg, k.Backup):
		return b.startFlow(newDirectFlow(b.svc, "Backup", nothingToBackUp, b.backupPlan()))
	case len(b.rows) == 0:
	case key.Matches(msg, k.Accept):
		return b.openFiles(b.rows[b.cursor].entry)
	case key.Matches(msg, k.Restore):
		e := b.rows[b.cursor].entry
		return b.startFlow(newDirectFlow(b.svc, "Restore "+e.Stamp, nothingToRestore, b.restorePlan(e)))
	case key.Matches(msg, k.Toggle):
		path := b.rows[b.cursor].entry.Path
		b.marks[path] = !b.marks[path]
	case key.Matches(msg, k.All):
		all := len(b.markedEntries()) == len(b.rows)
		for _, r := range b.rows {
			b.marks[r.entry.Path] = !all
		}
	case key.Matches(msg, k.Delete):
		entries := b.markedEntries()
		if len(entries) == 0 {
			entries = []backup.Entry{b.rows[b.cursor].entry}
		}
		title := "Delete backup " + entries[0].Stamp
		if len(entries) > 1 {
			title = fmt.Sprintf("Delete %d backups", len(entries))
		}
		f := newDirectFlow(b.svc, title, nothingToDoText, b.deletePlan(entries))
		f.confirmFirst = true
		return b.startFlow(f)
	}
	return nil
}

// markedEntries returns the marked backups in list order.
func (b *backupsTab) markedEntries() []backup.Entry {
	var entries []backup.Entry
	for _, r := range b.rows {
		if b.marks[r.entry.Path] {
			entries = append(entries, r.entry)
		}
	}
	return entries
}

func (b *backupsTab) dropVanishedMarks() {
	present := map[string]bool{}
	for _, r := range b.rows {
		present[r.entry.Path] = true
	}
	for path := range b.marks {
		if !present[path] {
			delete(b.marks, path)
		}
	}
}

func (b *backupsTab) startFlow(f *flow) tea.Cmd {
	b.flow = f
	return f.start()
}

func (b *backupsTab) restorePlan(e backup.Entry) planFunc {
	svc, t := b.svc, b.t
	return func(ctx context.Context, _ []string) (plan.Plan, error) {
		return action.PlanRestore(ctx, svc.Env, t, e)
	}
}

func (b *backupsTab) backupPlan() planFunc {
	svc, t := b.svc, b.t
	return func(context.Context, []string) (plan.Plan, error) {
		return action.PlanBackup(svc.Env, t)
	}
}

func (b *backupsTab) deletePlan(entries []backup.Entry) planFunc {
	svc, t := b.svc, b.t
	return func(context.Context, []string) (plan.Plan, error) {
		return action.PlanDeleteBackup(svc.Env, t, entries...)
	}
}

func (b *backupsTab) openFiles(e backup.Entry) tea.Cmd {
	b.files = &fileView{entry: e}
	env := b.svc.Env
	return func() tea.Msg { return filesLoaded{b: b, entry: e, files: backup.Files(env.FS, e)} }
}

func (b *backupsTab) Keys() []key.Binding {
	k := b.svc.keys
	switch {
	case b.flow != nil:
		return b.flow.Keys()
	case b.files != nil:
		return []key.Binding{k.Cancel, k.Move}
	case len(b.rows) == 0:
		return []key.Binding{k.Backup}
	}
	return []key.Binding{k.Toggle, k.All, described(k.Accept, "view files"), k.Backup, k.Restore, k.Delete, k.Move}
}

func (b *backupsTab) View(w, h int) string {
	th := b.svc.th
	switch {
	case b.flow != nil:
		return b.flow.View(w, h)
	case b.files != nil:
		return strings.Join(b.filesLines(w, h), "\n")
	case !b.loaded:
		return th.muted.Render(computingText)
	case len(b.rows) == 0:
		return th.muted.Render(noBackupsText)
	}
	rows := make([]listRow, len(b.rows))
	for i, r := range b.rows {
		mark := "[ ]"
		if b.marks[r.entry.Path] {
			mark = "[x]"
		}
		rows[i] = listRow{text: mark + " " + backupText(r)}
	}
	lines, at := listLines(th, rows, b.cursor, w)
	return strings.Join(windowAround(lines, at, h), "\n")
}

func (b *backupsTab) filesLines(w, h int) []string {
	th, v := b.svc.th, b.files
	header := styleAll(th.section, wrap(v.entry.Label+" "+v.entry.Stamp, w))
	var body []string
	switch {
	case !v.loaded:
		body = []string{th.muted.Render(computingText)}
	case len(v.files) == 0:
		body = []string{th.muted.Render(emptyFiles)}
	default:
		for _, f := range v.files {
			body = append(body, wrap(sanitizeLine(f), w)...)
		}
	}
	return append(header, v.scroll.window(body, h-len(header))...)
}

func backupText(r backupRow) string {
	text := fmt.Sprintf("%s  %s  %s", r.entry.Label, r.entry.Stamp, humanSize(r.size))
	if r.entry.IsLink {
		text += " (symlink)"
	}
	return sanitizeLine(text)
}

func humanSize(n int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	size, unit := float64(n), 0
	for size >= bytesPerUnit && unit < len(units)-1 {
		size /= bytesPerUnit
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d B", n)
	}
	return fmt.Sprintf("%.1f %s", size, units[unit])
}

var _ tab = (*backupsTab)(nil)
