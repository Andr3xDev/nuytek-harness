package target_test

import (
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

// rowText joins every text field of a row; the exact field a line lives in is not part of the contract.
func rowText(r target.ManageRow) string { return r.Detail + "\n" + r.Preview }

func nonBlankLines(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func TestSpecManage_NilByDefault(t *testing.T) {
	if target.Planned("x", "X").Manage != nil {
		t.Fatal("a planned target must have no Manage func (no tab)")
	}
}

func TestDoctorSection_ErrorWithSuggestionShowsAllFields(t *testing.T) {
	sec := target.DoctorSection([]check.Result{{
		CheckID: "frontmatter", Subject: "skills/foo/SKILL.md", Message: "missing description",
		Status: check.StatusError, Suggestion: "add a description: field to the frontmatter",
	}})
	if sec.Title != "Doctor" || len(sec.Rows) != 1 {
		t.Fatalf("section = %+v, want title Doctor with one row", sec)
	}
	row := sec.Rows[0]
	if row.Status != check.StatusError || row.Name != "skills/foo/SKILL.md" {
		t.Fatalf("row = %+v, want status error and subject as name", row)
	}
	text := rowText(row)
	for _, want := range []string{"missing description", "add a description: field to the frontmatter"} {
		if !strings.Contains(text, want) {
			t.Errorf("row text %q lacks %q", text, want)
		}
	}
}

func TestDoctorSection_WarningWithoutSuggestionHasNoSuggestionLine(t *testing.T) {
	sec := target.DoctorSection([]check.Result{{Subject: "~/.claude.json", Message: "file missing", Status: check.StatusWarning}})
	text := rowText(sec.Rows[0])
	if !strings.Contains(text, "file missing") {
		t.Fatalf("row text %q lacks the message", text)
	}
	if got := nonBlankLines(text); len(got) != 1 {
		t.Fatalf("lines = %q, want only the message line", got)
	}
}

func TestDoctorSection_SuggestionAddsExactlyOneLine(t *testing.T) {
	without := target.DoctorSection([]check.Result{{Subject: "s", Message: "bad", Status: check.StatusError}})
	with := target.DoctorSection([]check.Result{{Subject: "s", Message: "bad", Status: check.StatusError, Suggestion: "do this"}})
	if got, base := len(nonBlankLines(rowText(with.Rows[0]))), len(nonBlankLines(rowText(without.Rows[0]))); got != base+1 {
		t.Fatalf("lines with suggestion = %d, without = %d, want one more", got, base)
	}
}

func TestDoctorSection_UnknownShowsReason(t *testing.T) {
	sec := target.DoctorSection([]check.Result{{Subject: "settings.json", Message: "invalid JSON: unexpected end", Status: check.StatusUnknown}})
	row := sec.Rows[0]
	if row.Status != check.StatusUnknown || !strings.Contains(rowText(row), "invalid JSON: unexpected end") {
		t.Fatalf("row = %+v, want unknown status and the reason", row)
	}
}

func TestDoctorSection_SkippedShowsMessage(t *testing.T) {
	sec := target.DoctorSection([]check.Result{{Subject: "s", Message: "not applicable here", Status: check.StatusSkipped}})
	row := sec.Rows[0]
	if row.Status != check.StatusSkipped || !strings.Contains(rowText(row), "not applicable here") {
		t.Fatalf("row = %+v, want skipped status and its message", row)
	}
}

func TestDoctorSection_OKShowsNoFailureDetail(t *testing.T) {
	sec := target.DoctorSection([]check.Result{{Subject: "settings.json", Status: check.StatusOK}})
	row := sec.Rows[0]
	if row.Status != check.StatusOK || strings.TrimSpace(rowText(row)) != "" {
		t.Fatalf("row = %+v, want ok status and no detail", row)
	}
}
