package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

func availableTarget() target.Target {
	return target.Target{Spec: target.Spec{
		ID:     "t",
		Name:   "T",
		Checks: func(manifest.Manifest) ([]check.Check, error) { return nil, nil },
	}}
}

func newRoot(t *testing.T, targets ...target.Target) *Root {
	t.Helper()
	return NewRoot(targets, check.Env{}, false)
}

func textKey(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

var (
	enterKey = tea.KeyPressMsg{Code: tea.KeyEnter}
	escKey   = tea.KeyPressMsg{Code: tea.KeyEscape}
	ctrlC    = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

func press(r *Root, k tea.KeyPressMsg) tea.Cmd {
	_, cmd := r.Update(k)
	return cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}
