package validation

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
)

// RegionMessage is @Region's default message.
const RegionMessage = "Region 值不在可选范围内"

// regions is the set RegionValidator builds on every call.
var regions = map[string]struct{}{
	"China":          {},
	"China-Taiwan":   {},
	"China-HongKong": {},
}

// Region is the @Region constraint annotation, validated by IsValidRegion.
func Region() beanvalidation.Constraint {
	return beanvalidation.StringConstraint("Region", RegionMessage, IsValidRegion)
}

// IsValidRegion is RegionValidator.isValid.
//
// Unlike PhoneNumberValidator this one does not null-guard: it asks the set
// whether it contains the value, and a set of three strings contains no null.
// An absent region therefore fails with RegionMessage even though @Region
// carries no @NotNull.
func IsValidRegion(value *string) bool {
	if value == nil {
		return false
	}
	_, ok := regions[*value]
	return ok
}
