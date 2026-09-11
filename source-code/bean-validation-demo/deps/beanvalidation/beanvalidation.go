// Package beanvalidation reproduces the part of JSR-380 Bean Validation, as
// implemented by Hibernate Validator, that this application's behaviour depends
// on: the built-in constraints and their null rules, validation groups, message
// interpolation, and the way a ConstraintViolationException renders itself.
//
// Java declares constraints as annotations and Hibernate discovers them by
// reflection. Go has no annotations, so a type states its constraints by
// implementing Bean, and the engine reads field values reflectively by their
// JSON name. The engine is deliberately narrow: it implements the constraints
// this application actually declares, not the whole specification.
package beanvalidation

import (
	"reflect"
	"strings"
)

// Group identifies a validation group. In Java a group is a marker interface
// and is named by its Class; here it is named by its fully qualified name, so
// that a violation's group set is as readable as the Java one.
type Group string

// Default is javax.validation.groups.Default, the group every constraint
// belongs to unless it names others.
const Default Group = "javax.validation.groups.Default"

// Violation mirrors javax.validation.ConstraintViolation.
type Violation struct {
	// PropertyPath is the Java property path: a bean property name for bean
	// validation, and `method.parameter[.property]` for method validation.
	PropertyPath string
	// Message is the interpolated constraint message.
	Message string
	// InvalidValue is the value that failed the constraint.
	InvalidValue any
}

// ViolationError mirrors javax.validation.ConstraintViolationException.
type ViolationError struct {
	Violations []Violation
}

// Error renders the violations the way ConstraintViolationException does:
// `propertyPath: message`, joined with ", ". This string is user surface — the
// controller advice returns it as the response body — so the separator and the
// single space after the colon are part of the contract.
func (e *ViolationError) Error() string {
	parts := make([]string, len(e.Violations))
	for i, violation := range e.Violations {
		parts[i] = violation.PropertyPath + ": " + violation.Message
	}
	return strings.Join(parts, ", ")
}

// Bean is implemented by a type that declares constraints on its properties,
// the equivalent of a class carrying constraint annotations on its fields.
type Bean interface {
	Constraints() Constraints
}

// Property binds a set of constraints to one bean property, named as the
// property is named on the wire.
type Property struct {
	Name        string
	Constraints []Constraint
}

// Constraints is a bean's full constraint declaration, in field order.
type Constraints []Property

// On declares the constraints that apply to one property.
func On(name string, constraints ...Constraint) Property {
	return Property{Name: name, Constraints: constraints}
}

// Validate applies bean's constraints and returns the violations, in property
// declaration order and then constraint declaration order.
//
// Passing no groups validates the Default group, which is what
// `validator.validate(bean)` does in Java.
func Validate(bean any, groups ...Group) []Violation {
	declaration, values, ok := describe(bean)
	if !ok {
		return nil
	}
	var violations []Violation
	for _, property := range declaration {
		value := values[property.Name]
		for _, constraint := range property.Constraints {
			if !constraint.appliesTo(groups) {
				continue
			}
			if constraint.check(value) {
				continue
			}
			violations = append(violations, Violation{
				PropertyPath: property.Name,
				Message:      constraint.resolvedMessage(),
				InvalidValue: value,
			})
		}
	}
	return violations
}

// describe returns a bean's constraint declaration alongside its property
// values, keyed by the property names the declaration uses.
func describe(bean any) (Constraints, map[string]any, bool) {
	declared, ok := bean.(Bean)
	if !ok {
		return nil, nil, false
	}
	value := reflect.ValueOf(bean)
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, nil, false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, nil, false
	}
	values := make(map[string]any, value.NumField())
	structType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}
		values[propertyName(field)] = value.Field(i).Interface()
	}
	return declared.Constraints(), values, true
}

// propertyName resolves the name a field is known by: its JSON name when it has
// one, otherwise the Go field name. Java property names and JSON names coincide
// here because Jackson derives one from the other.
func propertyName(field reflect.StructField) string {
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
