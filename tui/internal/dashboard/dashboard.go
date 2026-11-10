package dashboard

import (
	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

type Dashboard struct {
	Counts      map[check.Status]int
	Results     []check.Result
	Components  []drift.Status
	MissingDeps []manifest.MissingDep
	LastBackup  *backup.Info // nil = no backups
	Err         error
}

// Issues returns the non-ok results, errors first.
func (d Dashboard) Issues() []check.Result {
	var errs, others []check.Result
	for _, r := range d.Results {
		switch r.Status {
		case check.StatusOK:
		case check.StatusError:
			errs = append(errs, r)
		default:
			others = append(others, r)
		}
	}
	return append(errs, others...)
}
