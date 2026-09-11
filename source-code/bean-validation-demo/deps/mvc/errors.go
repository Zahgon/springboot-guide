package mvc

import (
	"net/http"
	"strconv"
	"time"
)

// The exceptions Spring MVC raises before a handler runs. They surface either
// through a @ControllerAdvice that declares them or, when none does, through
// Spring Boot's default error page.
//
// Each message below is the text the corresponding Spring exception carries,
// recorded from the original with server.error.include-message=always. The
// shipped configuration leaves that property at its default of `never`, so none
// of these strings reaches a client — but they are what a reader sees in a log
// or a debugger, and they are cheap to get right.

// MethodArgumentNotValidError is org.springframework.web.bind.MethodArgumentNotValidException:
// a @Valid @RequestBody failed validation. It carries the field errors so an
// advice can render them.
type MethodArgumentNotValidError struct {
	// FieldErrors is the binding result, in the order validation produced it.
	FieldErrors []FieldError
}

// FieldError is org.springframework.validation.FieldError.
type FieldError struct {
	Field          string
	DefaultMessage string
}

func (e *MethodArgumentNotValidError) Error() string {
	return "Validation failed for argument with " +
		strconv.Itoa(len(e.FieldErrors)) + " error(s)"
}

// httpMessageNotReadableError is org.springframework.http.converter.HttpMessageNotReadableException:
// the request body was absent or could not be parsed as JSON.
type httpMessageNotReadableError struct{ reason string }

func (e *httpMessageNotReadableError) Error() string { return e.reason }

// missingRequestParamError is org.springframework.web.bind.MissingServletRequestParameterException.
type missingRequestParamError struct{ name string }

func (e *missingRequestParamError) Error() string {
	return "Required request parameter '" + e.name +
		"' for method parameter type String is not present"
}

// typeMismatchError is org.springframework.beans.TypeMismatchException, raised
// when a path variable or request parameter will not convert to the declared type.
type typeMismatchError struct{ value string }

func (e *typeMismatchError) Error() string {
	return "Failed to convert value of type 'java.lang.String' to required type " +
		`'java.lang.Integer'; nested exception is java.lang.NumberFormatException: ` +
		`For input string: "` + e.value + `"`
}

// errorBody is Spring Boot's DefaultErrorAttributes response, in the order
// BasicErrorController writes it.
//
// `message` is empty because Spring Boot 2.3 onwards defaults
// server.error.include-message to `never`, and this application overrides
// nothing — application.properties is empty.
type errorBody struct {
	Timestamp string `json:"timestamp"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

// errorTimestampLayout renders the instant the way Spring Boot's ISO-8601
// serialiser does: milliseconds, and a colon-separated UTC offset.
const errorTimestampLayout = "2006-01-02T15:04:05.000-07:00"

// now is a variable so a test can pin the timestamp; production never replaces it.
var now = time.Now

// errorResponse builds Spring Boot's default error page for a status and path.
func errorResponse(status int, path string) ResponseEntity {
	return Status(status, JSON(errorBody{
		Timestamp: now().UTC().Format(errorTimestampLayout),
		Status:    status,
		Error:     http.StatusText(status),
		Message:   "",
		Path:      path,
	}))
}
