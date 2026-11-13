package specter_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const validateCommand = "openspec validate --all --strict --json --no-interactive --store s"

const invalidChangeJSON = `{"items":[
 {"id":"good","type":"change","valid":true,"issues":[],"durationMs":3},
 {"id":"bad","type":"change","valid":false,"issues":[{"level":"ERROR","path":"proposal","message":"missing Why"}],"durationMs":4},
 {"id":"some-spec","type":"spec","valid":true,"issues":[],"durationMs":1}
],"summary":{"totals":{"items":3,"passed":2,"failed":1}}}`

func validateWith(res check.ExecResult) (specter.Validation, error, *fakeExec) {
	exec := &fakeExec{results: map[string]check.ExecResult{validateCommand: res}}
	v, err := specter.ValidateStore(context.Background(), exec, "s")
	return v, err, exec
}

func itemByID(t *testing.T, v specter.Validation, id string) specter.ValidationItem {
	t.Helper()
	for _, it := range v.Items {
		if it.ID == id {
			return it
		}
	}
	t.Fatalf("no item %q in %+v", id, v.Items)
	return specter.ValidationItem{}
}

func TestValidateStore_InvalidChangeWithNonZeroExit_IsAResultNotAToolFailure(t *testing.T) {
	v, err, exec := validateWith(check.ExecResult{ExitCode: 1, Stdout: []byte(invalidChangeJSON), Stderr: []byte("noise")})

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	if len(exec.callsTo("openspec validate")) != 1 {
		t.Errorf("calls = %v, want exactly the validate command", exec.calls)
	}
	bad := itemByID(t, v, "bad")
	if bad.Valid || len(bad.Issues) != 1 || bad.Issues[0].Level != "ERROR" || bad.Issues[0].Path != "proposal" || bad.Issues[0].Message != "missing Why" {
		t.Errorf("bad = %+v", bad)
	}
	if !itemByID(t, v, "good").Valid {
		t.Error("good change reported invalid")
	}
}

func TestValidateStore_NonZeroExitWithoutJSON_IsToolFailureWithStderr(t *testing.T) {
	_, err, _ := validateWith(check.ExecResult{ExitCode: 2, Stderr: []byte("boom: cannot open store")})

	if err == nil || !strings.Contains(err.Error(), "boom: cannot open store") {
		t.Errorf("err = %v, want a tool failure carrying stderr", err)
	}
}

func TestValidateStore_JSONWithoutItems_IsToolFailure(t *testing.T) {
	for _, res := range []check.ExecResult{
		{ExitCode: 1, Stdout: []byte(`{"error":"no store"}`), Stderr: []byte("boom: no such store")},
		{ExitCode: 0, Stdout: []byte(`{"summary":{}}`)},
		{ExitCode: 1, Stdout: []byte(`{"items":null}`), Stderr: []byte("boom: no such store")},
	} {
		_, err, _ := validateWith(res)

		if err == nil {
			t.Errorf("result %+v: err = nil, want a tool failure", res)
		}
		if res.ExitCode != 0 && (err == nil || !strings.Contains(err.Error(), "boom: no such store")) {
			t.Errorf("err = %v, want stderr carried", err)
		}
	}
}

func TestValidateStore_RunFailure_IsToolFailure(t *testing.T) {
	exec := &fakeExec{results: map[string]check.ExecResult{}}

	if _, err := specter.ValidateStore(context.Background(), exec, "s"); err == nil {
		t.Error("err = nil, want a tool failure when the command cannot run")
	}
}

// ChangeRecords is an assumed accessor: it turns a result into the session map keyed by change name.
func TestValidation_ChangeRecords_OnlyChangeItemsUpdateTheMap(t *testing.T) {
	v, err, _ := validateWith(check.ExecResult{ExitCode: 1, Stdout: []byte(invalidChangeJSON)})
	if err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

	records := v.ChangeRecords(at)

	if len(records) != 2 {
		t.Fatalf("records = %+v, want only the two change items", records)
	}
	if _, ok := records["some-spec"]; ok {
		t.Error("a spec item became a change record")
	}
	if rec := records["bad"]; rec.Valid || rec.At != at || len(rec.Issues) != 1 {
		t.Errorf("bad record = %+v", rec)
	}
	if rec := records["good"]; !rec.Valid || rec.At != at {
		t.Errorf("good record = %+v", rec)
	}
}

func TestIsValidated(t *testing.T) {
	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	valid := specter.ValidationRecord{At: at, Valid: true}
	cases := []struct {
		name         string
		rec          specter.ValidationRecord
		found        bool
		lastModified string
		want         bool
	}{
		{"validated and not modified since", valid, true, at.Add(-time.Minute).Format(time.RFC3339), true},
		{"modified at the validation instant", valid, true, at.Format(time.RFC3339), true},
		{"edited after validation", valid, true, at.Add(time.Second).Format(time.RFC3339), false},
		{"invalid result", specter.ValidationRecord{At: at, Valid: false}, true, at.Add(-time.Minute).Format(time.RFC3339), false},
		{"never validated this session", specter.ValidationRecord{}, false, at.Add(-time.Minute).Format(time.RFC3339), false},
		{"unparseable lastModified fails closed", valid, true, "yesterday", false},
		{"empty lastModified fails closed", valid, true, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := specter.IsValidated(tc.rec, tc.found, tc.lastModified); got != tc.want {
				t.Errorf("IsValidated = %v, want %v", got, tc.want)
			}
		})
	}
}
