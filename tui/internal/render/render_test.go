package render_test

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/osfs"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/render"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/state"
)

const (
	defaultRoot = "~/dev/specter"
	customRoot  = "~/work/specs"
	placeholder = "{{SPECTER_ROOT}}"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name, in, root, want string
	}{
		{"default root", "a " + placeholder + " b", defaultRoot, "a ~/dev/specter b"},
		{"custom root", "a " + placeholder + "/x", customRoot, "a ~/work/specs/x"},
		{"every occurrence", placeholder + " " + placeholder, defaultRoot, "~/dev/specter ~/dev/specter"},
		{"no placeholder", "plain ~/dev/other", defaultRoot, "plain ~/dev/other"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(render.Render([]byte(tt.in), tt.root)); got != tt.want {
				t.Errorf("Render = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnrender_StopsAtNameBytes(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"end of data", "at ~/dev/specter", "at " + placeholder},
		{"slash", "~/dev/specter/x", placeholder + "/x"},
		{"dot and paren", "(~/dev/specter).", "(" + placeholder + ")."},
		{"hyphen continues the name", "~/dev/specter-old", "~/dev/specter-old"},
		{"underscore continues the name", "~/dev/specter_v2", "~/dev/specter_v2"},
		{"digit continues the name", "~/dev/specter2", "~/dev/specter2"},
		{"letter continues the name", "~/dev/specterX", "~/dev/specterX"},
		{"clean and dirty hits in one text", "~/dev/specter-old ~/dev/specter", "~/dev/specter-old " + placeholder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(render.Unrender([]byte(tt.in), defaultRoot)); got != tt.want {
				t.Errorf("Unrender = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFixturePairs(t *testing.T) {
	read := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	tests := []struct {
		name, from, to, root string
		apply                func([]byte, string) []byte
	}{
		{"render default root", "source.md", "rendered-default.md", defaultRoot, render.Render},
		{"render custom root", "source.md", "rendered-custom.md", customRoot, render.Render},
		{"unrender default root", "rendered-default.md", "source.md", defaultRoot, render.Unrender},
		{"unrender custom root", "rendered-custom.md", "source.md", customRoot, render.Unrender},
		{"unrender boundary cases", "boundary-rendered.md", "boundary-source.md", defaultRoot, render.Unrender},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := string(tt.apply(read(tt.from), tt.root)), string(read(tt.to)); got != want {
				t.Errorf("%s -> %s\ngot:\n%s\nwant:\n%s", tt.from, tt.to, got, want)
			}
		})
	}
}

func TestRenderThenUnrender_RestoresTheSource(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "source.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{defaultRoot, customRoot} {
		t.Run(root, func(t *testing.T) {
			if got := render.Unrender(render.Render(source, root), root); string(got) != string(source) {
				t.Errorf("round trip changed the source:\n%s", got)
			}
		})
	}
}

func TestRenderFor_TransformsOnlyListedFiles(t *testing.T) {
	listed := []string{".claude/agents/tech-orchestrator.md", ".claude/CLAUDE.md"}
	in := []byte("root " + placeholder)
	tests := []struct {
		name, target, rel string
		want              string
	}{
		{"listed file in a directory component", ".claude/agents", "tech-orchestrator.md", "root " + defaultRoot},
		{"listed single-file component", ".claude/CLAUDE.md", ".", "root " + defaultRoot},
		{"unlisted file next to a listed one", ".claude/agents", "README.md", "root " + placeholder},
		{"unlisted single-file component", ".claude/commands", ".", "root " + placeholder},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(render.RenderFor(listed, tt.target, defaultRoot)(tt.rel, in))
			if got != tt.want {
				t.Errorf("RenderFor = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUnrenderFor_TransformsOnlyListedFiles(t *testing.T) {
	listed := []string{".claude/agents/tech-orchestrator.md"}
	in := []byte("root " + defaultRoot)
	if got := string(render.UnrenderFor(listed, ".claude/agents", defaultRoot)("tech-orchestrator.md", in)); got != "root "+placeholder {
		t.Errorf("listed file = %q, want placeholder restored", got)
	}
	if got := string(render.UnrenderFor(listed, ".claude/agents", defaultRoot)("README.md", in)); got != string(in) {
		t.Errorf("unlisted file = %q, want untouched", got)
	}
}

func writeList(t *testing.T, lines string) string {
	t.Helper()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "targets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, render.ListFile), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestForComponent_SavedRootRule(t *testing.T) {
	const home = "/home/u"
	tests := []struct {
		name, saved, want string
	}{
		{"unset falls back", "", "~/dev/specter"},
		{"valid custom root", home + "/work/specs", "~/work/specs"},
		{"outside home falls back", "/srv/specs", "~/dev/specter"},
		{"home itself falls back", home, "~/dev/specter"},
		{"dotdot out of home falls back", home + "/work/../../etc", "~/dev/specter"},
		{"dotdot staying under home falls back", home + "/a/../work", "~/dev/specter"},
		{"relative falls back", "work/specs", "~/dev/specter"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := check.Env{FS: osfs.FS{}, Home: home, Repo: writeList(t, ".claude/CLAUDE.md\n"), Saved: state.State{SpecterRoot: tt.saved}}
			renderFn, _ := render.ForComponent(env, ".claude/CLAUDE.md")
			if got := string(renderFn(".", []byte(placeholder))); got != tt.want {
				t.Errorf("rendered root = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForComponent_NilWithoutListedFiles(t *testing.T) {
	env := check.Env{FS: osfs.FS{}, Home: "/home/u", Repo: writeList(t, ".claude/agents/a.md\n")}
	if r, u := render.ForComponent(env, ".claude/commands"); r != nil || u != nil {
		t.Error("component without listed files must get nil transforms")
	}
	if r, u := render.ForComponent(env, ".claude/agents"); r == nil || u == nil {
		t.Error("component with a listed file must get transforms")
	}
}

// root-cases.txt is shared with scripts/tests/render-check.sh and the hook selftest.
func TestForComponent_SharedRootCases(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "root-cases.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	home := t.TempDir()
	repo := writeList(t, ".claude/CLAUDE.md\n")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if scanner.Text() == "" || strings.HasPrefix(scanner.Text(), "#") {
			continue
		}
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) != 3 {
			t.Fatalf("bad case line %q: want name, state line, expected root", scanner.Text())
		}
		name, stateLine, want := fields[0], strings.ReplaceAll(fields[1], "$HOME", home), fields[2]
		t.Run(name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "state.toml")
			if err := os.WriteFile(statePath, []byte(stateLine+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			saved, warning := state.NewStore(statePath).Load()
			if warning != "" {
				t.Fatalf("state did not parse: %s", warning)
			}
			env := check.Env{FS: osfs.FS{}, Home: home, Repo: repo, Saved: saved}

			renderFn, _ := render.ForComponent(env, ".claude/CLAUDE.md")

			if got := string(renderFn(".", []byte(placeholder))); got != want {
				t.Errorf("rendered root = %q, want %q", got, want)
			}
		})
	}
}
