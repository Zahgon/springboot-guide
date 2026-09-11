// Package exception mirrors com.example.beanvalidationdemo.exception.
package exception

import (
	"net/http"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/javamap"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/mvc"
)

// GlobalExceptionHandler is the
// @ControllerAdvice(assignableTypes = {PersonController.class}).
//
// It is scoped to PersonController, so an exception from any other controller
// still reaches Spring Boot's default error page. That scoping is expressed by
// registering Advice only on PersonController's mappings.
type GlobalExceptionHandler struct{}

// NewGlobalExceptionHandler returns the advice bean.
func NewGlobalExceptionHandler() *GlobalExceptionHandler { return &GlobalExceptionHandler{} }

// Advice returns the handler's @ExceptionHandler methods, in declaration order.
func (h *GlobalExceptionHandler) Advice() []mvc.Advice {
	return []mvc.Advice{h.HandleValidationExceptions, h.HandleConstraintViolationException}
}

// HandleValidationExceptions is @ExceptionHandler(MethodArgumentNotValidException.class).
//
// It collects the binding result into a `new HashMap<>()` keyed by field name
// and returns it as the 400 body. Two consequences of that being a HashMap are
// visible to a client, and both are reproduced by javamap.StringMap: the JSON
// keys come out in bucket order rather than the order validation found them,
// and a field with two violations keeps only the last message written.
func (h *GlobalExceptionHandler) HandleValidationExceptions(err error) (mvc.ResponseEntity, bool) {
	invalid, ok := mvc.AsMethodArgumentNotValidError(err)
	if !ok {
		return mvc.ResponseEntity{}, false
	}
	errors := javamap.NewStringMap()
	for _, fieldError := range invalid.FieldErrors {
		errors.Put(fieldError.Field, fieldError.DefaultMessage)
	}
	return mvc.Status(http.StatusBadRequest, mvc.JSON(errors)).Closing(), true
}

// HandleConstraintViolationException is @ExceptionHandler(ConstraintViolationException.class).
//
// The body is the exception's own message, which Bean Validation renders as
// `propertyPath: message` joined with ", ".
func (h *GlobalExceptionHandler) HandleConstraintViolationException(err error) (mvc.ResponseEntity, bool) {
	violation, ok := mvc.AsViolationError(err)
	if !ok {
		return mvc.ResponseEntity{}, false
	}
	return mvc.Status(http.StatusBadRequest, mvc.Text(violation.Error())).Closing(), true
}
