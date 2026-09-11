// Package jackson reproduces the parts of Jackson databind that decide what a
// JSON request body binds to, where Go's encoding/json makes different choices.
//
// Three of those choices are observable through this application's API:
//
//	{"name": 5}          Jackson coerces a JSON scalar to a String field and
//	                     keeps the token's own text, so the value becomes "5"
//	                     and echoes back as "5". encoding/json rejects it.
//	{...} trailing junk  Jackson reads one value and stops;
//	                     DeserializationFeature.FAIL_ON_TRAILING_TOKENS is off
//	                     by default. encoding/json rejects the trailing bytes.
//	{"a":1,"a":2}        Both keep the last occurrence, so nothing to reproduce.
//
// The package binds only what this application declares: structs of nullable
// string properties named by their `json` tags.
package jackson

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
)

// ErrMissingBody is what Spring reports as "Required request body is missing":
// the payload parsed cleanly and was JSON null.
var ErrMissingBody = errors.New("required request body is missing")

// MismatchedInputError is Jackson's MismatchedInputException: a token that
// cannot be coerced into the target property's type.
type MismatchedInputError struct {
	Property string
	Token    string
}

func (e *MismatchedInputError) Error() string {
	return "cannot deserialize value of type `java.lang.String` from " + e.Token +
		" (property \"" + e.Property + "\")"
}

// Unmarshal binds a JSON payload onto target, which must be a pointer to a
// struct of *string fields.
func Unmarshal(payload []byte, target any) error {
	value, err := firstValue(payload)
	if err != nil {
		return err
	}
	if isNull(value) {
		return ErrMissingBody
	}
	var properties map[string]json.RawMessage
	if err := json.Unmarshal(value, &properties); err != nil {
		return err
	}
	return bind(properties, target)
}

// firstValue decodes exactly one JSON value and discards whatever follows it,
// which is what Jackson's parser does when FAIL_ON_TRAILING_TOKENS is off.
func firstValue(payload []byte) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

// bind writes each recognised property onto the target's matching field,
// leaving unknown properties alone: Spring Boot disables
// FAIL_ON_UNKNOWN_PROPERTIES, so an unexpected key is simply ignored.
func bind(properties map[string]json.RawMessage, target any) error {
	structValue := reflect.ValueOf(target)
	if structValue.Kind() != reflect.Pointer || structValue.IsNil() {
		return errors.New("jackson: target must be a non-nil pointer to a struct")
	}
	structValue = structValue.Elem()
	if structValue.Kind() != reflect.Struct {
		return errors.New("jackson: target must be a pointer to a struct")
	}
	structType := structValue.Type()
	for i := 0; i < structValue.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		name := jsonName(field)
		raw, present := properties[name]
		if !present {
			continue
		}
		text, err := coerceToString(name, raw)
		if err != nil {
			return err
		}
		structValue.Field(i).Set(reflect.ValueOf(text))
	}
	return nil
}

// coerceToString applies Jackson's StringDeserializer: a JSON string binds
// directly, JSON null binds null, and any other scalar binds the token's own
// text — which is why 0.30 arrives as "0.30" and not as "0.3". Structured
// tokens have no String representation and are an error.
func coerceToString(property string, raw json.RawMessage) (*string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, &MismatchedInputError{Property: property, Token: "an empty token"}
	}
	switch trimmed[0] {
	case 'n':
		return nil, nil
	case '"':
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return nil, err
		}
		return &text, nil
	case '{':
		return nil, &MismatchedInputError{Property: property, Token: "Object value"}
	case '[':
		return nil, &MismatchedInputError{Property: property, Token: "Array value"}
	default:
		// A number or a boolean: keep the source text verbatim, which is what
		// JsonParser.getText() returns for a scalar token.
		text := string(trimmed)
		return &text, nil
	}
}

func isNull(raw json.RawMessage) bool {
	return string(bytes.TrimSpace(raw)) == "null"
}

// jsonName is the property name Jackson binds a field by.
func jsonName(field reflect.StructField) string {
	tag, ok := field.Tag.Lookup("json")
	if !ok {
		return field.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return field.Name
	}
	return name
}
