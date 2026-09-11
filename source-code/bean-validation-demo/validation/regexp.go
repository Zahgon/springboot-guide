package validation

import "regexp"

// matchesPhoneNumber applies phoneNumberRegExp the way String.matches does:
// against the whole string.
//
// The expression is compiled once. Java recompiles it on every isValid call,
// which is a cost rather than a behaviour, so it is not reproduced.
var phoneNumberPattern = regexp.MustCompile(`\A(?:` + phoneNumberRegExp + `)\z`)

func matchesPhoneNumber(value string) bool { return phoneNumberPattern.MatchString(value) }
