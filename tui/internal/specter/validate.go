package specter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const changeItemType = "change"

type Issue struct {
	Level   string `json:"level"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type ValidationItem struct {
	ID     string  `json:"id"`
	Type   string  `json:"type"`
	Valid  bool    `json:"valid"`
	Issues []Issue `json:"issues"`
}

type Validation struct {
	Items []ValidationItem `json:"items"`
}

// UnmarshalJSON rejects output without `items`, so JSON that is not a validation report counts as a tool failure.
func (v *Validation) UnmarshalJSON(data []byte) error {
	var raw struct {
		Items *[]ValidationItem `json:"items"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw.Items == nil {
		return errors.New("missing items")
	}
	v.Items = *raw.Items
	return nil
}

// ValidationRecord is the session-only outcome for one change.
type ValidationRecord struct {
	At     time.Time
	Valid  bool
	Issues []Issue
}

// ValidateStore treats an invalid change as a result: only a run without JSON output is an error.
func ValidateStore(ctx context.Context, exec check.Exec, id string) (Validation, error) {
	ctx, cancel := context.WithTimeout(ctx, OpenspecValidateTimeout)
	defer cancel()
	var v Validation
	err := openspecJSON(ctx, exec, &v, true, "validate", "--all", "--strict", "--json", "--no-interactive", "--store", id)
	return v, err
}

// ChangeRecords keys the result by change name; spec items are not tracked.
func (v Validation) ChangeRecords(at time.Time) map[string]ValidationRecord {
	records := map[string]ValidationRecord{}
	for _, item := range v.Items {
		if item.Type == changeItemType {
			records[item.ID] = ValidationRecord{At: at, Valid: item.Valid, Issues: item.Issues}
		}
	}
	return records
}

// IsValidated is false on any doubt: never validated, invalid, edited after the run started, or an unparseable lastModified.
func IsValidated(rec ValidationRecord, found bool, lastModified string) bool {
	modified, ok := ParseLastModified(lastModified)
	return found && rec.Valid && ok && !modified.After(rec.At)
}
