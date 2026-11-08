package domain

import (
	"testing"
	"time"
)

func TestParseBackupStamp(t *testing.T) {
	t.Run("local stamp is read in local time", func(t *testing.T) {
		got, ok := ParseBackupStamp("20240131-235959")
		want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.Local)
		if !ok || !got.Equal(want) || got.Location() != time.Local {
			t.Fatalf("got (%v, %v), want (%v, true) in time.Local", got, ok, want)
		}
	})

	t.Run("UTC stamp", func(t *testing.T) {
		got, ok := ParseBackupStamp("20240131T235959Z")
		want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
		if !ok || !got.Equal(want) {
			t.Fatalf("got (%v, %v), want (%v, true)", got, ok, want)
		}
	})

	t.Run("UTC stamp with collision suffix", func(t *testing.T) {
		for _, name := range []string{"20240131T235959Z-1", "20240131T235959Z-1-1"} {
			got, ok := ParseBackupStamp(name)
			want := time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC)
			if !ok || !got.Equal(want) {
				t.Fatalf("ParseBackupStamp(%q) = (%v, %v), want (%v, true)", name, got, ok, want)
			}
		}
	})

	for _, name := range []string{"manual-copy", "", "20240131", "20241301-000000", "20240131T235959Z-x", "20240131T235959Z-", "20240131-235959-1"} {
		t.Run("invalid "+name, func(t *testing.T) {
			if _, ok := ParseBackupStamp(name); ok {
				t.Fatalf("ParseBackupStamp(%q) ok = true, want false", name)
			}
		})
	}
}
