package handlers

import (
	"strconv"
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
