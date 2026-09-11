package mvc

import "testing"

// Every case below was checked against the running Spring Boot application by
// requesting /api/persons/<value> and reading which of the three possible
// answers came back: the echoed number, the @Max message, or the framework's
// 400 type-mismatch page.
func TestParseIntegerFollowsNumberUtils(t *testing.T) {
	for _, testCase := range []struct {
		input string
		want  int
		ok    bool
	}{
		{"5", 5, true},
		{"05", 5, true},
		{"+5", 5, true},
		{"+05", 5, true},
		{"-3", -3, true},
		// Integer.valueOf reads plain decimal: "010" is ten, not eight. Only the
		// hex branch reaches Integer.decode, so there is no octal literal here.
		{"010", 10, true},
		{"08", 8, true},
		{"007", 7, true},
		// isHexNumber routes 0x, 0X and # to Integer.decode, sign included.
		{"0x5", 5, true},
		{"0X5", 5, true},
		{"#5", 5, true},
		{"-0x5", -5, true},
		{"+0x5", 5, true},
		{"0xff", 255, true},
		// trimAllWhitespace deletes whitespace wherever it occurs, so "1 2" is
		// twelve — not a parse failure, and not one.
		{" 5 ", 5, true},
		{"\t5", 5, true},
		{"\n5", 5, true},
		{"1 2", 12, true},
		{"2147483647", 2147483647, true},
		{"-2147483648", -2147483648, true},
		// Out of Integer range is a conversion failure, not a large number that
		// then fails @Max — which is why it yields the framework error page.
		{"2147483648", 0, false},
		{"-2147483649", 0, false},
		{"abc", 0, false},
		{"5.0", 0, false},
		{"0b101", 0, false},
		{"5_0", 0, false},
		{"", 0, false},
		{"   ", 0, false},
		{"0x", 0, false},
		{"#", 0, false},
	} {
		got, err := ParseInteger(testCase.input)
		if testCase.ok {
			if err != nil {
				t.Errorf("ParseInteger(%q) = error %v, want %d", testCase.input, err, testCase.want)
				continue
			}
			if got != testCase.want {
				t.Errorf("ParseInteger(%q) = %d, want %d", testCase.input, got, testCase.want)
			}
			continue
		}
		if err == nil {
			t.Errorf("ParseInteger(%q) = %d, want a conversion failure", testCase.input, got)
		}
	}
}
