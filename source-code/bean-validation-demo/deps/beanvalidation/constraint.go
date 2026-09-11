package beanvalidation

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Constraint is one annotation's worth of validation: a predicate, the message
// it reports when the predicate fails, and the groups it belongs to.
//
// Constraint is a value; the With* methods return a modified copy, so a
// declaration reads like the annotation it replaces and cannot be mutated by a
// later one.
type Constraint struct {
	kind            string
	message         string
	defaultTemplate string
	groups          []Group
	predicate       func(value any) bool
}

// WithMessage overrides the constraint's message, the equivalent of the
// annotation's `message` attribute.
func (c Constraint) WithMessage(message string) Constraint {
	c.message = message
	return c
}

// WithGroups restricts the constraint to the named groups, the equivalent of
// the annotation's `groups` attribute. A constraint that names groups no longer
// belongs to Default.
func (c Constraint) WithGroups(groups ...Group) Constraint {
	c.groups = append([]Group(nil), groups...)
	return c
}

// Kind returns the constraint's annotation name, for diagnostics.
func (c Constraint) Kind() string { return c.kind }

func (c Constraint) resolvedMessage() string {
	if c.message != "" {
		return c.message
	}
	return c.defaultTemplate
}

func (c Constraint) check(value any) bool { return c.predicate(value) }

// appliesTo reports whether the constraint runs for the requested groups. An
// empty request means Default, and a constraint that names no groups is in
// Default, which together give the usual "unqualified constraint, unqualified
// validation" case.
func (c Constraint) appliesTo(requested []Group) bool {
	declared := c.groups
	if len(declared) == 0 {
		declared = []Group{Default}
	}
	if len(requested) == 0 {
		requested = []Group{Default}
	}
	for _, want := range requested {
		for _, have := range declared {
			if want == have {
				return true
			}
		}
	}
	return false
}

// IsNull reports whether value is the Java null: a nil interface, or a nil
// pointer, map, slice or interface behind one.
//
// The distinction matters throughout this application. A JSON body that omits
// `name` yields null and trips @NotNull; a body with `"name": ""` yields a
// present empty string that passes both @NotNull and @Size.
func IsNull(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		return reflected.IsNil()
	default:
		return false
	}
}

// StringValue dereferences value to a string, reporting false when it is null
// or not a string.
func StringValue(value any) (string, bool) {
	if IsNull(value) {
		return "", false
	}
	reflected := reflect.ValueOf(value)
	for reflected.Kind() == reflect.Pointer {
		reflected = reflected.Elem()
	}
	if reflected.Kind() != reflect.String {
		return "", false
	}
	return reflected.String(), true
}

// intValue dereferences value to an int64, reporting false when it is null or
// not an integer.
func intValue(value any) (int64, bool) {
	if IsNull(value) {
		return 0, false
	}
	reflected := reflect.ValueOf(value)
	for reflected.Kind() == reflect.Pointer {
		reflected = reflected.Elem()
	}
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflected.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(reflected.Uint()), true
	default:
		return 0, false
	}
}

// NotNull is javax.validation.constraints.NotNull.
func NotNull() Constraint {
	return Constraint{
		kind:            "NotNull",
		defaultTemplate: "must not be null",
		predicate:       func(value any) bool { return !IsNull(value) },
	}
}

// Null is javax.validation.constraints.Null.
func Null() Constraint {
	return Constraint{
		kind:            "Null",
		defaultTemplate: "must be null",
		predicate:       IsNull,
	}
}

// Size is javax.validation.constraints.Size over a CharSequence.
//
// Two details of the Java constraint are load-bearing here. It passes on null,
// leaving absence to @NotNull. And its default message interpolates both
// bounds, so @Size(max = 33) — whose min defaults to 0 — reports
// "size must be between 0 and 33" even though no minimum was written down.
func Size(min, max int) Constraint {
	return Constraint{
		kind: "Size",
		defaultTemplate: "size must be between " +
			strconv.Itoa(min) + " and " + strconv.Itoa(max),
		predicate: func(value any) bool {
			text, ok := StringValue(value)
			if !ok {
				return true
			}
			length := charSequenceLength(text)
			return length >= min && length <= max
		},
	}
}

// charSequenceLength returns the length Java's CharSequence.length() reports:
// a count of UTF-16 code units, not of runes or of bytes.
func charSequenceLength(text string) int {
	return len(utf16.Encode([]rune(text)))
}

// Max is javax.validation.constraints.Max. It bounds only from above, and
// passes on null.
func Max(limit int64) Constraint {
	return Constraint{
		kind:            "Max",
		defaultTemplate: "must be less than or equal to " + strconv.FormatInt(limit, 10),
		predicate: func(value any) bool {
			number, ok := intValue(value)
			if !ok {
				return true
			}
			return number <= limit
		},
	}
}

// Pattern is javax.validation.constraints.Pattern.
//
// It passes on null, and it matches the whole value: Hibernate applies the
// regex with Matcher.matches(), which is anchored at both ends regardless of
// what the pattern itself says.
func Pattern(expression string) Constraint {
	compiled := regexp.MustCompile(anchored(expression))
	return Constraint{
		kind:            "Pattern",
		defaultTemplate: `must match "` + expression + `"`,
		predicate: func(value any) bool {
			text, ok := StringValue(value)
			if !ok {
				return true
			}
			return compiled.MatchString(text)
		},
	}
}

// anchored wraps an expression so that it must match the entire input, which is
// what java.util.regex.Matcher.matches() requires. The group is non-capturing
// and wraps the whole alternation, so a top-level `|` cannot escape the anchors.
func anchored(expression string) string {
	return `\A(?:` + expression + `)\z`
}

// StringConstraint builds a custom constraint over a nullable string, the
// equivalent of a @Constraint annotation with a ConstraintValidator<A, String>.
//
// The validator receives the raw nullable value rather than a dereferenced
// string, because whether a custom validator accepts null is its own decision:
// PhoneNumberValidator returns true for null and RegionValidator returns false.
func StringConstraint(kind string, defaultMessage string, isValid func(value *string) bool) Constraint {
	return Constraint{
		kind:            kind,
		defaultTemplate: defaultMessage,
		predicate: func(value any) bool {
			text, ok := StringValue(value)
			if !ok {
				return isValid(nil)
			}
			return isValid(&text)
		},
	}
}

// Valid marks a parameter for cascaded validation, the equivalent of @Valid on
// a method parameter. It never fails on its own; ValidateParameters recognises
// it and validates the parameter's own constraints instead.
func Valid() Constraint {
	return Constraint{
		kind:      cascadeKind,
		predicate: func(any) bool { return true },
	}
}

const cascadeKind = "Valid"

func (c Constraint) isCascade() bool { return c.kind == cascadeKind }

// String renders the constraint for diagnostics.
func (c Constraint) String() string {
	if c.message == "" {
		return "@" + c.kind
	}
	return "@" + c.kind + `(message="` + strings.ReplaceAll(c.message, `"`, `\"`) + `")`
}
