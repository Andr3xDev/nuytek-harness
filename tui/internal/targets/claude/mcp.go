package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/check"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/drift"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
	"github.com/Andr3xDev/nuytek-harness/tui/internal/plan"
)

const mcpServersKey = "mcpServers"

// ownedServers reads the harness-owned server definitions with their raw JSON values.
func ownedServers(fs check.FS, repo string, c manifest.Component) (map[string]json.RawMessage, error) {
	var owned map[string]json.RawMessage
	err := readJSON(fs, filepath.Join(repo, c.Source), &owned)
	return owned, err
}

// editServers applies change to the mcpServers object and re-encodes the file: values are kept,
// top-level keys come out sorted and the document is reindented.
func editServers(old []byte, change func(servers map[string]json.RawMessage) map[string]json.RawMessage) ([]byte, error) {
	top, servers, err := parseServers(old)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(top)+1)
	for key, value := range top {
		out[key] = value
	}
	out[mcpServersKey] = change(servers)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// parseServers decodes the top level and mcpServers; empty input is an empty object.
func parseServers(data []byte) (top, servers map[string]json.RawMessage, err error) {
	top = map[string]json.RawMessage{}
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &top); err != nil {
			return nil, nil, err
		}
	}
	if top == nil {
		top = map[string]json.RawMessage{}
	}
	servers = map[string]json.RawMessage{}
	if raw, ok := top[mcpServersKey]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return nil, nil, err
		}
	}
	if servers == nil {
		servers = map[string]json.RawMessage{}
	}
	return top, servers, nil
}

func mergeChange(owned map[string]json.RawMessage) func(map[string]json.RawMessage) map[string]json.RawMessage {
	return func(servers map[string]json.RawMessage) map[string]json.RawMessage {
		maps.Copy(servers, owned)
		return servers
	}
}

func replaceChange(owned map[string]json.RawMessage) func(map[string]json.RawMessage) map[string]json.RawMessage {
	return func(map[string]json.RawMessage) map[string]json.RawMessage {
		return maps.Clone(owned)
	}
}

// MergeServers sets the owned servers and keeps every other server and top-level key.
func MergeServers(old []byte, owned map[string]json.RawMessage) ([]byte, error) {
	return editServers(old, mergeChange(owned))
}

// ReplaceServers sets mcpServers to exactly the owned servers and keeps every other top-level key.
func ReplaceServers(old []byte, owned map[string]json.RawMessage) ([]byte, error) {
	return editServers(old, replaceChange(owned))
}

// currentServers parses the deployed file at plan time so an invalid file fails before the preview.
func currentServers(env check.Env, c manifest.Component) (map[string]json.RawMessage, error) {
	path := filepath.Join(env.Home, c.Target)
	data, err := env.FS.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	_, servers, err := parseServers(data)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return servers, nil
}

func serverEdit(env check.Env, c manifest.Component, rows []plan.Change, change func(map[string]json.RawMessage) map[string]json.RawMessage) plan.Part {
	path := filepath.Join(env.Home, c.Target)
	return plan.Part{Steps: []plan.Step{plan.Edit{
		Path: path,
		Rows: rows,
		Edit: func(old []byte, _ bool) ([]byte, error) {
			out, err := editServers(old, change)
			if err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
			return out, nil
		},
	}}}
}

func serverRow(c manifest.Component, effect plan.Effect, key string) plan.Change {
	return plan.Change{Effect: effect, Path: "~/" + c.Target + "#" + mcpServersKey + "." + key}
}

// ownedRows lists create and overwrite rows for the owned servers that differ from the deployed ones.
func ownedRows(c manifest.Component, current, owned map[string]json.RawMessage) []plan.Change {
	var rows []plan.Change
	for _, key := range slices.Sorted(maps.Keys(owned)) {
		switch deployed, ok := current[key]; {
		case !ok:
			rows = append(rows, serverRow(c, plan.EffectCreate, key))
		case !sameJSON(deployed, owned[key]):
			rows = append(rows, serverRow(c, plan.EffectOverwrite, key))
		}
	}
	return rows
}

func sameJSON(a, b json.RawMessage) bool {
	var x, y any
	return json.Unmarshal(a, &x) == nil && json.Unmarshal(b, &y) == nil && reflect.DeepEqual(x, y)
}

// planServers returns the repo-owned servers and the deployed ones, the input of every MCP plan.
func planServers(env check.Env, c manifest.Component) (owned, current map[string]json.RawMessage, err error) {
	if owned, err = ownedServers(env.FS, env.Repo, c); err != nil {
		return nil, nil, err
	}
	if current, err = currentServers(env, c); err != nil {
		return nil, nil, err
	}
	return owned, current, nil
}

func installMCP(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	owned, current, err := planServers(env, c)
	if err != nil {
		return plan.Part{}, err
	}
	rows := ownedRows(c, current, owned)
	if len(rows) == 0 {
		return plan.Part{}, nil
	}
	return serverEdit(env, c, rows, mergeChange(owned)), nil
}

func removeMCP(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	owned, current, err := planServers(env, c)
	if err != nil {
		return plan.Part{}, err
	}
	var rows []plan.Change
	for _, key := range slices.Sorted(maps.Keys(owned)) {
		if _, ok := current[key]; ok {
			rows = append(rows, serverRow(c, plan.EffectDelete, key))
		}
	}
	if len(rows) == 0 {
		return plan.Part{}, nil
	}
	return serverEdit(env, c, rows, func(servers map[string]json.RawMessage) map[string]json.RawMessage {
		for key := range owned {
			delete(servers, key)
		}
		return servers
	}), nil
}

func hardInstallMCP(_ context.Context, env check.Env, c manifest.Component) (plan.Part, error) {
	owned, current, err := planServers(env, c)
	if err != nil {
		return plan.Part{}, err
	}
	var rows []plan.Change
	for _, key := range slices.Sorted(maps.Keys(current)) {
		if _, ok := owned[key]; !ok {
			rows = append(rows, serverRow(c, plan.EffectDelete, key))
		}
	}
	rows = append(rows, ownedRows(c, current, owned)...)
	if len(rows) == 0 {
		return plan.Part{}, nil
	}
	return serverEdit(env, c, rows, replaceChange(owned)), nil
}

// mcpExpandedKeys are the server keys where Claude Code expands ${VAR}.
var mcpExpandedKeys = []string{"command", "args", "env", "url", "headers"}

type mcpServers map[string]map[string]any

// readMCPSource reads the harness-owned server definitions at the repo-relative source.
func readMCPSource(env check.Env, source string) (mcpServers, error) {
	var servers mcpServers
	err := readJSON(env.FS, filepath.Join(env.Repo, source), &servers)
	return servers, err
}

func mcpEnvTexts(source string) func(check.Env) ([]check.TextSource, error) {
	return func(env check.Env) ([]check.TextSource, error) { return mcpTexts(env, source) }
}

func mcpCommandRequirements(source string) func(check.Env) ([]check.Requirement, error) {
	return func(env check.Env) ([]check.Requirement, error) { return mcpCommands(env, source) }
}

func mcpTexts(env check.Env, source string) ([]check.TextSource, error) {
	servers, err := readMCPSource(env, source)
	if err != nil {
		return nil, err
	}
	var sources []check.TextSource
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		var leaves []string
		for _, key := range mcpExpandedKeys {
			leaves = appendStringLeaves(leaves, servers[name][key])
		}
		sources = append(sources, check.TextSource{Subject: name, Text: strings.Join(leaves, "\n")})
	}
	return sources, nil
}

func appendStringLeaves(leaves []string, value any) []string {
	switch v := value.(type) {
	case string:
		return append(leaves, v)
	case []any:
		for _, item := range v {
			leaves = appendStringLeaves(leaves, item)
		}
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(v)) {
			leaves = appendStringLeaves(leaves, v[key])
		}
	}
	return leaves
}

func mcpCommands(env check.Env, source string) ([]check.Requirement, error) {
	servers, err := readMCPSource(env, source)
	if err != nil {
		return nil, err
	}
	var reqs []check.Requirement
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		if command, _ := servers[name]["command"].(string); command != "" {
			reqs = append(reqs, check.Requirement{Subject: name, Command: command})
		}
	}
	return reqs, nil
}

func evaluateMCP(_ context.Context, env check.Env, c manifest.Component) drift.Status {
	status := func(state drift.State, reason string) drift.Status {
		return drift.Status{Name: c.Name, State: state, Reason: reason}
	}

	owned, err := readMCPSource(env, c.Source)
	if err != nil {
		return status(drift.StateUnknown, err.Error())
	}
	var deployed struct {
		MCPServers mcpServers `json:"mcpServers"`
	}
	if err := readJSON(env.FS, filepath.Join(env.Home, c.Target), &deployed); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return status(drift.StateNotInstalled, "")
		}
		return status(drift.StateUnknown, err.Error())
	}

	var differing []string
	present := false
	for _, name := range slices.Sorted(maps.Keys(owned)) {
		got, ok := deployed.MCPServers[name]
		present = present || ok
		if !ok || !reflect.DeepEqual(got, owned[name]) {
			differing = append(differing, name)
		}
	}
	switch {
	case !present:
		return status(drift.StateNotInstalled, "")
	case len(differing) > 0:
		return status(drift.StateDrift, "differs or missing: "+strings.Join(differing, ","))
	}
	return status(drift.StateInstalled, "")
}
