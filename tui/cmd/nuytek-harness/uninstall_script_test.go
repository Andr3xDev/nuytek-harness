package main_test

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	uninstallScript = "../../../scripts/uninstall-tui.sh"
	nothingMessage  = "nothing installed"
)

func binPath(home string) string   { return filepath.Join(home, ".local", "bin", "nuytek-harness") }
func statePath(home string) string { return filepath.Join(home, ".local", "state", "nuytek-harness") }

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func installFiles(t *testing.T, home string) {
	t.Helper()
	writeFile(t, binPath(home), "binary")
	writeFile(t, filepath.Join(statePath(home), "state.toml"), "{}")
}

// runUninstall runs the script with only HOME and PATH set, so it can never reach the real HOME.
func runUninstall(t *testing.T, home string) (string, int) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	script, err := filepath.Abs(uninstallScript)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bash, script)
	cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH")}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return out.String(), 0
	case errors.As(err, &exitErr):
		return out.String(), exitErr.ExitCode()
	}
	t.Fatalf("running script: %v", err)
	return "", -1
}

func requireGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s still exists (err=%v)", path, err)
	}
}

func TestUninstallScript_RemovesBinaryAndState(t *testing.T) {
	home := t.TempDir()
	installFiles(t, home)

	out, code := runUninstall(t, home)

	if code != 0 {
		t.Fatalf("exit = %d, output:\n%s", code, out)
	}
	for _, p := range []string{binPath(home), statePath(home)} {
		requireGone(t, p)
		if !strings.Contains(out, "removed: "+p) {
			t.Errorf("output lacks %q:\n%s", "removed: "+p, out)
		}
	}
}

func TestUninstallScript_RejectsUnsafeHome(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	script, err := filepath.Abs(uninstallScript)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"unset", nil},
		{"empty", []string{"HOME="}},
		{"relative", []string{"HOME=relative/home"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(bash, script)
			cmd.Dir = t.TempDir()
			cmd.Env = append([]string{"PATH=" + os.Getenv("PATH")}, tc.env...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("exit = 0, want non-zero; output:\n%s", out)
			}
			if strings.Contains(string(out), "removed:") {
				t.Errorf("script removed something:\n%s", out)
			}
		})
	}
}

func TestUninstallScript_NothingInstalled(t *testing.T) {
	home := t.TempDir()
	out, code := runUninstall(t, home)
	if code != 0 || !strings.Contains(out, nothingMessage) {
		t.Fatalf("exit = %d, output:\n%s", code, out)
	}
}

func TestUninstallScript_SecondRunAfterRemovalReportsNothing(t *testing.T) {
	home := t.TempDir()
	installFiles(t, home)
	runUninstall(t, home)

	out, code := runUninstall(t, home)

	if code != 0 || !strings.Contains(out, nothingMessage) || strings.Contains(out, "removed:") {
		t.Fatalf("exit = %d, output:\n%s", code, out)
	}
}

func TestUninstallScript_LeavesHarnessConfigUntouched(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), `{"a":1}`)
	writeFile(t, filepath.Join(home, ".claude", "agents", "x.md"), "agent")
	writeFile(t, filepath.Join(home, ".claude.json"), `{"mcpServers":{}}`)
	writeFile(t, filepath.Join(home, ".claude-backups", "20260101", "settings.json"), "old")
	writeFile(t, filepath.Join(home, "repo", "targets", "claude.toml"), "x")
	if err := os.Symlink("settings.json", filepath.Join(home, ".claude", "link")); err != nil {
		t.Fatal(err)
	}
	installFiles(t, home)
	before := treeHash(t, home, binPath(home), statePath(home))

	if out, code := runUninstall(t, home); code != 0 {
		t.Fatalf("exit = %d, output:\n%s", code, out)
	}

	if after := treeHash(t, home, binPath(home), statePath(home)); after != before {
		t.Fatalf("tree changed outside bin and state:\nbefore %s\nafter  %s", before, after)
	}
}

func TestUninstallScript_SymlinkedStateRemovesOnlyLink(t *testing.T) {
	home := t.TempDir()
	external := filepath.Join(t.TempDir(), "real-state")
	writeFile(t, filepath.Join(external, "state.json"), "{}")
	if err := os.MkdirAll(filepath.Dir(statePath(home)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, statePath(home)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(binPath(home)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "missing-target"), binPath(home)); err != nil {
		t.Fatal(err)
	}

	out, code := runUninstall(t, home)

	if code != 0 {
		t.Fatalf("exit = %d, output:\n%s", code, out)
	}
	requireGone(t, statePath(home))
	requireGone(t, binPath(home))
	if got, err := os.ReadFile(filepath.Join(external, "state.json")); err != nil || string(got) != "{}" {
		t.Errorf("link target damaged: content=%q err=%v", got, err)
	}
	for _, p := range []string{binPath(home), statePath(home)} {
		if !strings.Contains(out, "removed: "+p) {
			t.Errorf("output lacks %q:\n%s", "removed: "+p, out)
		}
	}
}

// treeHash covers path, mode, content and link target of everything under root except the skipped paths.
func treeHash(t *testing.T, root string, skip ...string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		for _, s := range skip {
			if path == s {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		fmt.Fprintf(h, "%s|%v|", rel, info.Mode())
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			link, _ := os.Readlink(path)
			fmt.Fprint(h, link)
		case info.Mode().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			h.Write(data)
		}
		fmt.Fprintln(h)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
