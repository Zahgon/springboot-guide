package main

import (
	"fmt"
	"testing"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/mvc"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/entity"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/service"
)

// Port of PersonServiceTest.java.
//
// These three tests are declared in the original but never execute there:
// spring-boot-starter-test 2.4.5 excludes junit-vintage-engine, so surefire's
// JUnit Platform provider finds no engine for the file's JUnit 4 annotations
// and skips it without reporting a skip. Adding the vintage engine locally
// confirms all three pass. They run here.

func TestShouldThrowExceptionWhenPersonRequestIsNotValid(t *testing.T) {
	personService := service.NewPersonService()
	personRequest := entity.NewPersonRequestBuilder().Sex("Man22").
		ClassID("82938390").Build()

	err := personService.ValidatePersonRequest(personRequest)

	violation, ok := mvc.AsViolationError(err)
	if !ok {
		t.Fatalf("ValidatePersonRequest returned %v, want a ConstraintViolationException", err)
	}
	for _, constraintViolation := range violation.Violations {
		fmt.Println(constraintViolation.Message)
	}

	// The Java test swallowed the exception in a catch block and asserted
	// nothing, so it passed whether or not validation ran at all. The intent is
	// that an invalid request throws; that is asserted here, and the violation
	// paths show the cascade Spring's proxy performs.
	got := make([]string, len(violation.Violations))
	for i, constraintViolation := range violation.Violations {
		got[i] = constraintViolation.PropertyPath
	}
	want := []string{
		"validatePersonRequest.personRequest.name",
		"validatePersonRequest.personRequest.sex",
		"validatePersonRequest.personRequest.region",
		"validatePersonRequest.personRequest.phoneNumber",
	}
	if len(got) != len(want) {
		t.Fatalf("violation paths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("violation path %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestShouldCheckPersonWithGroups(t *testing.T) {
	personService := service.NewPersonService()
	person := &entity.Person{}
	person.SetGroup("group1")

	err := personService.ValidatePersonGroupForAdd(person)

	if _, ok := mvc.AsViolationError(err); !ok {
		t.Fatalf("ValidatePersonGroupForAdd returned %v, want a ConstraintViolationException", err)
	}
}

func TestShouldCheckPersonWithGroups2(t *testing.T) {
	personService := service.NewPersonService()
	person := &entity.Person{}

	err := personService.ValidatePersonGroupForDelete(person)

	if _, ok := mvc.AsViolationError(err); !ok {
		t.Fatalf("ValidatePersonGroupForDelete returned %v, want a ConstraintViolationException", err)
	}
}
