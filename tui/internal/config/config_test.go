package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/config"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
)

const home = "/home/u"

func TestExpand(t *testing.T) {
	cases := []struct{ name, raw, want string }{
		{"tilde alone", "~", home},
		{"tilde slash", "~/x", home + "/x"},
		{"absolute cleaned", "/a/b/../c", "/a/c"},
		{"tilde not leading is literal", "/a/~/b", "/a/~/b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := config.Expand(c.raw, home); got != c.want {
				t.Errorf("Expand(%q) = %q, want %q", c.raw, got, c.want)
			}
		})
	}
}

func TestValidateSpecterRoot(t *testing.T) {
	realHome := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(realHome, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(outside, "afile")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, raw, want string
		wantErr         string // substring of the error; "" means accepted
	}{
		{"tilde path returns expanded", "~/specs", filepath.Join(realHome, "specs"), ""},
		{"outside HOME accepted", outside, outside, ""},
		{"missing path with existing parent accepted", "~/new", filepath.Join(realHome, "new"), ""},
		{"relative rejected", "specs", "", "absolute"},
		{"empty rejected", "", "", "empty"},
		{"missing path and missing parent rejected", filepath.Join(outside, "none", "specs"), "", "parent directory does not exist"},
		{"file rejected", file, "", "not a directory"}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := config.ValidateSpecterRoot(osfs.FS{}, realHome, c.raw)
			if c.wantErr == "" && err != nil {
				t.Fatalf("err = %v, want accepted", err)
			}
			if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
				t.Fatalf("err = %v, want one containing %q", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("path = %q, want %q", got, c.want)
			}
		})
	}
}

func TestValidateBackupsDir(t *testing.T) {
	own := home + "/.claude"
	cases := []struct {
		name, raw, want string
		wantErr         bool
	}{
		{"default sibling passes the prefix trap", "~/.claude-backups", home + "/.claude-backups", false},
		{"own tree rejected", "~/.claude", "", true},
		{"inside own tree rejected", "~/.claude/x", "", true},
		{"similar name accepted", "~/.claudeX", home + "/.claudeX", false},
		{"nonexistent accepted", "~/never/created", home + "/never/created", false},
		{"absolute outside HOME accepted", "/data/bk", "/data/bk", false},
		{"dotdot back into own tree rejected", "~/x/../.claude/y", "", true},
		{"relative rejected", "bk", "", true},
		{"empty rejected", "", "", true},
		{"blank rejected", "   ", "", true},
		{"surrounding spaces trimmed", "  ~/bk \n", home + "/bk", false},
		{"filesystem root rejected", "/", "", true},
		{"HOME rejected", "~", "", true},
		{"repo store rejected", "/repo/.refresh-backups", "", true},
		{"inside repo store rejected", "/repo/.refresh-backups/sub", "", true},
		{"ancestor of repo store rejected", "/repo", "", true},
		{"unrelated sibling of repo accepted", "/other", "/other", false},
		{"repo store sibling accepted", "/repo/.refresh-backupsX", "/repo/.refresh-backupsX", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := config.ValidateBackupsDir(home, c.raw, own, "/repo/.refresh-backups")
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("path = %q, want %q", got, c.want)
			}
		})
	}
}

func TestPathLabel(t *testing.T) {
	cases := []struct{ name, path, want string }{
		{"under HOME", home + "/bk", "~/bk"},
		{"outside HOME stays absolute", "/data/bk", "/data/bk"},
		{"HOME itself", home, "~"},
		{"sibling sharing HOME prefix stays absolute", home + "x/bk", home + "x/bk"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := config.PathLabel(home, c.path); got != c.want {
				t.Errorf("PathLabel(%q) = %q, want %q", c.path, got, c.want)
			}
		})
	}
}
