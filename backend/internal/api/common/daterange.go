// Package common holds small HTTP-layer helpers shared across the api domain
// packages (request scheme/host derivation, date-range parsing, ...). They are
// pure presentation-layer utilities with no dependency on the handlers core.
package common

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ParseDateRange resolves the [from, to] window from a request's query string,
// defaulting to the last 30 days.
func ParseDateRange(c *gin.Context) (time.Time, time.Time) {
	return ParseDateRangeWithValues(c.Query("from"), c.Query("to"))
}

// ParseDateRangeWithValues resolves the [from, to] window from raw date strings
// (YYYY-MM-DD), defaulting to the last 30 days when values are missing/invalid.
func ParseDateRangeWithValues(fromRaw, toRaw string) (time.Time, time.Time) {
	to := time.Now()
	from := to.AddDate(0, 0, -30)
	if v := strings.TrimSpace(fromRaw); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := strings.TrimSpace(toRaw); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t
		}
	}
	return from, to
}
