package beanvalidation

// Method validation is what Spring's @Validated gives a bean: the container
// wraps it in a proxy that validates constrained parameters on every call and
// throws ConstraintViolationException before the body runs.
//
// Go has no proxies and no interception, so there is nothing to hide the check
// behind: a @Validated method opens by calling ValidateParameters itself. That
// is the honest translation — the interception point becomes a visible line —
// and it produces the same violations, the same property paths and the same
// exception message the proxy would have.

// Parameter is one constrained method parameter: its name as it appears in the
// property path, its value, and the constraints declared on it.
type Parameter struct {
	Name        string
	Value       any
	Constraints []Constraint
}

// Param declares a constrained method parameter.
func Param(name string, value any, constraints ...Constraint) Parameter {
	return Parameter{Name: name, Value: value, Constraints: constraints}
}

// ValidateParameters validates a @Validated method's parameters and returns a
// *ViolationError when any constraint fails, or nil when they all pass.
//
// Property paths follow Hibernate's method-validation scheme: `method.parameter`
// for a constraint declared directly on the parameter, and
// `method.parameter.property` for a violation found by cascading into it
// through @Valid.
func ValidateParameters(method string, parameters []Parameter, groups ...Group) error {
	var violations []Violation
	for _, parameter := range parameters {
		prefix := method + "." + parameter.Name
		for _, constraint := range parameter.Constraints {
			if constraint.isCascade() {
				for _, nested := range Validate(parameter.Value, groups...) {
					nested.PropertyPath = prefix + "." + nested.PropertyPath
					violations = append(violations, nested)
				}
				continue
			}
			if !constraint.appliesTo(groups) || constraint.check(parameter.Value) {
				continue
			}
			violations = append(violations, Violation{
				PropertyPath: prefix,
				Message:      constraint.resolvedMessage(),
				InvalidValue: parameter.Value,
			})
		}
	}
	if len(violations) == 0 {
		return nil
	}
	return &ViolationError{Violations: violations}
}
