package check

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// Run runs every check concurrently; a panicking check becomes one error result.
func Run(ctx context.Context, env Env, checks []Check) []Result {
	perCheck := make([][]Result, len(checks))
	var wg sync.WaitGroup
	for i, c := range checks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					perCheck[i] = []Result{{CheckID: c.ID, Subject: c.ID, Status: StatusError, Message: fmt.Sprintf("check panicked: %v", r)}}
				}
			}()
			perCheck[i] = c.Run(ctx, env)
		}()
	}
	wg.Wait()
	return slices.Concat(perCheck...)
}
