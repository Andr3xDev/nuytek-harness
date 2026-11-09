package dashboard_test

import (
	"reflect"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/dashboard"
)

func TestDashboard_Issues(t *testing.T) {
	warn, err1, err2, ok := resultOf("w", check.StatusWarning), resultOf("e1", check.StatusError), resultOf("e2", check.StatusError), resultOf("o", check.StatusOK)

	t.Run("errors first and ok excluded", func(t *testing.T) {
		got := dashboard.Dashboard{Results: []check.Result{warn, err1, ok, err2}}.Issues()
		want := []check.Result{err1, err2, warn}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	})

	t.Run("empty when all ok", func(t *testing.T) {
		if got := (dashboard.Dashboard{Results: []check.Result{ok}}).Issues(); len(got) != 0 {
			t.Fatalf("got %v, want empty", got)
		}
	})
}
