package check_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

func TestValidJSON(t *testing.T) {
	env := newHome(t, nil)
	writeFile(t, filepath.Join(env.Home, "good.json"), `{"a": 1}`, 0o644)
	writeFile(t, filepath.Join(env.Home, "bad.json"), "{\n  \"a\": 1,\n}", 0o644)
	writeFile(t, filepath.Join(env.Home, "empty.json"), "", 0o644)
	if err := os.MkdirAll(filepath.Join(env.Home, "adir.json"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := runCheck(t, check.ValidJSON("json", []check.JSONFile{
		{Path: "good.json", Missing: check.StatusError},
		{Path: "bad.json", Missing: check.StatusError},
		{Path: "absent-error.json", Missing: check.StatusError},
		{Path: "absent-warning.json", Missing: check.StatusWarning},
		{Path: "adir.json", Missing: check.StatusError},
		{Path: "empty.json", Missing: check.StatusError},
	}), env)

	if len(got) != 6 {
		t.Fatalf("got %d results, want 6: %+v", len(got), got)
	}
	tests := []struct {
		subject string
		want    check.Status
	}{
		{"good.json", check.StatusOK},
		{"bad.json", check.StatusError},
		{"absent-error.json", check.StatusError},
		{"absent-warning.json", check.StatusWarning},
		{"adir.json", check.StatusUnknown},
		{"empty.json", check.StatusError},
	}
	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			r := resultFor(t, got, tt.subject)
			if r.Status != tt.want {
				t.Errorf("status = %q, want %q (%+v)", r.Status, tt.want, r)
			}
			if r.CheckID != "json" {
				t.Errorf("CheckID = %q, want json", r.CheckID)
			}
		})
	}

	// The trailing comma sits on line 3: the position must be line/column, not a byte offset.
	if msg := resultFor(t, got, "bad.json").Message; !strings.Contains(msg, "line 3, column ") {
		t.Errorf("message %q lacks \"line 3, column N\"", msg)
	}
}
