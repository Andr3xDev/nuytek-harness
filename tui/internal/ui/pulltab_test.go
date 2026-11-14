package ui

import "testing"

func TestPullTab_ListsOffTheUILoopAndDropsStaleResults(t *testing.T) {
	svc := newRoot(t, availableTarget()).svc
	p := newPullTab(svc, &targetState{}, availableTarget())

	first := p.Activate(availableTarget())
	if first == nil || p.listed {
		t.Fatalf("Activate cmd = %v, listed = %v; want a background cmd and no listing yet", first, p.listed)
	}
	p.Activate(availableTarget())

	p.Update(pullListed{p: p, epoch: 1, names: []string{"stale"}})
	if p.listed {
		t.Fatal("a superseded result was applied")
	}
	p.Update(pullListed{p: p, epoch: 2, names: []string{"agents"}})
	if !p.listed || len(p.pick.choices) != 1 || p.pick.choices[0].name != "agents" {
		t.Errorf("listed = %v, choices = %v; want the current result applied", p.listed, p.pick.choices)
	}
}

func TestPullTab_DashboardReadyOnlyMarksDirty(t *testing.T) {
	svc := newRoot(t, availableTarget()).svc
	st := &targetState{}
	p := newPullTab(svc, st, availableTarget())

	if _, cmd := p.Update(dashboardReady{st: st}); cmd != nil || !p.dirty {
		t.Fatalf("cmd = %v, dirty = %v; want no load and a dirty tab", cmd, p.dirty)
	}
	if p.load(); p.dirty {
		t.Error("load left the tab dirty")
	}
}
