package check_test

import (
	"context"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

func okCheck(id string) check.Check {
	return check.Check{ID: id, Run: func(context.Context, check.Env) []check.Result {
		return []check.Result{{CheckID: id, Subject: id, Status: check.StatusOK}}
	}}
}
