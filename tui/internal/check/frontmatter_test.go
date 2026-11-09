package check_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

func TestFrontmatter_Validation(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    check.Status
	}{
		{"valid", "---\nname: x\ndescription: does x\n---\nbody", check.StatusOK},
		{"no fence", "# just markdown", check.StatusError},
		{"empty description", "---\nname: x\ndescription: \"\"\n---\n", check.StatusError},
		{"missing name", "---\ndescription: does x\n---\n", check.StatusError},
		{"invalid yaml", "---\nname: [unclosed\ndescription: d\n---\n", check.StatusError},
		{"unterminated fence", "---\nname: x\ndescription: d\n", check.StatusError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newHome(t, nil)
			writeFile(t, filepath.Join(env.Home, "items", "a.md"), tt.content, 0o644)

			got := runCheck(t, check.Frontmatter("fm", []check.Set{{Root: "items", Pattern: "*.md"}}), env)

			if len(got) != 1 {
				t.Fatalf("got %d results, want 1: %+v", len(got), got)
			}
			if got[0].Status != tt.want || got[0].Subject != "a.md" || got[0].CheckID != "fm" {
				t.Errorf("got %+v, want status %q subject a.md check fm", got[0], tt.want)
			}
			if tt.want == check.StatusError && got[0].Message == "" {
				t.Errorf("error result needs a message: %+v", got[0])
			}
		})
	}
}

func TestFrontmatter_PatternsAndSkip(t *testing.T) {
	env := newHome(t, nil)
	const valid = "---\nname: n\ndescription: d\n---\n"
	root := filepath.Join(env.Home, "tree")
	writeFile(t, filepath.Join(root, "README.md"), "# no frontmatter", 0o644)
	writeFile(t, filepath.Join(root, "a.md"), valid, 0o644)
	writeFile(t, filepath.Join(root, "sub-agents", "x", "b.md"), valid, 0o644)
	writeFile(t, filepath.Join(root, "sub-agents", "x", "deep", "c.md"), valid, 0o644)
	writeFile(t, filepath.Join(root, "s", "SKILL.md"), valid, 0o644)

	got := runCheck(t, check.Frontmatter("fm", []check.Set{
		{Root: "tree", Pattern: "*.md", Skip: []string{"README.md"}},
		{Root: "tree", Pattern: "sub-agents/*/*.md"},
		{Root: "tree", Pattern: "*/SKILL.md"},
	}), env)

	var subjects []string
	for _, r := range got {
		subjects = append(subjects, r.Subject)
		if r.Status != check.StatusOK {
			t.Errorf("%s: status %q, want ok", r.Subject, r.Status)
		}
	}
	slices.Sort(subjects)
	want := []string{"a.md", "s/SKILL.md", "sub-agents/x/b.md"}
	if !slices.Equal(subjects, want) {
		t.Fatalf("subjects = %v, want %v", subjects, want)
	}
}

func TestFrontmatter_MissingRootYieldsNoResults(t *testing.T) {
	got := runCheck(t, check.Frontmatter("fm", []check.Set{{Root: "absent", Pattern: "*.md"}}), newHome(t, nil))

	if len(got) != 0 {
		t.Fatalf("got %+v, want no results for a missing root", got)
	}
}
