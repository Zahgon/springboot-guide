package main

import (
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/controller"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/mvc"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/exception"
)

// newRouter builds the application's request mappings.
//
// This is what component scanning does in the original: it discovers the two
// @RestControllers and the @ControllerAdvice, and wires the advice to the
// controller its `assignableTypes` names — PersonController, and only it.
func newRouter() *mvc.Router {
	router := mvc.New()
	advice := exception.NewGlobalExceptionHandler().Advice()
	controller.NewHelloWorldController().Register(router)
	controller.NewPersonController().Register(router, advice...)
	return router
}
