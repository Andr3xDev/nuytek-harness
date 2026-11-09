package backup

import (
	"regexp"
	"time"
)

const (
	localStampLayout = "20060102-150405"
	utcStampLayout   = "20060102T150405Z"
)

// The bash backup flow appends "-N" to a UTC stamp on a same-second collision.
var collisionSuffix = regexp.MustCompile(`(-\d+)+$`)

func ParseStamp(name string) (time.Time, bool) {
	if t, err := time.ParseInLocation(localStampLayout, name, time.Local); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(utcStampLayout, collisionSuffix.ReplaceAllString(name, ""), time.UTC); err == nil {
		return t, true
	}
	return time.Time{}, false
}
