package jackson

import (
	"errors"
	"testing"
)

// Every expectation here was read off the running Spring Boot application:
// each payload was POSTed to /api/persons and the echoed body recorded.

type payload struct {
	ClassID *string `json:"classId"`
	Name    *string `json:"name"`
}

func bindName(t *testing.T, body string) (*string, error) {
	t.Helper()
	var target payload
	err := Unmarshal([]byte(body), &target)
	return target.Name, err
}

func TestUnmarshalCoercesScalarsToStringKeepingTokenText(t *testing.T) {
	// Jackson's StringDeserializer takes JsonParser.getText() for a scalar,
	// which is the token's own source text — so trailing zeros and the
	// exponent's spelling survive, where a parse-then-format round trip would
	// lose both.
	for _, testCase := range []struct{ body, want string }{
		{`{"name":5}`, "5"},
		{`{"name":-0}`, "-0"},
		{`{"name":1.5}`, "1.5"},
		{`{"name":5.0}`, "5.0"},
		{`{"name":0.30}`, "0.30"},
		{`{"name":1e3}`, "1e3"},
		{`{"name":1E3}`, "1E3"},
		{`{"name":1.0e400}`, "1.0e400"},
		{`{"name":123456789012345678901234567890}`, "123456789012345678901234567890"},
		{`{"name":true}`, "true"},
		{`{"name":false}`, "false"},
		{`{"name":"n"}`, "n"},
	} {
		got, err := bindName(t, testCase.body)
		if err != nil {
			t.Errorf("Unmarshal(%s): %v", testCase.body, err)
			continue
		}
		if got == nil {
			t.Errorf("Unmarshal(%s) left name null, want %q", testCase.body, testCase.want)
			continue
		}
		if *got != testCase.want {
			t.Errorf("Unmarshal(%s) bound %q, want %q", testCase.body, *got, testCase.want)
		}
	}
}

func TestUnmarshalRejectsStructuredTokensForAStringProperty(t *testing.T) {
	for _, body := range []string{`{"name":{}}`, `{"name":[]}`, `{"name":["x"]}`} {
		if _, err := bindName(t, body); err == nil {
			t.Errorf("Unmarshal(%s) succeeded, want a MismatchedInputException", body)
		}
	}
}

func TestUnmarshalBindsNullAndAbsenceAlike(t *testing.T) {
	got, err := bindName(t, `{"name":null}`)
	if err != nil || got != nil {
		t.Errorf("explicit null bound %v (err %v), want null", got, err)
	}
	got, err = bindName(t, `{"classId":"1"}`)
	if err != nil || got != nil {
		t.Errorf("absent property bound %v (err %v), want null", got, err)
	}
}

func TestUnmarshalDistinguishesNullFromEmptyString(t *testing.T) {
	got, err := bindName(t, `{"name":""}`)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got == nil {
		t.Fatal("empty string bound as null; @NotNull would then reject a value the original accepts")
	}
	if *got != "" {
		t.Errorf("bound %q, want the empty string", *got)
	}
}

func TestUnmarshalIgnoresTrailingTokens(t *testing.T) {
	// FAIL_ON_TRAILING_TOKENS is off by default, so Jackson reads one value and
	// stops. encoding/json would reject every one of these.
	for _, body := range []string{
		`{"name":"n"} junk`,
		`{"name":"n"}{`,
		`{"name":"n"}]`,
		`{"name":"n"} {}`,
	} {
		got, err := bindName(t, body)
		if err != nil {
			t.Errorf("Unmarshal(%s): %v", body, err)
			continue
		}
		if got == nil || *got != "n" {
			t.Errorf("Unmarshal(%s) bound %v, want \"n\"", body, got)
		}
	}
}

func TestUnmarshalIgnoresUnknownProperties(t *testing.T) {
	got, err := bindName(t, `{"name":"n","zzz":1,"nested":{"a":[1,2]}}`)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got == nil || *got != "n" {
		t.Errorf("bound %v, want \"n\"", got)
	}
}

func TestUnmarshalKeepsTheLastOfDuplicateProperties(t *testing.T) {
	got, err := bindName(t, `{"name":"a","name":"bb"}`)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got == nil || *got != "bb" {
		t.Errorf("bound %v, want \"bb\"", got)
	}
}

func TestUnmarshalReportsAJSONNullBodyAsMissing(t *testing.T) {
	var target payload
	err := Unmarshal([]byte("null"), &target)
	if !errors.Is(err, ErrMissingBody) {
		t.Errorf("Unmarshal(null) = %v, want ErrMissingBody", err)
	}
}

func TestUnmarshalRejectsNonObjectPayloads(t *testing.T) {
	for _, body := range []string{`[]`, `"x"`, `{`, ``, `   `} {
		var target payload
		if err := Unmarshal([]byte(body), &target); err == nil {
			t.Errorf("Unmarshal(%q) succeeded, want an error", body)
		}
	}
}

func TestMismatchedInputErrorNamesTheProperty(t *testing.T) {
	var target payload
	err := Unmarshal([]byte(`{"name":{}}`), &target)
	var mismatch *MismatchedInputError
	if !errors.As(err, &mismatch) {
		t.Fatalf("Unmarshal = %v, want a *MismatchedInputError", err)
	}
	if mismatch.Property != "name" {
		t.Errorf("Property = %q, want %q", mismatch.Property, "name")
	}
	if mismatch.Error() == "" {
		t.Error("Error() is empty")
	}
}

func TestUnmarshalRejectsANonStructTarget(t *testing.T) {
	var notAStruct int
	if err := Unmarshal([]byte(`{}`), &notAStruct); err == nil {
		t.Error("Unmarshal into *int succeeded, want an error")
	}
	if err := Unmarshal([]byte(`{}`), payload{}); err == nil {
		t.Error("Unmarshal into a non-pointer succeeded, want an error")
	}
}
