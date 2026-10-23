package zod2go

import (
	"fmt"
	"regexp"
)

// String format regexes aligned with Zod v4 where Go's RE2 engine allows.
// Go regexp lacks PCRE lookaheads, so email uses a practical subset.
var (
	reEmail  = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?(?:\.[A-Za-z0-9](?:[A-Za-z0-9\-]*[A-Za-z0-9])?)+$`)
	reUUID   = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$`)
	reUUID4  = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12})$`)
	reCUID2  = regexp.MustCompile(`^[0-9a-z]+$`)
	reULID   = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Za-hjkmnp-tv-z]{26}$`)
	reNanoID = regexp.MustCompile(`^[a-zA-Z0-9_-]{21}$`)
	reIPv4   = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])\.){3}(?:25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]|[0-9])$`)
	reURL    = regexp.MustCompile(`^https?://`)
)

func formatRegex(format string, uuidVer *int) (*regexp.Regexp, bool) {
	switch format {
	case "email":
		return reEmail, true
	case "uuid":
		if uuidVer != nil && *uuidVer >= 1 && *uuidVer <= 8 {
			return regexp.MustCompile(fmt.Sprintf(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-%d[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`, *uuidVer)), true
		}
		return reUUID, true
	case "cuid2":
		return reCUID2, true
	case "ulid":
		return reULID, true
	case "nanoid":
		return reNanoID, true
	case "ipv4":
		return reIPv4, true
	case "url":
		return reURL, true
	default:
		return nil, false
	}
}
