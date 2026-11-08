package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

type CheckResult struct {
	CheckID, Subject, Message string
	Status                    Status
}

type Kind string

const (
	KindDir   Kind = "dir"
	KindDirX  Kind = "dirx"
	KindFile  Kind = "file"
	KindFileX Kind = "filex"
)

func (k Kind) IsDir() bool     { return k == KindDir || k == KindDirX }
func (k Kind) NeedsExec() bool { return k == KindDirX || k == KindFileX }

type Component struct {
	Name           string
	Kind           Kind
	Source, Target string
	Needs          []string
}

type Manifest struct {
	Target, Path string
	Components   []Component
}

func (m Manifest) TargetOf(name string) (string, bool) {
	for _, c := range m.Components {
		if c.Name == name {
			return c.Target, true
		}
	}
	return "", false
}

func (m Manifest) AllNeeds() []string {
	var needs []string
	for _, c := range m.Components {
		needs = append(needs, c.Needs...)
	}
	slices.Sort(needs)
	return slices.Compact(needs)
}

func ValidateManifest(m Manifest, allowedKinds []Kind) error {
	seen := map[string]bool{}
	for i, c := range m.Components {
		switch {
		case c.Name == "":
			return fmt.Errorf("component %d: empty name", i)
		case c.Kind == "":
			return fmt.Errorf("component %q: empty kind", c.Name)
		case c.Source == "":
			return fmt.Errorf("component %q: empty source", c.Name)
		case c.Target == "":
			return fmt.Errorf("component %q: empty target", c.Name)
		case !slices.Contains(allowedKinds, c.Kind):
			return fmt.Errorf("component %q: unknown kind %q", c.Name, c.Kind)
		case seen[c.Name]:
			return fmt.Errorf("duplicate component name %q", c.Name)
		}
		seen[c.Name] = true
		if !isRelativeInside(c.Source) {
			return fmt.Errorf("component %q: source %q must be relative without \"..\"", c.Name, c.Source)
		}
		if !isRelativeInside(c.Target) {
			return fmt.Errorf("component %q: target %q must be relative without \"..\"", c.Name, c.Target)
		}
	}
	return nil
}

func isRelativeInside(p string) bool {
	return !strings.HasPrefix(p, "/") && !slices.Contains(strings.Split(p, "/"), "..")
}

type MissingDep struct {
	Tool       string
	Dependents []string
}

func MissingDeps(m Manifest, results []CheckResult) []MissingDep {
	var missing []MissingDep
	for _, r := range results {
		if r.Status != StatusError {
			continue
		}
		dep := MissingDep{Tool: r.Subject}
		for _, c := range m.Components {
			if slices.Contains(c.Needs, r.Subject) {
				dep.Dependents = append(dep.Dependents, c.Name)
			}
		}
		missing = append(missing, dep)
	}
	slices.SortFunc(missing, func(a, b MissingDep) int { return strings.Compare(a.Tool, b.Tool) })
	return missing
}

const (
	localStampLayout = "20060102-150405"
	utcStampLayout   = "20060102T150405Z"
)

func ParseBackupStamp(name string) (time.Time, bool) {
	if t, err := time.ParseInLocation(localStampLayout, name, time.Local); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(utcStampLayout, name, time.UTC); err == nil {
		return t, true
	}
	return time.Time{}, false
}

type ComponentState string

const (
	StateInstalled    ComponentState = "installed"
	StateDrift        ComponentState = "drift"
	StateNotInstalled ComponentState = "not_installed"
	StateUnknown      ComponentState = "unknown"
)

type ComponentStatus struct {
	Name   string
	State  ComponentState
	Reason string
}

type State struct {
	RepoPath string
}
