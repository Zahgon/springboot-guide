package mvc

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// String-to-target conversion, as Spring's WebDataBinder performs it on a path
// variable or request parameter. It is not strconv.Atoi, and the difference is
// visible: `/api/persons/0x5` and `/api/persons/%205` both reach the handler
// with 5, while `/api/persons/2147483648` never reaches it at all.

// errNotAnInteger reports a value org.springframework.util.NumberUtils would
// have rejected.
var errNotAnInteger = errors.New("not a valid Integer")

// ParseInteger converts a request string to a Java Integer the way
// org.springframework.util.NumberUtils.parseNumber does:
//
//	StringUtils.trimAllWhitespace(text)             strip every whitespace
//	                                                character, not just the ends
//	isHexNumber(trimmed) ? Integer.decode(trimmed)  0x, 0X and # prefixes,
//	                     : Integer.valueOf(trimmed) after an optional sign
//
// Note what this does not do. Integer.valueOf reads plain decimal, so "010" is
// ten and not eight; only the hex branch reaches Integer.decode. And the result
// is a 32-bit int, so a value outside that range is a conversion failure rather
// than a large number that then fails @Max.
func ParseInteger(text string) (int, error) {
	trimmed := trimAllWhitespace(text)
	if trimmed == "" {
		return 0, errNotAnInteger
	}
	if isHexNumber(trimmed) {
		return decodeHex(trimmed)
	}
	return parseInt32(trimmed, 10)
}

// trimAllWhitespace is org.springframework.util.StringUtils.trimAllWhitespace:
// it deletes every whitespace character, wherever it occurs, so "1 2" becomes
// the number twelve.
func trimAllWhitespace(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
}

// isHexNumber is NumberUtils.isHexNumber: a "0x", "0X" or "#" prefix, allowed
// to follow a sign.
func isHexNumber(text string) bool {
	index := 0
	if text[0] == '-' || text[0] == '+' {
		index = 1
	}
	rest := text[index:]
	return strings.HasPrefix(rest, "0x") || strings.HasPrefix(rest, "0X") ||
		strings.HasPrefix(rest, "#")
}

// decodeHex is the hexadecimal branch of Integer.decode.
func decodeHex(text string) (int, error) {
	sign := ""
	if text[0] == '-' || text[0] == '+' {
		sign, text = string(text[0]), text[1:]
	}
	switch {
	case strings.HasPrefix(text, "0x"), strings.HasPrefix(text, "0X"):
		text = text[2:]
	case strings.HasPrefix(text, "#"):
		text = text[1:]
	}
	return parseInt32(sign+text, 16)
}

// parseInt32 parses a signed integer in the given radix and enforces Java's
// 32-bit Integer range. Go's ParseInt accepts underscores in some modes and a
// base-0 prefix in others; radix is passed explicitly here so it does neither.
func parseInt32(text string, radix int) (int, error) {
	value, err := strconv.ParseInt(text, radix, 64)
	if err != nil {
		return 0, errNotAnInteger
	}
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, errNotAnInteger
	}
	return int(value), nil
}
