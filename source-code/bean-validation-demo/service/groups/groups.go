// Package groups holds the validation-group markers PersonService selects with
// @Validated and Person's constraints name in their `groups` attribute.
//
// In Java these are two empty marker interfaces in the `service` package, and
// `entity.Person` imports them while `service.PersonService` imports
// `entity.Person`. Java tolerates that cycle between packages; Go does not
// permit an import cycle at all, so the two markers sit one level down where
// both sides can reach them. Nothing else moved, and the markers are still
// declared beside the service that selects them.
package groups

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
)

// AddPersonGroup is the AddPersonGroup marker interface.
const AddPersonGroup beanvalidation.Group = "com.example.beanvalidationdemo.service.AddPersonGroup"

// DeletePersonGroup is the DeletePersonGroup marker interface.
const DeletePersonGroup beanvalidation.Group = "com.example.beanvalidationdemo.service.DeletePersonGroup"
