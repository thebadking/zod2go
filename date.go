package zod2go

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func clipDate(value float64) (time.Time, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) > 8640000000000000 {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(math.Trunc(value))).UTC(), true
}
func coerceDate(input any) (time.Time, bool) {
	switch value := input.(type) {
	case time.Time:
		return clipDate(float64(value.UnixMilli()))
	case *time.Time:
		if value != nil {
			return clipDate(float64(value.UnixMilli()))
		}
	case string:
		return parseDateString(value)
	case []any:
		return parseDateString(jsString(value))
	case nil:
		return clipDate(0)
	case bool:
		if value {
			return clipDate(1)
		}
		return clipDate(0)
	default:
		if number, ok := toFloat64(input); ok {
			return clipDate(number)
		}
	}
	return time.Time{}, false
}

var isoDateOnly = regexp.MustCompile(`^(\d{4})(?:-(\d{2})(?:-(\d{2}))?)?$`)

func parseDateString(input string) (time.Time, bool) {
	value := jsTrim(input)
	if match := isoDateOnly.FindStringSubmatch(value); match != nil {
		year, _ := strconv.Atoi(match[1])
		month, day := 1, 1
		if match[2] != "" {
			month, _ = strconv.Atoi(match[2])
		}
		if match[3] != "" {
			day, _ = strconv.Atoi(match[3])
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			return time.Time{}, false
		}
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), true
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC1123, time.RFC1123Z, time.RFC850, time.ANSIC, time.UnixDate, time.RubyDate, "Mon Jan 02 2006 15:04:05 GMT-0700", "Mon Jan 02 2006", "Jan 2, 2006", "January 2, 2006", "2006-1-2", "1/2/2006", "2006/1/2", "2006-01-02T15:04:05.999999999", "2006-01-02T15:04"} {
		candidate := value
		if at := strings.Index(candidate, " ("); at >= 0 {
			candidate = candidate[:at]
		}
		parsed, err := time.ParseInLocation(layout, candidate, time.Local)
		if err == nil {
			return clipDate(float64(parsed.UnixMilli()))
		}
	}
	if number, err := strconv.Atoi(value); err == nil && number >= 0 {
		year, month := number, 1
		if len(value) < 3 {
			if number == 0 {
				year = 2000
			} else if number <= 12 {
				year = 2001
				month = number
			} else if number < 32 {
				return time.Time{}, false
			} else if number < 50 {
				year = 2000 + number
			} else {
				year = 1900 + number
			}
		}
		return clipDate(float64(time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local).UnixMilli()))
	}
	return time.Time{}, false
}

func jsWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}
func jsTrim(value string) string { return strings.TrimFunc(value, jsWhitespace) }
