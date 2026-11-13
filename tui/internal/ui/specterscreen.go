package ui

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const (
	specterName    = "Specter"
	pendingText    = "pending"
	unknownText    = "unknown"
	separatorGlyph = "─"
	notYetGlyph    = "·"
)

// storeValidation is the session-only result of validating one store; nothing is persisted.
type storeValidation struct {
	running bool
	records map[string]specter.ValidationRecord // by change name
	err     error
}

// specterLine is the one-line Specter health shown above the CLI dashboard and on the Specter screen.
func specterLine(th theme, st *specterState, width int) string {
	if !st.loaded {
		return th.muted.Render(truncate(specterName+" "+pendingText, width))
	}
	sum := specter.Summary(append(specter.Checks(st.facts), specter.Drift(st.facts)...))
	return th.forStatus(sum.Status).Render(truncate(specterName+" "+statusGlyph(sum.Status)+" "+sum.Text, width))
}

func separatorLine(th theme, width int) string {
	return th.muted.Render(strings.Repeat(separatorGlyph, width))
}

// specterScreen is the panel content while the Specter entry is selected.
func specterScreen(th theme, st *specterState, width int) []string {
	lines := []string{specterLine(th, st, width)}
	if !st.loaded {
		return lines
	}
	f := st.facts
	lines = append(lines, rootLines(th, f, width)...)
	lines = append(lines, overviewLines(th, f, width)...)
	lines = append(lines, "", th.section.Render("Checks"))
	lines = append(lines, resultLines(th, specter.Checks(f), width)...)
	if drift := specter.Drift(f); len(drift) > 0 {
		lines = append(lines, "", th.section.Render("Drift"))
		lines = append(lines, resultLines(th, drift, width)...)
	}
	return append(lines, validationLines(th, st, width)...)
}

func rootLines(th theme, f specter.Facts, width int) []string {
	lines := []string{"", th.section.Render("Root")}
	lines = append(lines, styleAll(th.plain, wrap(fmt.Sprintf("%s (%s)", f.Root.Path, f.Root.Source), width))...)
	switch {
	case !f.RootExists && f.FoldersErr != nil:
		lines = append(lines, styleAll(th.warning, wrap("Root unreadable: "+f.FoldersErr.Error(), width))...)
	case !f.RootExists:
		lines = append(lines, styleAll(th.warning, wrap("Root not found. Clone it with:", width))...)
		lines = append(lines, styleAll(th.plain, wrap(f.CloneCommand, width))...)
	}
	if f.Root.OutsideHome {
		lines = append(lines, styleAll(th.warning, wrap("Root is outside HOME: fast init and maintenance are not supported.", width))...)
	}
	for _, ref := range f.RootRefs {
		text := ref.Label + ": not found"
		if ref.Found {
			text = fmt.Sprintf("%s: %s copy, %s", ref.Label, ref.Copy, referenceText(ref.References))
		}
		lines = append(lines, styleAll(th.muted, wrap(text, width))...)
	}
	return lines
}

func referenceText(references bool) string {
	if references {
		return "references the root"
	}
	return "does not reference the root"
}

func overviewLines(th theme, f specter.Facts, width int) []string {
	lines := []string{"", th.section.Render("Stores")}
	switch {
	case !f.Openspec.Found || f.RegisteredErr != nil:
		return append(lines, th.unknown.Render(unknownText))
	case len(f.Registered) == 0:
		return append(lines, th.muted.Render("No store registered."))
	}
	for _, s := range f.Registered {
		lines = append(lines, styleAll(th.plain, wrap(s.ID+"  "+s.Root, width))...)
		stats, queried := f.Stores[s.ID]
		for _, row := range []string{
			"repos: " + reposText(f.Repos, s.ID),
			"engram: " + engramText(f.Repos, s.ID),
			"counts: " + countsText(stats, queried),
		} {
			lines = append(lines, styleAll(th.muted, wrap("  "+row, width))...)
		}
	}
	return lines
}

func reposText(view specter.ReposView, store string) string {
	if view.Err != nil {
		return unknownText
	}
	var repos []string
	for _, e := range view.Repos {
		if e.Store == store {
			repos = append(repos, cmp.Or(e.Remote, e.Path))
		}
	}
	return listOrNone(repos)
}

func engramText(view specter.ReposView, store string) string {
	if view.Err != nil {
		return unknownText
	}
	return listOrNone(view.Stores[store].Engram)
}

func countsText(stats specter.StoreStats, queried bool) string {
	if !queried || stats.Err != nil {
		return unknownText
	}
	return fmt.Sprintf("active %d, archived %d, specs %d", stats.Active, stats.Archived, stats.Specs)
}

func listOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

// resultLines renders each result as glyph and message on one line, then its suggestion.
func resultLines(th theme, results []check.Result, width int) []string {
	var lines []string
	for _, r := range results {
		lines = append(lines, styleAll(th.forStatus(r.Status), wrap(statusGlyph(r.Status)+" "+r.Message, width))...)
		if r.Suggestion != "" {
			lines = append(lines, styleAll(th.muted, wrap("  Suggestion: "+r.Suggestion, width))...)
		}
	}
	return lines
}

func validationLines(th theme, st *specterState, width int) []string {
	var lines []string
	for _, s := range st.facts.Registered {
		stats, sv := st.facts.Stores[s.ID], st.validation[s.ID]
		if len(stats.Changes) == 0 && sv == nil {
			continue
		}
		if lines == nil {
			lines = []string{"", th.section.Render("Validation")}
		}
		lines = append(lines, styleAll(th.plain, wrap(s.ID, width))...)
		if sv == nil {
			sv = &storeValidation{}
		}
		switch {
		case sv.running:
			lines = append(lines, styleAll(th.muted, wrap("  validating...", width))...)
		case sv.err != nil:
			lines = append(lines, styleAll(th.err, wrap("  validation failed: "+sv.err.Error(), width))...)
		}
		for _, c := range stats.Changes {
			lines = append(lines, storeChangeLines(th, c, sv, width)...)
		}
	}
	return lines
}

// storeChangeLines marks a change validated only while its record still holds, then lists the issues of an invalid one.
func storeChangeLines(th theme, c specter.ChangeInfo, sv *storeValidation, width int) []string {
	rec, found := sv.records[c.Name]
	mark, style := notYetGlyph, th.muted
	switch {
	case specter.IsValidated(rec, found, c.LastModified):
		mark, style = statusGlyph(check.StatusOK), th.ok
	case found && !rec.Valid:
		mark, style = statusGlyph(check.StatusError), th.err
	}
	lines := styleAll(style, wrap("  "+mark+" "+c.Name, width))
	if found && !rec.Valid {
		for _, issue := range rec.Issues {
			lines = append(lines, styleAll(th.err, wrap(fmt.Sprintf("      %s: %s", issue.Level, issue.Message), width))...)
		}
	}
	return lines
}
