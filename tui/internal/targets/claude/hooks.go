package claude

import (
	"errors"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const (
	complexCommandMessage  = "complex command, not analyzed"
	commandHookType        = "command"
	unquotedComplexChars   = "|&;<>`\\\n*?["
	inlineCodeFlags        = "cep"
	homeVarBraced, homeVar = "${HOME}", "$HOME"
	homeTilde              = "~"
)

var (
	interpreters = []string{"python3", "python", "bash", "sh", "node"}
	homePrefixes = []string{homeVarBraced, homeVar, homeTilde}
)

// HookRefs holds the raw (unexpanded) references found in one hook command.
type HookRefs struct {
	Complex    bool     // unquoted shell metacharacter: not analyzed
	ScriptPath string   // script run through an interpreter: must exist
	ExecPath   string   // path-like executable run directly: must exist and be executable
	Command    string   // bare executable: must resolve on PATH
	ArgPaths   []string // other path-like arguments: must exist
}

// ParseHookCommand classifies a hook command into the paths and commands that must exist; anything it cannot analyze safely is returned as Complex.
func ParseHookCommand(cmd string) HookRefs {
	tokens, complex := tokenize(cmd)
	if complex {
		return HookRefs{Complex: true}
	}
	for len(tokens) > 0 && strings.Contains(tokens[0], "=") && !isPathLike(tokens[0]) {
		tokens = tokens[1:]
	}
	if len(tokens) == 0 || (strings.HasPrefix(tokens[0], "$") && !isPathLike(tokens[0])) {
		return HookRefs{}
	}

	var refs HookRefs
	rest := tokens[1:]
	switch {
	case slices.Contains(interpreters, tokens[0]):
		i := slices.IndexFunc(rest, func(t string) bool { return !strings.HasPrefix(t, "-") })
		if i < 0 || slices.ContainsFunc(rest[:i], isInlineCodeFlag) {
			return refs
		}
		if isPathLike(rest[i]) {
			refs.ScriptPath = rest[i]
		}
		rest = rest[i+1:]
	case isPathLike(tokens[0]):
		refs.ExecPath = tokens[0]
	case strings.Contains(tokens[0], "/"):
		return refs
	default:
		refs.Command = tokens[0]
	}
	for _, arg := range rest {
		if isPathLike(arg) {
			refs.ArgPaths = append(refs.ArgPaths, arg)
		}
	}
	return refs
}

// isInlineCodeFlag reports short flags (-c, -e, -p, or combined like -lc) whose next argument is code, not a script.
func isInlineCodeFlag(token string) bool {
	return strings.HasPrefix(token, "-") && !strings.HasPrefix(token, "--") && strings.ContainsAny(token, inlineCodeFlags)
}

// tokenize splits on unquoted spaces and tabs honoring single and double quotes.
// shortcut: backslashes, unquoted newlines and globs, and $( or backticks inside double quotes are reported as complex, model them if a real hook needs one.
func tokenize(cmd string) (tokens []string, complex bool) {
	var token strings.Builder
	var quote rune
	inToken := false
	flush := func() {
		if inToken {
			tokens = append(tokens, token.String())
			token.Reset()
			inToken = false
		}
	}
	runes := []rune(cmd)
	for i, r := range runes {
		switch {
		case quote != 0:
			switch {
			case r == quote:
				quote = 0
			case quote == '"' && (r == '`' || startsSubstitution(runes, i)):
				return nil, true
			default:
				token.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inToken = r, true
		case strings.ContainsRune(unquotedComplexChars, r), startsSubstitution(runes, i):
			return nil, true
		case r == ' ' || r == '\t':
			flush()
		default:
			token.WriteRune(r)
			inToken = true
		}
	}
	flush()
	return tokens, false
}

func startsSubstitution(runes []rune, i int) bool {
	return runes[i] == '$' && i+1 < len(runes) && runes[i+1] == '('
}

func isPathLike(token string) bool {
	_, ok := cutHomePrefix(token)
	return ok || strings.HasPrefix(token, "/")
}

// cutHomePrefix only matches a whole home reference, so $HOMEX is not $HOME.
func cutHomePrefix(path string) (rest string, ok bool) {
	for _, prefix := range homePrefixes {
		if rest, found := strings.CutPrefix(path, prefix); found && (rest == "" || strings.HasPrefix(rest, "/")) {
			return rest, true
		}
	}
	return "", false
}

func expandHome(home, path string) string {
	if rest, ok := cutHomePrefix(path); ok {
		return filepath.Join(home, rest)
	}
	return path
}

// hookScriptRequirements and hookArgRequirements read the hooks from settingsPath (home-relative).
func hookScriptRequirements(settingsPath string) func(check.Env) ([]check.Requirement, error) {
	return func(env check.Env) ([]check.Requirement, error) { return hookScripts(env, settingsPath) }
}

func hookArgRequirements(settingsPath string) func(check.Env) ([]check.Requirement, error) {
	return func(env check.Env) ([]check.Requirement, error) { return hookArgs(env, settingsPath) }
}

func hookScripts(env check.Env, settingsPath string) ([]check.Requirement, error) {
	return hookRequirements(env, settingsPath, func(subject string, refs HookRefs) []check.Requirement {
		switch {
		case refs.Complex:
			return []check.Requirement{{Subject: subject, Unanalyzed: complexCommandMessage}}
		case refs.ScriptPath != "":
			return []check.Requirement{{Subject: subject, Path: expandHome(env.Home, refs.ScriptPath)}}
		case refs.ExecPath != "":
			return []check.Requirement{{Subject: subject, Path: expandHome(env.Home, refs.ExecPath), MustExec: true}}
		case refs.Command != "":
			return []check.Requirement{{Subject: subject, Command: refs.Command}}
		}
		return nil
	})
}

func hookArgs(env check.Env, settingsPath string) ([]check.Requirement, error) {
	return hookRequirements(env, settingsPath, func(subject string, refs HookRefs) []check.Requirement {
		var reqs []check.Requirement
		for _, arg := range refs.ArgPaths {
			reqs = append(reqs, check.Requirement{Subject: subject, Path: expandHome(env.Home, arg)})
		}
		return reqs
	})
}

type hookSettings struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks   []struct {
			Type    string `json:"type"`
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}

// hookRequirements walks events alphabetically and hooks in declaration order; a missing settings file has no hooks.
func hookRequirements(env check.Env, settingsPath string, build func(subject string, refs HookRefs) []check.Requirement) ([]check.Requirement, error) {
	var settings hookSettings
	if err := readJSON(env.FS, filepath.Join(env.Home, settingsPath), &settings); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var reqs []check.Requirement
	for _, event := range slices.Sorted(maps.Keys(settings.Hooks)) {
		for _, group := range settings.Hooks[event] {
			subject := event
			if group.Matcher != "" {
				subject += "[" + group.Matcher + "]"
			}
			for _, hook := range group.Hooks {
				if hook.Type == commandHookType {
					reqs = append(reqs, build(subject, ParseHookCommand(hook.Command))...)
				}
			}
		}
	}
	return reqs, nil
}
