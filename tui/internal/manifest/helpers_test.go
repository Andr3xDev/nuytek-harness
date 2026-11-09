package manifest_test

import "github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"

var testKinds = []manifest.Kind{manifest.KindDir, manifest.KindDirX, manifest.KindFile, manifest.KindFileX, "mcp", "plugins"}

func comp(name string, kind manifest.Kind, source, target string, needs ...string) manifest.Component {
	return manifest.Component{Name: name, Kind: kind, Source: source, Target: target, Needs: needs}
}
