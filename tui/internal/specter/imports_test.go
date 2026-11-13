package specter

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// The Specter root is global, so this package must stay independent of per-CLI targets, plan execution and the UI.
func TestPackage_NeverImportsTargetActionOrUI(t *testing.T) {
	forbidden := []string{"/internal/target", "/internal/action", "/internal/ui"}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		checked++
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			for _, bad := range forbidden {
				if strings.HasSuffix(path, bad) || strings.Contains(path, bad+"/") {
					t.Errorf("%s imports %s", name, path)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no production files found to check")
	}
}
