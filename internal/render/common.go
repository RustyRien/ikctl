package render

import (
	"strconv"
	"time"
)

func ToAge(ts time.Time, now time.Time) string {
	if ts.IsZero() {
		return "-"
	}

	d := now.Sub(ts)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "m")
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "h")
	case d < 30*24*time.Hour:
		return plural(int(d.Hours()/24), "d")
	default:
		return plural(int(d.Hours()/(24*30)), "mo")
	}
}

func plural(value int, unit string) string {
	if value <= 0 {
		return "now"
	}
	return strconv.Itoa(value) + unit
}

func normalizeCell(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
