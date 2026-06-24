package common

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTimeFlexible parses a timestamp as RFC3339, falling back to a YYYY-MM-DD date.
func ParseTimeFlexible(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", value)
}

// ParseInt parses a base-10 integer.
func ParseInt(value string) (int, error) {
	return strconv.Atoi(value)
}

// ToString renders an arbitrary value as a trimmed string.
func ToString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
