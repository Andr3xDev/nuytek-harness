package manifest_test

import (
	"testing"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/manifest"
)

func TestKind_IsDirNeedsExec(t *testing.T) {
	tests := []struct {
		kind     manifest.Kind
		wantDir  bool
		wantExec bool
	}{
		{manifest.KindDir, true, false},
		{manifest.KindDirX, true, true},
		{manifest.KindFile, false, false},
		{manifest.KindFileX, false, true},
		{"mcp", false, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			if got := tt.kind.IsDir(); got != tt.wantDir {
				t.Errorf("IsDir = %v, want %v", got, tt.wantDir)
			}
			if got := tt.kind.NeedsExec(); got != tt.wantExec {
				t.Errorf("NeedsExec = %v, want %v", got, tt.wantExec)
			}
		})
	}
}
