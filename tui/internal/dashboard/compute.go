package dashboard

import (
	"context"
	"fmt"
	"sync"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const MaxConcurrentEvaluations = 4

func Compute(ctx context.Context, env check.Env, t target.Target) Dashboard {
	d := Dashboard{Counts: map[check.Status]int{}, Err: t.Err}
	if t.Err != nil {
		return d
	}
	d.Results = check.Run(ctx, env, t.Checks)
	var missingTools []string
	for _, r := range d.Results {
		d.Counts[r.Status]++
		if r.CheckID == check.DependenciesID && r.Status == check.StatusError {
			missingTools = append(missingTools, r.Subject)
		}
	}
	d.MissingDeps = t.Manifest.MissingDeps(missingTools)
	d.Components = evaluateComponents(ctx, env, t)
	if t.Spec.BackupDirs != nil {
		d.LastBackup = backup.Latest(env.FS, t.Spec.BackupDirs(env.Home, env.Repo))
	}
	return d
}

func evaluateComponents(ctx context.Context, env check.Env, t target.Target) []drift.Status {
	statuses := make([]drift.Status, len(t.Manifest.Components))
	slots := make(chan struct{}, MaxConcurrentEvaluations)
	var wg sync.WaitGroup
	for i, c := range t.Manifest.Components {
		evaluate := t.Spec.Evaluators[c.Kind]
		if evaluate == nil {
			statuses[i] = drift.Status{Name: c.Name, State: drift.StateUnknown}
			continue
		}
		slots <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-slots }()
			defer func() {
				if r := recover(); r != nil {
					statuses[i] = drift.Status{Name: c.Name, State: drift.StateUnknown, Reason: fmt.Sprintf("evaluator panicked: %v", r)}
				}
			}()
			statuses[i] = evaluate(ctx, env, c)
		}()
	}
	wg.Wait()
	return statuses
}
