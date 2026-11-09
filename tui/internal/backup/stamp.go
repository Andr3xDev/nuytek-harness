package backup

import (
	"regexp"
	"time"
)

const (
	localStampLayout = "20060102-150405"
	utcStampLayout   = "20060102T150405Z"
)

// claude.sh appends "-1" to a UTC stamp for each same-second collision.
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
