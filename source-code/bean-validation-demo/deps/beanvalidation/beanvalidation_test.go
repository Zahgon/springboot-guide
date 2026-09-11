package beanvalidation

import (
	"strings"
	"testing"
)

const (
	groupA Group = "AddPersonGroup"
	groupD Group = "DeletePersonGroup"
)

func ptr(value string) *string { return &value }

type bean struct {
	Text   *string `json:"text"`
	Number *int    `json:"number"`
}

func (bean) Constraints() Constraints { return nil }

type sized struct {
	Text *string `json:"text"`
}

func (sized) Constraints() Constraints {
	return Constraints{On("text", Size(0, 33), NotNull().WithMessage("text 不能为空"))}
}

type grouped struct {
	Group *string `json:"group"`
}

func (grouped) Constraints() Constraints {
	return Constraints{On("group",
		NotNull().WithGroups(groupD),
		Null().WithGroups(groupA),
	)}
}

func TestNotNullAndNullAreOppositesOnPointers(t *testing.T) {
	notNull, null := NotNull(), Null()
	for _, testCase := range []struct {
		name   string
		value  any
		isNull bool
	}{
		{"nil interface", nil, true},
		{"nil *string", (*string)(nil), true},
		{"empty string", ptr(""), false},
		{"non-empty string", ptr("x"), false},
		{"zero int", 0, false},
	} {
		if got := IsNull(testCase.value); got != testCase.isNull {
			t.Errorf("IsNull(%s) = %v, want %v", testCase.name, got, testCase.isNull)
		}
		if got := notNull.check(testCase.value); got == testCase.isNull {
			t.Errorf("@NotNull(%s) = %v, want %v", testCase.name, got, !testCase.isNull)
		}
		if got := null.check(testCase.value); got != testCase.isNull {
			t.Errorf("@Null(%s) = %v, want %v", testCase.name, got, testCase.isNull)
		}
	}
}

func TestSizePassesOnNullAndCountsUTF16Units(t *testing.T) {
	size := Size(0, 33)
	if !size.check(nil) {
		t.Error("@Size rejected null; Bean Validation leaves absence to @NotNull")
	}
	if !size.check(ptr("")) {
		t.Error("@Size(max=33) rejected the empty string")
	}
	if !size.check(ptr(strings.Repeat("0", 33))) {
		t.Error("@Size(max=33) rejected 33 characters")
	}
	if size.check(ptr(strings.Repeat("0", 34))) {
		t.Error("@Size(max=33) accepted 34 characters")
	}
	// String.length() counts UTF-16 code units, so a BMP character is one and an
	// emoji is two. Counting runes would let 33 emoji through; counting bytes
	// would reject 33 Chinese characters.
	if !size.check(ptr(strings.Repeat("一", 33))) {
		t.Error("@Size(max=33) rejected 33 CJK characters, which are 33 UTF-16 units")
	}
	if size.check(ptr(strings.Repeat("😀", 17))) {
		t.Error("@Size(max=33) accepted 17 emoji, which are 34 UTF-16 units")
	}
	if !size.check(ptr(strings.Repeat("😀", 16))) {
		t.Error("@Size(max=33) rejected 16 emoji, which are 32 UTF-16 units")
	}
}

func TestSizeDefaultMessageInterpolatesBothBounds(t *testing.T) {
	// @Size(max = 33) has an implicit min of 0, and the default message names
	// it. This exact string is the 400 body for an over-long name.
	if got, want := Size(0, 33).resolvedMessage(), "size must be between 0 and 33"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}

func TestMaxBoundsOnlyFromAboveAndPassesOnNull(t *testing.T) {
	max := Max(5)
	for _, value := range []int{-3, 0, 5} {
		if !max.check(value) {
			t.Errorf("@Max(5) rejected %d", value)
		}
	}
	if max.check(6) {
		t.Error("@Max(5) accepted 6")
	}
	if !max.check(nil) {
		t.Error("@Max rejected null")
	}
	if got, want := max.resolvedMessage(), "must be less than or equal to 5"; got != want {
		t.Errorf("message = %q, want %q", got, want)
	}
}

func TestPatternIsAnchoredAndPassesOnNull(t *testing.T) {
	pattern := Pattern(`(^Man$|^Woman$|^UGM$)`)
	for _, value := range []string{"Man", "Woman", "UGM"} {
		if !pattern.check(ptr(value)) {
			t.Errorf("@Pattern rejected %q", value)
		}
	}
	// The alternation is wrapped before anchoring, so a top-level `|` cannot
	// escape the anchors and match a prefix.
	for _, value := range []string{"Man22", "xMan", "Manx", "Man\n", "", "man"} {
		if pattern.check(ptr(value)) {
			t.Errorf("@Pattern accepted %q", value)
		}
	}
	if !pattern.check(nil) {
		t.Error("@Pattern rejected null; @NotNull is what rejects a missing value")
	}

	// Everything above would also hold if Pattern did no anchoring at all,
	// because this application's only @Pattern already carries its own ^ and $.
	// Hibernate applies the regex with Matcher.matches(), which anchors whether
	// or not the pattern says so, and that is what this constraint claims to
	// reproduce — so the claim is tested with a pattern that has no anchors of
	// its own. Without the wrapper, "12a" and "a12" would both match here.
	unanchored := Pattern(`[0-9]+`)
	if !unanchored.check(ptr("123")) {
		t.Error(`@Pattern("[0-9]+") rejected "123"`)
	}
	for _, value := range []string{"12a", "a12", "a12b", "1 2"} {
		if unanchored.check(ptr(value)) {
			t.Errorf(`@Pattern("[0-9]+") accepted %q; Matcher.matches() anchors the whole value`, value)
		}
	}
}

func TestStringConstraintDecidesItsOwnNullPolicy(t *testing.T) {
	acceptsNull := StringConstraint("PhoneNumber", "bad", func(v *string) bool { return v == nil })
	rejectsNull := StringConstraint("Region", "bad", func(v *string) bool { return v != nil })
	if !acceptsNull.check(nil) {
		t.Error("a validator that accepts null was not given null")
	}
	if rejectsNull.check(nil) {
		t.Error("a validator that rejects null was not given null")
	}
	if acceptsNull.check(ptr("x")) {
		t.Error("a present value was passed as null")
	}
}

func TestValidateReportsViolationsInDeclarationOrder(t *testing.T) {
	violations := Validate(&sized{Text: nil})
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	if violations[0].PropertyPath != "text" {
		t.Errorf("path = %q, want %q", violations[0].PropertyPath, "text")
	}
	if violations[0].Message != "text 不能为空" {
		t.Errorf("message = %q", violations[0].Message)
	}
	if got := Validate(&sized{Text: ptr("ok")}); len(got) != 0 {
		t.Errorf("a valid bean produced %v", got)
	}
}

func TestValidateSelectsConstraintsByGroup(t *testing.T) {
	withGroup := &grouped{Group: ptr("group1")}
	empty := &grouped{}

	// Neither constraint is in Default, so the default group finds nothing.
	if got := Validate(withGroup); len(got) != 0 {
		t.Errorf("default group found %v, want nothing", got)
	}
	if got := Validate(empty); len(got) != 0 {
		t.Errorf("default group found %v, want nothing", got)
	}
	if got := Validate(withGroup, groupA); len(got) != 1 {
		t.Errorf("AddPersonGroup on a non-null group found %v, want one @Null violation", got)
	}
	if got := Validate(empty, groupA); len(got) != 0 {
		t.Errorf("AddPersonGroup on a null group found %v, want nothing", got)
	}
	if got := Validate(empty, groupD); len(got) != 1 {
		t.Errorf("DeletePersonGroup on a null group found %v, want one @NotNull violation", got)
	}
	if got := Validate(withGroup, groupD); len(got) != 0 {
		t.Errorf("DeletePersonGroup on a non-null group found %v, want nothing", got)
	}
}

func TestValidateIgnoresNonBeans(t *testing.T) {
	if got := Validate(nil); got != nil {
		t.Errorf("Validate(nil) = %v", got)
	}
	if got := Validate(42); got != nil {
		t.Errorf("Validate(int) = %v", got)
	}
	if got := Validate((*sized)(nil)); got != nil {
		t.Errorf("Validate(nil bean) = %v", got)
	}
	if got := Validate(&bean{}); got != nil {
		t.Errorf("Validate(bean with no constraints) = %v", got)
	}
}

func TestValidateParametersPathsAndCascade(t *testing.T) {
	// A constraint on the parameter itself: `method.parameter`.
	err := ValidateParameters("getPersonByID",
		[]Parameter{Param("id", 6, Max(5).WithMessage("超过 id 的范围了"))})
	if err == nil {
		t.Fatal("ValidateParameters accepted 6 against @Max(5)")
	}
	if got, want := err.Error(), "getPersonByID.id: 超过 id 的范围了"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// @Valid cascades into the bean: `method.parameter.property`.
	err = ValidateParameters("validatePersonRequest",
		[]Parameter{Param("personRequest", &sized{}, Valid())})
	violation, ok := err.(*ViolationError)
	if !ok {
		t.Fatalf("err = %v, want a *ViolationError", err)
	}
	if got, want := violation.Violations[0].PropertyPath, "validatePersonRequest.personRequest.text"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}

	if err := ValidateParameters("m", []Parameter{Param("id", 5, Max(5))}); err != nil {
		t.Errorf("ValidateParameters rejected a valid call: %v", err)
	}
}

func TestValidateParametersPassesGroupsThroughTheCascade(t *testing.T) {
	if err := ValidateParameters("validatePersonGroupForAdd",
		[]Parameter{Param("person", &grouped{Group: ptr("group1")}, Valid())},
		groupA); err == nil {
		t.Error("AddPersonGroup did not reach the cascaded bean")
	}
	if err := ValidateParameters("validatePersonGroupForAdd",
		[]Parameter{Param("person", &grouped{Group: ptr("group1")}, Valid())},
		groupD); err != nil {
		t.Errorf("DeletePersonGroup found a violation it should not have: %v", err)
	}
}

func TestViolationErrorJoinsWithCommaSpace(t *testing.T) {
	// ConstraintViolationException renders every violation as `path: message`
	// and joins them with ", ". The controller advice returns this verbatim.
	err := &ViolationError{Violations: []Violation{
		{PropertyPath: "a.b", Message: "first"},
		{PropertyPath: "a.c", Message: "second"},
	}}
	if got, want := err.Error(), "a.b: first, a.c: second"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestConstraintBuildersReturnCopies(t *testing.T) {
	base := NotNull()
	custom := base.WithMessage("custom").WithGroups(groupA)
	if base.resolvedMessage() != "must not be null" {
		t.Error("WithMessage mutated the constraint it was called on")
	}
	if !base.appliesTo(nil) {
		t.Error("WithGroups mutated the constraint it was called on")
	}
	if custom.appliesTo(nil) {
		t.Error("a group-scoped constraint still ran under the default group")
	}
	if !custom.appliesTo([]Group{groupA}) {
		t.Error("a group-scoped constraint did not run under its own group")
	}
	if got, want := custom.Kind(), "NotNull"; got != want {
		t.Errorf("Kind() = %q, want %q", got, want)
	}
	if got := custom.String(); got != `@NotNull(message="custom")` {
		t.Errorf("String() = %s", got)
	}
	if got := base.String(); got != "@NotNull" {
		t.Errorf("String() = %s", got)
	}
}

func TestStringValueDereferences(t *testing.T) {
	if _, ok := StringValue(nil); ok {
		t.Error("StringValue(nil) reported a value")
	}
	if _, ok := StringValue(42); ok {
		t.Error("StringValue(int) reported a value")
	}
	got, ok := StringValue(ptr("x"))
	if !ok || got != "x" {
		t.Errorf("StringValue = %q, %v", got, ok)
	}
}
