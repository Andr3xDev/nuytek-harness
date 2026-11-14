package specter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const (
	OpenspecQueryTimeout    = 10 * time.Second
	OpenspecValidateTimeout = 60 * time.Second
	RegisterTimeout         = 30 * time.Second
	MaxConcurrentOpenspec   = 4

	stderrSnippetLines = 3
	stdoutSnippetBytes = 200
)

// openspecJSON decodes stdout into out. A non-zero exit is a result only when allowNonZero is set and stdout is valid JSON.
func openspecJSON(ctx context.Context, exec check.Exec, out any, allowNonZero bool, args ...string) error {
	cmd := "openspec " + strings.Join(args, " ")
	res, err := exec.Run(ctx, "openspec", args...)
	if err != nil {
		return fmt.Errorf("%s: %w", cmd, err)
	}
	decodeErr := json.Unmarshal(res.Stdout, out)
	if res.ExitCode == 0 {
		if decodeErr != nil {
			return fmt.Errorf("%s: output is not JSON: %s", cmd, snippet(res.Stdout, stdoutSnippetBytes))
		}
		return nil
	}
	if allowNonZero && decodeErr == nil {
		return nil
	}
	return fmt.Errorf("%s: exit %d: %s", cmd, res.ExitCode, firstLines(res.Stderr, stderrSnippetLines))
}

func snippet(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return strings.ToValidUTF8(s[:max], "")
	}
	return s
}

func firstLines(b []byte, n int) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
