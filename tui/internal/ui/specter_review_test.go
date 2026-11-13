package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/specter"
)

const (
	extraStores    = 3
	scrollPresses  = 1000
	shortScreenRow = 12
)

func TestSpecterScreen_AcceptedFastInitDoesNotReappearAfterTheRun(t *testing.T) {
	r := screenRoot(t, facts2Candidates(), &specterFakes{})

	press(r, textKey("f"))
	deliver(r, press(r, enterKey))
	cmd := press(r, enterKey)
	if cmd == nil {
		t.Fatal("accepting the preview returned no command")
	}
	r.Update(cmd())
	r.run = nil // the output pane dismissed

	if r.specterFlow != nil || strings.Contains(view(r), "beta-store") {
		t.Fatalf("select flow came back after the run:\n%s", view(r))
	}
}

func TestSpecterScreen_ValidatesMoreStoresThanTheConcurrencyLimit(t *testing.T) {
	facts := baseFacts()
	for i := 0; i < specter.MaxConcurrentOpenspec+extraStores; i++ {
		facts.Registered = append(facts.Registered, specter.StoreInfo{ID: fmt.Sprintf("store-%d", i)})
	}
	fakes := &specterFakes{validation: validFixture()}
	r := screenRoot(t, facts, fakes)

	deliver(r, press(r, textKey("v")))

	if want := len(facts.Registered); fakes.validateCalls != want {
		t.Fatalf("validated %d stores, want %d", fakes.validateCalls, want)
	}
	for id, sv := range r.specter.validation {
		if sv.running {
			t.Errorf("store %s still validating", id)
		}
	}
}

func TestSpecterScreen_ScrollIsBoundedAndAdvertised(t *testing.T) {
	facts := baseFacts()
	r := screenRoot(t, facts, &specterFakes{validation: validFixture()})
	r.Update(tea.WindowSizeMsg{Width: 160, Height: shortScreenRow})
	deliver(r, press(r, textKey("v")))
	rootPath := facts.Root.Path

	if keys, _ := r.keybarKeys(); !claims(keys, "down") {
		t.Fatalf("keybar keys %v do not advertise move", keys)
	}
	if !strings.Contains(view(r), rootPath) {
		t.Fatalf("top of the screen lacks the root path:\n%s", view(r))
	}
	for range scrollPresses {
		press(r, downKey)
	}
	if bottom := view(r); strings.Contains(bottom, rootPath) || !strings.Contains(bottom, changeInvalid) {
		t.Fatalf("scrolled to the end, view:\n%s", bottom)
	}
	for range scrollPresses {
		press(r, upKey)
	}
	if !strings.Contains(view(r), rootPath) {
		t.Fatalf("scrolled back to the top, view:\n%s", view(r))
	}
}

func TestRoot_RefreshOnACLIPanelUpdatesDashboardAndManageTogether(t *testing.T) {
	probe := &manageProbe{}
	r := manageRoot(t, probe, plainSections("MCPs"))
	r.focus, r.tab = focusPanel, 1
	probe.calls = 0

	cmd := press(r, textKey("r"))
	for _, msg := range flattenMsgs(cmd()) {
		r.Update(msg)
	}

	if probe.calls != 1 {
		t.Errorf("Manage ran %d times on refresh, want 1", probe.calls)
	}
	if r.states[0].phase != ready {
		t.Errorf("dashboard phase = %v after refresh, want ready", r.states[0].phase)
	}
}
