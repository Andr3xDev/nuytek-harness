package backup

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	localStampLayout = "20060102-150405"
	utcStampLayout   = "20060102T150405Z"
)

// A same-second collision appends "-N" to a stamp, repeatedly if needed; no suffix also matches.
var collisionSuffix = regexp.MustCompile(`^(-\d+)*$`)

// StoreKind selects the stamp format of a backup store; StoreHome is the zero value.
type StoreKind int

const (
	StoreHome StoreKind = iota
	StoreRepo
)

// NewStamp appends -1, -2, ... until exists reports the name free.
func NewStamp(kind StoreKind, now time.Time, exists func(string) bool) string {
	base := now.Local().Format(localStampLayout)
	if kind == StoreRepo {
		base = now.UTC().Format(utcStampLayout)
	}
	name := base
	for n := 1; exists(name); n++ {
		name = base + "-" + strconv.Itoa(n)
	}
	return name
}

// ValidateStamp accepts only a single path element that is not "." or "..".
func ValidateStamp(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("invalid backup stamp %q", name)
	}
	return nil
}

func ParseStamp(name string) (time.Time, bool) {
	for _, l := range []struct {
		layout string
		loc    *time.Location
	}{{localStampLayout, time.Local}, {utcStampLayout, time.UTC}} {
		// Both layouts format to their own length, so the stamp is a fixed-width prefix.
		if len(name) < len(l.layout) || !collisionSuffix.MatchString(name[len(l.layout):]) {
			continue
		}
		if t, err := time.ParseInLocation(l.layout, name[:len(l.layout)], l.loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
