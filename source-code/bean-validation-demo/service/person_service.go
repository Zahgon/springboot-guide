// Package service mirrors com.example.beanvalidationdemo.service.
package service

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/entity"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/service/groups"
)

// PersonService is the @Validated @Service.
//
// Spring wraps a @Validated bean in a proxy that validates the constrained
// parameters of every call and throws ConstraintViolationException before the
// method body runs. Go has no proxies, so each method opens with the check the
// proxy would have performed. The bodies below the check are the originals',
// which is to say empty.
type PersonService struct{}

// NewPersonService returns the service bean.
func NewPersonService() *PersonService { return &PersonService{} }

// ValidatePersonRequest validates a PersonRequest under the Default group.
func (s *PersonService) ValidatePersonRequest(personRequest *entity.PersonRequest) error {
	if err := beanvalidation.ValidateParameters("validatePersonRequest",
		[]beanvalidation.Parameter{
			beanvalidation.Param("personRequest", personRequest, beanvalidation.Valid()),
		}); err != nil {
		return err
	}
	// do something
	return nil
}

// ValidatePersonGroupForAdd validates a Person under AddPersonGroup, where
// `group` must be null.
func (s *PersonService) ValidatePersonGroupForAdd(person *entity.Person) error {
	if err := beanvalidation.ValidateParameters("validatePersonGroupForAdd",
		[]beanvalidation.Parameter{
			beanvalidation.Param("person", person, beanvalidation.Valid()),
		}, groups.AddPersonGroup); err != nil {
		return err
	}
	// do something
	return nil
}

// ValidatePersonGroupForDelete validates a Person under DeletePersonGroup,
// where `group` must not be null.
func (s *PersonService) ValidatePersonGroupForDelete(person *entity.Person) error {
	if err := beanvalidation.ValidateParameters("validatePersonGroupForDelete",
		[]beanvalidation.Parameter{
			beanvalidation.Param("person", person, beanvalidation.Valid()),
		}, groups.DeletePersonGroup); err != nil {
		return err
	}
	// do something
	return nil
}
