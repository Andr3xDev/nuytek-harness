package backup_test

import (
	"slices"
	"testing"
	"time"

	"github.com/Andr3xDev/nuytek-harness/tui/internal/backup"
)

func TestParseStamp(t *testing.T) {
	t.Run("local stamp is read in local time", func(t *testing.T) {
		got, ok := backup.ParseStamp("20240131-235959")
		want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.Local)
		if !ok || !got.Equal(want) || got.Location() != time.Local {
			t.Fatalf("got (%v, %v), want (%v, true) in time.Local", got, ok, want)
		}
	})

	t.Run("UTC stamp", func(t *testing.T) {
		got, ok := backup.ParseStamp("20240131T235959Z")
		want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		if !ok || !got.Equal(want) {
			t.Fatalf("got (%v, %v), want (%v, true)", got, ok, want)
		}
	})

	t.Run("UTC stamp with collision suffix", func(t *testing.T) {
		for _, name := range []string{"20240131T235959Z-1", "20240131T235959Z-1-1"} {
			got, ok := backup.ParseStamp(name)
			want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
			if !ok || !got.Equal(want) {
				t.Fatalf("ParseStamp(%q) = (%v, %v), want (%v, true)", name, got, ok, want)
			}
		}
	})

	t.Run("local stamp with collision suffix", func(t *testing.T) {
		got, ok := backup.ParseStamp("20240131-235959-1")
		want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.Local)
		if !ok || !got.Equal(want) {
			t.Fatalf("got (%v, %v), want (%v, true)", got, ok, want)
		}
	})

	for _, name := range []string{"manual-copy", "", "20240131", "20241301-000000", "20240131T235959Z-x", "20240131T235959Z-"} {
		t.Run("invalid "+name, func(t *testing.T) {
			if _, ok := backup.ParseStamp(name); ok {
				t.Fatalf("ParseStamp(%q) ok = true, want false", name)
			}
		})
	}
}

func TestNewStamp(t *testing.T) {
	now := time.Date(2024, 1, 31, 23, 59, 59, 0, time.FixedZone("UTC+5", 5*3600))
	taken := func(names ...string) func(string) bool {
		return func(s string) bool { return slices.Contains(names, s) }
	}
	local := now.Local().Format("20060102-150405")

	tests := []struct {
		name   string
		kind   backup.StoreKind
		exists func(string) bool
		want   string
	}{
		{"home is local time", backup.StoreHome, taken(), local},
		{"repo is UTC", backup.StoreRepo, taken(), "20240131T185959Z"},
		{"zero kind is home", 0, taken(), local},
		{"one collision", backup.StoreRepo, taken("20240131T185959Z"), "20240131T185959Z-1"},
		{"two collisions", backup.StoreRepo, taken("20240131T185959Z", "20240131T185959Z-1"), "20240131T185959Z-2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := backup.NewStamp(tt.kind, now, tt.exists); got != tt.want {
				t.Fatalf("NewStamp = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateStamp(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"", true},
		{"a/b", true},
		{".", true},
		{"..", true},
		{"20240131-235959", false},
		{"20240131T235959Z-1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := backup.ValidateStamp(tt.name); (err != nil) != tt.wantErr {
				t.Fatalf("ValidateStamp(%q) = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
