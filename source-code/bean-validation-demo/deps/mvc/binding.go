package mvc

import (
	"errors"
	"io"
	"mime"
	"strings"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/jackson"
)

// The argument resolvers. In Spring these run before the controller method is
// entered, which is why a @Valid @RequestBody failure is reported against the
// method's *argument* and never reaches the method body.

// jsonMediaTypes is what MappingJackson2HttpMessageConverter says it can read,
// and what Spring advertises in the `Accept` header of a 415.
var jsonMediaTypes = []string{"application/json", "application/*+json"}

// unsupportedMediaTypeError is org.springframework.web.HttpMediaTypeNotSupportedException.
type unsupportedMediaTypeError struct {
	received  string
	supported []string
}

func (e *unsupportedMediaTypeError) Error() string {
	return "Content type '" + e.received + "' not supported"
}

// Reasons carried by httpMessageNotReadableError, quoted from Spring. The
// original appends the controller method's Java signature to the first and
// Jackson's parser location to the second; neither has a counterpart here, and
// neither is visible with the shipped error configuration.
const (
	missingBodyReason = "Required request body is missing"
	parseErrorReason  = "JSON parse error: "
)

// ValidJSONBody resolves a `@Valid @RequestBody T` argument: it checks the
// content type, deserialises the body, validates it, and only then calls the
// controller method.
//
// The zero value of T supplies the "absent property is null" behaviour: every
// nullable property is a pointer, so a key missing from the JSON stays nil and
// @NotNull sees the null that Jackson would have left.
func ValidJSONBody[T any](handler func(body *T) (ResponseEntity, error)) Handler {
	return func(request *Request) (ResponseEntity, error) {
		if err := requireJSONContentType(request); err != nil {
			return ResponseEntity{}, err
		}
		raw, err := io.ReadAll(request.Body)
		if err != nil {
			return ResponseEntity{}, &httpMessageNotReadableError{reason: "I/O error while reading request body"}
		}
		if len(strings.TrimSpace(string(raw))) == 0 {
			return ResponseEntity{}, &httpMessageNotReadableError{reason: missingBodyReason}
		}
		body := new(T)
		// Bound by the Jackson shim rather than encoding/json: the two disagree
		// about scalar-to-String coercion and about trailing tokens, and both
		// disagreements are visible in this endpoint's responses.
		if err := jackson.Unmarshal(raw, body); err != nil {
			if errors.Is(err, jackson.ErrMissingBody) {
				return ResponseEntity{}, &httpMessageNotReadableError{reason: missingBodyReason}
			}
			return ResponseEntity{}, &httpMessageNotReadableError{reason: parseErrorReason + err.Error()}
		}
		if violations := beanvalidation.Validate(body); len(violations) > 0 {
			return ResponseEntity{}, fieldErrorsFrom(violations)
		}
		return handler(body)
	}
}

// requireJSONContentType enforces the converter's consumable media types.
func requireJSONContentType(request *Request) error {
	header := request.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(header)
	if err != nil || !isJSONMediaType(mediaType) {
		return &unsupportedMediaTypeError{received: header, supported: jsonMediaTypes}
	}
	return nil
}

func isJSONMediaType(mediaType string) bool {
	return mediaType == "application/json" ||
		strings.HasPrefix(mediaType, "application/") && strings.HasSuffix(mediaType, "+json")
}

// fieldErrorsFrom turns bean violations into the binding result a
// MethodArgumentNotValidException carries.
func fieldErrorsFrom(violations []beanvalidation.Violation) *MethodArgumentNotValidError {
	fieldErrors := make([]FieldError, len(violations))
	for i, violation := range violations {
		fieldErrors[i] = FieldError{
			Field:          violation.PropertyPath,
			DefaultMessage: violation.Message,
		}
	}
	return &MethodArgumentNotValidError{FieldErrors: fieldErrors}
}

// IntPathVariable resolves an `@PathVariable("name") Integer` argument. A
// segment that will not convert never reaches validation — Spring reports the
// type mismatch first, which is why an out-of-range id produces the framework's
// error page rather than the @Max message.
func IntPathVariable(name string, handler func(value int) (ResponseEntity, error)) Handler {
	return func(request *Request) (ResponseEntity, error) {
		raw := request.PathVariable(name)
		value, err := ParseInteger(raw)
		if err != nil {
			return ResponseEntity{}, &typeMismatchError{value: raw}
		}
		return handler(value)
	}
}

// StringRequestParam resolves a `@RequestParam("name") String` argument.
//
// The parameter is required by default, so an absent one is an error while an
// empty one is a present empty string. A repeated parameter binds to every
// value joined with a comma — Spring converts the String[] the container
// supplies into the declared String — so `?name=ab&name=cd` is the six
// characters "ab,cd" and not "ab".
func StringRequestParam(name string, handler func(value string) (ResponseEntity, error)) Handler {
	return func(request *Request) (ResponseEntity, error) {
		values, present := request.URL.Query()[name]
		if !present || len(values) == 0 {
			return ResponseEntity{}, &missingRequestParamError{name: name}
		}
		return handler(strings.Join(values, ","))
	}
}

// AsViolationError reports whether err is a ConstraintViolationException and
// returns it, for an advice that declares that exception type.
func AsViolationError(err error) (*beanvalidation.ViolationError, bool) {
	var violation *beanvalidation.ViolationError
	if errors.As(err, &violation) {
		return violation, true
	}
	return nil, false
}

// AsMethodArgumentNotValidError reports whether err is a
// MethodArgumentNotValidException and returns it.
func AsMethodArgumentNotValidError(err error) (*MethodArgumentNotValidError, bool) {
	var invalid *MethodArgumentNotValidError
	if errors.As(err, &invalid) {
		return invalid, true
	}
	return nil, false
}
