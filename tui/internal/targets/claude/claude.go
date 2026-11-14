package claude

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/target"
)

const (
	kindMCP     manifest.Kind = "mcp"
	kindPlugins manifest.Kind = "plugins"
)

const (
	checkJSON         = "json"
	checkHookScripts  = "hook-scripts"
	checkHookArgs     = "hook-args"
	checkMCPEnv       = "mcp-env"
	checkMCPCommands  = "mcp-commands"
	checkFrontmatter  = "frontmatter"
	backupDirHome     = ".claude-backups"
	backupDirRepo     = ".refresh-backups"
	componentSettings = "settings"
	componentHUD      = "claude-hud-config"
	componentAgents   = "agents"
	componentSkills   = "skills"
	componentMCP      = "mcp"
)

// mcpEnvPattern matches ${VAR} and ${VAR:-default}; group 1 is the name, group 2 the default.
var mcpEnvPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-[^}]*)?\}`)

func Spec() target.Spec {
	return target.Spec{
		ID:   "claude",
		Name: "Claude",
		Evaluators: drift.Evaluators(map[manifest.Kind]drift.Evaluator{
			kindMCP:     evaluateMCP,
			kindPlugins: evaluatePlugins,
		}),
		Special: map[manifest.Kind]plan.KindOps{
			kindMCP:     {Install: installMCP, Remove: removeMCP, Hard: hardInstallMCP},
			kindPlugins: {Install: installPlugins, Hard: hardInstallPlugins, Late: true, ExplicitUpdate: true},
		},
		Checks: checks,
		BackupDirs: func(home, repo string) []backup.Dir {
			return []backup.Dir{
				{Path: filepath.Join(home, backupDirHome), Label: "~/" + backupDirHome, Kind: backup.StoreHome, Base: home},
				{Path: filepath.Join(repo, backupDirRepo), Label: backupDirRepo, Kind: backup.StoreRepo, Base: repo},
			}
		},
	}
}

func checks(m manifest.Manifest) ([]check.Check, error) {
	lookup := func(name string) (manifest.Component, error) {
		i := slices.IndexFunc(m.Components, func(c manifest.Component) bool { return c.Name == name })
		if i < 0 {
			return manifest.Component{}, fmt.Errorf("manifest %s: component %q is required by the claude checks", m.Path, name)
		}
		return m.Components[i], nil
	}
	settings, err := lookup(componentSettings)
	if err != nil {
		return nil, err
	}
	hud, err := lookup(componentHUD)
	if err != nil {
		return nil, err
	}
	agents, err := lookup(componentAgents)
	if err != nil {
		return nil, err
	}
	skills, err := lookup(componentSkills)
	if err != nil {
		return nil, err
	}
	mcp, err := lookup(componentMCP)
	if err != nil {
		return nil, err
	}
	return []check.Check{
		check.ValidJSON(checkJSON, []check.JSONFile{
			{Path: settings.Target, Missing: check.StatusError},
			{Path: mcp.Target, Missing: check.StatusWarning},
			{Path: hud.Target, Missing: check.StatusWarning},
		}),
		check.Requirements(checkHookScripts, hookScriptRequirements(settings.Target)),
		check.Requirements(checkHookArgs, hookArgRequirements(settings.Target)),
		check.EnvRefs(checkMCPEnv, mcpEnvPattern, mcpEnvTexts(mcp.Source)),
		check.Requirements(checkMCPCommands, mcpCommandRequirements(mcp.Source)),
		check.Frontmatter(checkFrontmatter, []check.Set{
			{Root: agents.Target, Pattern: "*.md", Skip: []string{"README.md"}},
			{Root: agents.Target, Pattern: "sub-agents/*/*.md"},
			{Root: skills.Target, Pattern: "*/SKILL.md"},
		}),
		check.Dependencies(check.DependenciesID, m.AllNeeds, check.DefaultProbeTimeout),
	}, nil
}
