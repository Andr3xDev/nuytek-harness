package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const componentHeader = "[[component]]"

type rawFile struct {
	Component []rawComponent
}

// Needs is a pointer so an omitted key is distinguishable from needs = [].
type rawComponent struct {
	Name   string
	Kind   Kind
	Source string
	Target string
	Needs  *[]string
	Pull   bool
}

// Load parses and validates the manifest at path. Errors read "path:LINE: message".
func Load(path string, kinds []Kind) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}
	var raw rawFile
	dec := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return Manifest{}, decodeError(path, err)
	}

	// shortcut: headers found by line scan, so a "[[component]]" line inside a multi-line string shifts the mapping; switch to a position-aware TOML decoder if manifests ever need multi-line strings.
	headers := headerLines(data)
	m := Manifest{Path: path}
	for i, rc := range raw.Component {
		line := 0
		if i < len(headers) {
			line = headers[i]
		}
		if rc.Needs == nil {
			return Manifest{}, positioned(path, line, "component %q: needs is required (use [] if none)", rc.Name)
		}
		m.Components = append(m.Components, Component{Name: rc.Name, Kind: rc.Kind, Source: rc.Source, Target: rc.Target, Needs: *rc.Needs, Pull: rc.Pull})
		// Validating each growing prefix pins the first failure to its component.
		if err := Validate(m, kinds); err != nil {
			return Manifest{}, positioned(path, line, "%v", err)
		}
	}
	return m, nil
}

func headerLines(data []byte) []int {
	var lines []int
	for n, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), componentHeader) {
			lines = append(lines, n+1)
		}
	}
	return lines
}

func decodeError(path string, err error) error {
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) && len(strict.Errors) > 0 {
		e := strict.Errors[0]
		line, _ := e.Position()
		return positioned(path, line, "unknown field %q", strings.Join(e.Key(), "."))
	}
	var de *toml.DecodeError
	if errors.As(err, &de) {
		line, _ := de.Position()
		return positioned(path, line, "%s", de.Error())
	}
	return fmt.Errorf("%s: %w", path, err)
}

func positioned(path string, line int, format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	if line == 0 {
		return fmt.Errorf("%s: %s", path, msg)
	}
	return fmt.Errorf("%s:%d: %s", path, line, msg)
}
