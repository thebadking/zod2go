package zod2go

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type stringRules struct {
	format, pattern, alg, hostname, protocol string
	normalize                                bool
	uuid                                     *int
}

func patternMatches(pattern, value string) bool {
	const emailPrefix = `^(?!\.)(?!.*\.\.)`
	if strings.HasPrefix(pattern, emailPrefix) {
		if strings.HasPrefix(value, ".") || strings.Contains(value, "..") {
			return false
		}
		pattern = "^" + strings.TrimPrefix(pattern, emailPrefix)
	}
	re, err := regexp.Compile(pattern)
	return err == nil && re.MatchString(value)
}

func validateStringRules(value string, rules stringRules) bool {
	switch rules.format {
	case "url":
		_, valid := validateURL(value, rules)
		return valid
	case "base64":
		if len(value)%4 != 0 || strings.ContainsAny(value, "\t\n\r\v\f ") {
			return false
		}
		_, err := base64.StdEncoding.DecodeString(value)
		return err == nil
	case "base64url":
		if !patternMatches(`^[A-Za-z0-9_-]*$`, value) {
			return false
		}
		_, err := base64.RawURLEncoding.DecodeString(value)
		return err == nil
	case "ipv6":
		return !strings.Contains(value, "%") && strings.Contains(value, ":") && net.ParseIP(value) != nil
	case "cidrv6":
		pieces := strings.Split(value, "/")
		if len(pieces) != 2 || !validateStringRules(pieces[0], stringRules{format: "ipv6"}) {
			return false
		}
		number := jsNumber(pieces[1])
		return !math.IsNaN(number) && number >= 0 && number <= 128 && jsNumberString(number) == pieces[1]
	case "jwt":
		pieces := strings.Split(value, ".")
		if len(pieces) != 3 || pieces[0] == "" {
			return false
		}
		header, err := decodeAtob(pieces[0])
		if err != nil {
			return false
		}
		var obj map[string]any
		if json.Unmarshal(header, &obj) != nil || obj == nil {
			return false
		}
		if typ, present := obj["typ"]; present && typ != "JWT" {
			return false
		}
		if !jsTruthy(obj["alg"]) {
			return false
		}
		return rules.alg == "" || obj["alg"] == rules.alg
	}
	if rules.format == "hostname" {
		plain := strings.TrimSuffix(value, ".")
		if len(plain) < 1 || len(plain) > 253 {
			return false
		}
		for _, label := range strings.Split(plain, ".") {
			if !patternMatches(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`, label) {
				return false
			}
		}
		return true
	}
	pattern := rules.pattern
	if pattern == "" {
		if rules.format == "" {
			return true
		}
		if rules.format == "uuid" && rules.uuid != nil && *rules.uuid > 0 {
			re, _ := formatRegex("uuid", rules.uuid)
			return re.MatchString(value)
		}
		pattern = builtinFormatPatterns[rules.format]
		if pattern == "" {
			return false
		}
	}
	if rules.format == "emoji" && strings.Contains(pattern, `\p{Extended_Pictographic}`) {
		return validEmoji(value)
	}
	if rules.format == "duration" && strings.Contains(pattern, `(?!.*W)`) {
		return validDuration(value)
	}
	return patternMatches(pattern, value)
}

func jsTruthy(value any) bool {
	switch x := value.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	default:
		if n, ok := toFloat64(value); ok {
			return n != 0
		}
		return true
	}
}

var durationPattern = regexp.MustCompile(`^P(?:[0-9]+W|(?:[0-9]+Y)?(?:[0-9]+M)?(?:[0-9]+D)?(?:T(?:[0-9]+H)?(?:[0-9]+M)?(?:[0-9]+(?:[.,][0-9]+)?S)?)?)$`)

func validDuration(value string) bool {
	if value == "P" || strings.HasSuffix(value, "T") {
		return false
	}
	return durationPattern.MatchString(value)
}

func validateURL(value string, rules stringRules) (string, bool) {
	trimmed := jsTrim(value)
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" {
		return "", false
	}
	scheme := strings.ToLower(parsed.Scheme)
	special := scheme == "http" || scheme == "https" || scheme == "ftp" || scheme == "ws" || scheme == "wss"
	if special && parsed.Host == "" {
		// WHATWG special schemes accept a missing authority separator unless the
		// explicit HTTP URL constraint disallows it.
		if !rules.normalize && rules.protocol == `^https?$` {
			return "", false
		}
		rest := strings.TrimLeft(strings.TrimPrefix(trimmed, parsed.Scheme+":"), "/\\")
		parsed, err = url.Parse(scheme + "://" + rest)
		if err != nil {
			return "", false
		}
	}
	if special && parsed.Host == "" {
		return "", false
	}
	if strings.ContainsAny(parsed.Host, " \t\r\n") {
		return "", false
	}
	hostname := strings.ToLower(parsed.Hostname())
	if rules.hostname != "" && !patternMatches(rules.hostname, hostname) {
		return "", false
	}
	if rules.protocol != "" && !patternMatches(rules.protocol, scheme) {
		return "", false
	}
	if port := parsed.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 0 || n > 65535 {
			return "", false
		}
	}
	if !rules.normalize {
		return trimmed, true
	}
	parsed.Scheme = scheme
	if parsed.Host != "" {
		port := parsed.Port()
		if (scheme == "http" && port == "80") || (scheme == "https" && port == "443") || (scheme == "ftp" && port == "21") {
			port = ""
		}
		host := hostname
		if strings.Contains(host, ":") {
			host = "[" + host + "]"
		}
		if port != "" {
			host += ":" + port
		}
		parsed.Host = host
	}
	if special {
		if parsed.Path == "" {
			parsed.Path = "/"
		} else {
			trailing := strings.HasSuffix(parsed.Path, "/")
			parsed.Path = path.Clean(parsed.Path)
			if trailing && !strings.HasSuffix(parsed.Path, "/") {
				parsed.Path += "/"
			}
		}
	}
	return parsed.String(), true
}

func decodeAtob(value string) ([]byte, error) {
	compact := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			return -1
		}
		return r
	}, value)
	var decoded []byte
	var err error
	if strings.Contains(compact, "=") {
		decoded, err = base64.StdEncoding.DecodeString(compact)
	} else {
		decoded, err = base64.RawStdEncoding.DecodeString(compact)
	}
	if err != nil {
		return nil, err
	}
	var text strings.Builder
	for _, b := range decoded {
		text.WriteRune(rune(b))
	}
	return []byte(text.String()), nil
}
