package check

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// JSONFile is a home-relative file that must parse as JSON; Missing is the status when it is absent.
type JSONFile struct {
	Path    string
	Missing Status
}

func ValidJSON(id string, files []JSONFile) Check {
	return Check{ID: id, Run: func(_ context.Context, env Env) []Result {
		results := make([]Result, 0, len(files))
		for _, f := range files {
			r := Result{CheckID: id, Subject: f.Path}
			data, err := env.FS.ReadFile(filepath.Join(env.Home, f.Path))
			switch {
			case errors.Is(err, fs.ErrNotExist):
				r.Status, r.Message = f.Missing, "file not found"
			case err != nil:
				r.Status, r.Message = StatusUnknown, err.Error()
			case len(data) == 0:
				r.Status, r.Message = StatusError, "invalid JSON: empty file"
			case json.Valid(data):
				r.Status = StatusOK
			default:
				r.Status, r.Message = StatusError, "invalid JSON: "+jsonErrorAt(data)
			}
			results = append(results, r)
		}
		return results
	}}
}

// jsonErrorAt describes the first syntax error of data as "<reason> at line N, column M".
func jsonErrorAt(data []byte) string {
	var v any
	err := json.Unmarshal(data, &v)
	var syntax *json.SyntaxError
	if !errors.As(err, &syntax) {
		return err.Error()
	}
	// Offset counts the bytes read including the offending one.
	end := min(max(int(syntax.Offset), 1), len(data))
	before := data[:end-1]
	line := 1 + bytes.Count(before, []byte("\n"))
	column := end - bytes.LastIndexByte(before, '\n') - 1
	return fmt.Sprintf("%s at line %d, column %d", syntax.Error(), line, column)
}
