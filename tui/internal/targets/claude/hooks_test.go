package claude

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
)

const hooksSettingsRel = ".claude/settings.json"

func equalRefs(a, b HookRefs) bool {
	return a.Complex == b.Complex && a.ScriptPath == b.ScriptPath && a.ExecPath == b.ExecPath &&
		a.Command == b.Command && slices.Equal(a.ArgPaths, b.ArgPaths)
}

// The first six rows are the command hooks of claude/claude-settings.json, verbatim.
func TestParseHookCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want HookRefs
	}{
		{"real: python3 guard", `python3 '/home/andrex/.claude/hooks/infra-agent-bash-guard.py'`,
			HookRefs{ScriptPath: "/home/andrex/.claude/hooks/infra-agent-bash-guard.py"}},
		{"real: python3 engram guard", `python3 '/home/andrex/.claude/hooks/engram-tool-guard.py'`,
			HookRefs{ScriptPath: "/home/andrex/.claude/hooks/engram-tool-guard.py"}},
		{"real: bash script with word argument", `bash '/home/andrex/.claude/hooks/herdr-agent-state.sh' session`,
			HookRefs{ScriptPath: "/home/andrex/.claude/hooks/herdr-agent-state.sh"}},
		{"real: tilde executable", `~/.claude/hooks/stop-verify.sh`,
			HookRefs{ExecPath: "~/.claude/hooks/stop-verify.sh"}},
		{"real: bare command with word argument", `codegraph prompt-hook`,
			HookRefs{Command: "codegraph"}},
		{"real: bare command with sound file", `paplay /home/andrex/.claude/sounds/complete.wav`,
			HookRefs{Command: "paplay", ArgPaths: []string{"/home/andrex/.claude/sounds/complete.wav"}}},
		{"interpreter flag before script", `python3 -u ~/.claude/hooks/g.py --flag`,
			HookRefs{ScriptPath: "~/.claude/hooks/g.py"}},
		{"interpreter script plus path argument", `python3 '/x/g.py' ~/a.wav`,
			HookRefs{ScriptPath: "/x/g.py", ArgPaths: []string{"~/a.wav"}}},
		{"$HOME executable", `$HOME/.claude/hooks/a.sh`,
			HookRefs{ExecPath: "$HOME/.claude/hooks/a.sh"}},
		{"double-quoted ${HOME} executable", `"${HOME}/a.sh" arg`,
			HookRefs{ExecPath: "${HOME}/a.sh"}},
		{"quoted path with space", `node "/opt/my dir/x.js"`,
			HookRefs{ScriptPath: "/opt/my dir/x.js"}},
		{"metacharacter inside quotes is not complex", `python3 '/x/a|b.py'`,
			HookRefs{ScriptPath: "/x/a|b.py"}},
		{"relative, other env forms and --opt=path are ignored", `paplay --opt=/x/y.wav rel/file.wav ${CLAUDE_PLUGIN_ROOT}/s.wav`,
			HookRefs{Command: "paplay"}},
		{"$HOMEX is not $HOME", `$HOMEX/a.sh`, HookRefs{}},
		{"empty command", ``, HookRefs{}},
		{"pipe", `cat /x | grep y`, HookRefs{Complex: true}},
		{"and-and", `a && b`, HookRefs{Complex: true}},
		{"semicolon", `a; b`, HookRefs{Complex: true}},
		{"background ampersand", `a & b`, HookRefs{Complex: true}},
		{"command substitution", `echo $(date)`, HookRefs{Complex: true}},
		{"backtick", "echo `date`", HookRefs{Complex: true}},
		{"redirect out", `a > /tmp/f`, HookRefs{Complex: true}},
		{"redirect in", `a < /tmp/f`, HookRefs{Complex: true}},
		{"inline code -c is not a script", `bash -c "/usr/bin/foo bar"`, HookRefs{}},
		{"inline code -e is not a script", `node -e "/re/.test(x)"`, HookRefs{}},
		{"combined inline flag -lc", `bash -lc '/x'`, HookRefs{}},
		{"relative executable with slash is ignored", `./x.sh`, HookRefs{}},
		{"unquoted backslash", `echo a\ b`, HookRefs{Complex: true}},
		{"unquoted newline", "echo a\necho b", HookRefs{Complex: true}},
		{"substitution inside double quotes", `echo "$(date)"`, HookRefs{Complex: true}},
		{"backtick inside double quotes", "echo \"`date`\"", HookRefs{Complex: true}},
		{"substitution inside single quotes is literal", `echo '$(date)'`, HookRefs{Command: "echo"}},
		{"unquoted glob", `paplay /x/*.wav`, HookRefs{Complex: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseHookCommand(tt.cmd); !equalRefs(got, tt.want) {
				t.Errorf("ParseHookCommand(%q) = %+v, want %+v", tt.cmd, got, tt.want)
			}
		})
	}
}

const hooksSettings = `{
  "hooks": {
    "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "codegraph prompt-hook"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "$HOME/.claude/hooks/stop-verify.sh"}]}],
    "SessionStart": [{"matcher": "*", "hooks": [{"type": "command", "command": "echo a | tee b"}]}],
    "PreToolUse": [{"matcher": "Bash", "hooks": [
      {"type": "prompt", "prompt": "python3 '/nowhere/ignored.py'"},
      {"type": "command", "command": "python3 '~/.claude/hooks/guard.py'"}
    ]}],
    "Notification": [{"matcher": "agent_completed", "hooks": [
      {"type": "command", "command": "paplay ~/.claude/sounds/done.oga"}
    ]}]
  }
}`

func TestHookRequirements(t *testing.T) {
	env := newEnv(t)
	writeFile(t, env.Home, ".claude/settings.json", hooksSettings)
	home := func(rel string) string { return filepath.Join(env.Home, rel) }

	tests := []struct {
		name    string
		extract func(check.Env) ([]check.Requirement, error)
		want    []check.Requirement
	}{
		{"scripts and commands", hookScriptRequirements(hooksSettingsRel), []check.Requirement{
			{Subject: "Notification[agent_completed]", Command: "paplay"},
			{Subject: "PreToolUse[Bash]", Path: home(".claude/hooks/guard.py")},
			{Subject: "SessionStart[*]", Unanalyzed: complexCommandMessage},
			{Subject: "Stop", Path: home(".claude/hooks/stop-verify.sh"), MustExec: true},
			{Subject: "UserPromptSubmit", Command: "codegraph"},
		}},
		{"file arguments", hookArgRequirements(hooksSettingsRel), []check.Requirement{
			{Subject: "Notification[agent_completed]", Path: home(".claude/sounds/done.oga")},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.extract(env)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("requirements = %+v\nwant         %+v", got, tt.want)
			}
		})
	}
}

func TestHookRequirements_SettingsEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content *string
		wantErr bool
	}{
		{"missing settings yields no requirements", nil, false},
		{"invalid JSON is an error", ptr(`{"hooks": `), true},
	}
	for _, tt := range tests {
		for name, extract := range map[string]func(check.Env) ([]check.Requirement, error){
			"scripts": hookScriptRequirements(hooksSettingsRel), "args": hookArgRequirements(hooksSettingsRel),
		} {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				env := newEnv(t)
				if tt.content != nil {
					writeFile(t, env.Home, ".claude/settings.json", *tt.content)
				}
				got, err := extract(env)
				if (err != nil) != tt.wantErr {
					t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
				}
				if len(got) != 0 {
					t.Errorf("requirements = %+v, want none", got)
				}
			})
		}
	}
}

func ptr(s string) *string { return &s }
