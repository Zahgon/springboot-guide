// Package controller mirrors com.example.beanvalidationdemo.controller.
package controller

import (
	"net/http"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/mvc"
)

// HelloWorldController is the @RestController at /api.
//
// 验证基本环境搭建是否正确
type HelloWorldController struct{}

// NewHelloWorldController returns the controller bean.
func NewHelloWorldController() *HelloWorldController { return &HelloWorldController{} }

// HelloPath is the @GetMapping this controller answers.
const HelloPath = "/api/hello"

// Hello is @GetMapping("/hello"). A String return value is written as
// text/plain with a Content-Length.
func (c *HelloWorldController) Hello() string { return "Hello" }

// Register maps the controller onto a router.
func (c *HelloWorldController) Register(router *mvc.Router) {
	router.Handle(http.MethodGet, HelloPath, func(*mvc.Request) (mvc.ResponseEntity, error) {
		return mvc.OK(mvc.Text(c.Hello())), nil
	})
}
