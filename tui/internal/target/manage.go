package target

import (
	"cmp"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const doctorTitle = "Doctor"

// ManageRow Status "" means no glyph; Preview is the long text shown for the selected row.
type ManageRow struct {
	Name    string
	Status  check.Status
	Detail  string
	Preview string
}

// ManageSection Empty is the message for a section without rows; Unknown, when set, marks the section unreadable and gives the reason.
type ManageSection struct {
	Title   string
	Rows    []ManageRow
	Empty   string
	Unknown string
}

// DoctorSection lists every check result; only non-ok rows carry the message and, when known, the suggestion.
func DoctorSection(results []check.Result) ManageSection {
	section := ManageSection{Title: doctorTitle}
	for _, r := range results {
		row := ManageRow{Name: cmp.Or(r.Subject, r.CheckID), Status: r.Status}
		if r.Status != check.StatusOK {
			row.Detail = r.Message
			if r.Suggestion != "" {
				row.Detail += "\n" + r.Suggestion
			}
		}
		section.Rows = append(section.Rows, row)
	}
	return section
}
