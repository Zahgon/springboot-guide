package controller

import (
	"net/http"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/mvc"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/entity"
)

// The two method-level constraint messages, quoted from the annotations.
const (
	IDMaxMessage    = "超过 id 的范围了"
	NameSizeMessage = "超过 name 的范围了"
)

// idMax is @Max(value = 5) on getPersonByID's id parameter.
const idMax = 5

// nameMaxSize is @Size(max = 6) on getPersonByName's name parameter.
const nameMaxSize = 6

// Paths the controller is mapped to.
const (
	PersonsPath    = "/api/persons"
	PersonByIDPath = "/api/persons/{id}"
)

// PersonController is the @RestController @Validated at /api/persons.
//
// @Validated makes Spring enforce the parameter constraints below through a
// proxy. With no proxy available, each method opens with the check the proxy
// would have run; a failure is the same ConstraintViolationException, which
// GlobalExceptionHandler turns into the response body.
type PersonController struct{}

// NewPersonController returns the controller bean.
func NewPersonController() *PersonController { return &PersonController{} }

// Save is @PostMapping: it echoes back the request body that passed validation.
//
// The body is validated by the argument resolver before this method is entered,
// so by the time it runs there is nothing left to check.
func (c *PersonController) Save(personRequest *entity.PersonRequest) (mvc.ResponseEntity, error) {
	return mvc.OK(mvc.JSON(personRequest)), nil
}

// GetPersonByID is @GetMapping("/{id}") with @Max(5) on the path variable.
func (c *PersonController) GetPersonByID(id int) (mvc.ResponseEntity, error) {
	if err := beanvalidation.ValidateParameters("getPersonByID",
		[]beanvalidation.Parameter{
			beanvalidation.Param("id", id,
				beanvalidation.Max(idMax).WithMessage(IDMaxMessage)),
		}); err != nil {
		return mvc.ResponseEntity{}, err
	}
	return mvc.OK(mvc.JSON(id)), nil
}

// GetPersonByName is @PutMapping with @Size(max = 6) on the request parameter.
func (c *PersonController) GetPersonByName(name string) (mvc.ResponseEntity, error) {
	if err := beanvalidation.ValidateParameters("getPersonByName",
		[]beanvalidation.Parameter{
			beanvalidation.Param("name", name,
				beanvalidation.Size(0, nameMaxSize).WithMessage(NameSizeMessage)),
		}); err != nil {
		return mvc.ResponseEntity{}, err
	}
	return mvc.OK(mvc.Text(name)), nil
}

// Register maps the controller onto a router, behind the controller advice that
// @ControllerAdvice(assignableTypes = PersonController.class) declares for it.
func (c *PersonController) Register(router *mvc.Router, advice ...mvc.Advice) {
	router.Handle(http.MethodPost, PersonsPath, mvc.ValidJSONBody(c.Save), advice...)
	router.Handle(http.MethodGet, PersonByIDPath, mvc.IntPathVariable("id", c.GetPersonByID), advice...)
	router.Handle(http.MethodPut, PersonsPath, mvc.StringRequestParam("name", c.GetPersonByName), advice...)
}
