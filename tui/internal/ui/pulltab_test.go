package ui

import "testing"

func TestPullTab_ListsOffTheUILoopAndDropsStaleResults(t *testing.T) {
	svc := newRoot(t, availableTarget()).svc
	p := newPullTab(svc, &targetState{}, availableTarget())

	first := p.Activate(availableTarget())
	if first == nil || p.listed {
		t.Fatalf("Activate cmd = %v, listed = %v; want a background cmd and no listing yet", first, p.listed)
	}
	second := p.Activate(availableTarget())
	staleEpoch := first().(pullListed).epoch
	currentEpoch := second().(pullListed).epoch

	p.Update(pullListed{p: p, epoch: staleEpoch, names: []string{"stale"}})
	if p.listed {
		t.Fatal("a superseded result was applied")
	}
	p.Update(pullListed{p: p, epoch: currentEpoch, names: []string{"agents"}})
	if !p.listed || len(p.pick.choices) != 1 || p.pick.choices[0].name != "agents" {
		t.Errorf("listed = %v, choices = %v; want the current result applied", p.listed, p.pick.choices)
	}
}
