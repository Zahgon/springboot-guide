package mvc

import (
	"errors"
	"net/http"
	"strings"
)

// Handler is one @RequestMapping method after its arguments have been bound.
type Handler func(request *Request) (ResponseEntity, error)

// Advice is one @ExceptionHandler method: it either renders the error or
// declines it, in which case the next advice, and finally Spring Boot's default
// error page, gets the chance.
type Advice func(err error) (ResponseEntity, bool)

// Request carries the bound path variables alongside the raw HTTP request.
type Request struct {
	*http.Request
	PathVariables map[string]string
}

// PathVariable returns the value bound to a {name} segment.
func (r *Request) PathVariable(name string) string { return r.PathVariables[name] }

// Router is the DispatcherServlet: it matches a request to a mapping, applies
// the mapping's controller advice to whatever the handler raises, and falls
// back to Spring Boot's error page.
type Router struct {
	mappings []mapping
}

type mapping struct {
	method   string
	segments []string
	handler  Handler
	advice   []Advice
}

// New returns an empty Router.
func New() *Router { return &Router{} }

// Handle registers a @RequestMapping. The pattern may contain `{name}`
// segments, each of which binds one non-empty path segment.
//
// advice is the @ControllerAdvice that applies to this mapping's controller.
// Registering it per mapping is how `@ControllerAdvice(assignableTypes = ...)`
// is expressed without a bean container: the advice reaches exactly the
// controllers it was declared for, and no others.
func (r *Router) Handle(method, pattern string, handler Handler, advice ...Advice) {
	r.mappings = append(r.mappings, mapping{
		method:   method,
		segments: splitPath(pattern),
		handler:  handler,
		advice:   advice,
	})
}

// requestMethodOrder is org.springframework.web.bind.annotation.RequestMethod's
// declaration order, which is the order Spring lists in an `Allow` header.
var requestMethodOrder = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
	http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodTrace,
}

// ServeHTTP dispatches one request, reproducing Spring's three failure modes:
// no mapping for the path (404), a mapping for the path but not the method
// (405 with Allow), and a mapping that raises before or during handling.
func (r *Router) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Spring's default trailing-slash match: `/api/persons/` reaches the
	// mapping registered as `/api/persons` rather than binding an empty {id}.
	// splitPath discards empty segments, which is what implements it; an
	// explicit TrimSuffix here read as load-bearing but changed nothing.
	segments := splitPath(request.URL.Path)

	var byPath []mapping
	for _, candidate := range r.mappings {
		if _, ok := bind(candidate.segments, segments); ok {
			byPath = append(byPath, candidate)
		}
	}
	if len(byPath) == 0 {
		// The Vary headers come from the CORS processor, which runs on the
		// no-handler-found path and advertises the request headers it inspected.
		errorResponse(http.StatusNotFound, request.URL.Path).
			WithHeader("Vary", "Origin").
			WithHeader("Vary", "Access-Control-Request-Method").
			WithHeader("Vary", "Access-Control-Request-Headers").
			write(writer)
		return
	}

	// A GET mapping answers HEAD as well: Spring records HEAD support on every
	// GET mapping, and the container suppresses the body while keeping the
	// headers the GET would have produced.
	wanted := request.Method
	if wanted == http.MethodHead && !mapped(byPath, http.MethodHead) {
		wanted = http.MethodGet
	}

	for _, candidate := range byPath {
		if candidate.method != wanted {
			continue
		}
		variables, _ := bind(candidate.segments, segments)
		candidate.serve(writer, &Request{Request: request, PathVariables: variables})
		return
	}

	declared := declaredMethods(byPath)

	// An OPTIONS request that no mapping claims is answered by Spring's own
	// HttpOptionsHandler rather than rejected.
	if request.Method == http.MethodOptions {
		optionsResponse(declared).write(writer)
		return
	}

	// DefaultHandlerExceptionResolver joins the supported methods with ", ",
	// which is a different separator from the one the OPTIONS handler uses.
	errorResponse(http.StatusMethodNotAllowed, request.URL.Path).
		WithHeader("Allow", strings.Join(declared, ", ")).
		write(writer)
}

// mapped reports whether any of the mappings claims method.
func mapped(mappings []mapping, method string) bool {
	for _, candidate := range mappings {
		if candidate.method == method {
			return true
		}
	}
	return false
}

// declaredMethods returns the methods the mappings claim, in RequestMethod
// declaration order.
func declaredMethods(mappings []mapping) []string {
	declared := make([]string, 0, len(mappings))
	for _, method := range requestMethodOrder {
		if mapped(mappings, method) {
			declared = append(declared, method)
		}
	}
	return declared
}

// optionsResponse is what RequestMappingInfoHandlerMapping.HttpOptionsHandler
// produces: 200 with an Allow header, an empty Accept-Patch, no content type
// and no body.
//
// HEAD joins the list whenever GET is mapped, OPTIONS is appended last, and
// HttpHeaders.setAllow joins with a bare comma — unlike the 405 above.
func optionsResponse(declared []string) ResponseEntity {
	allowed := make([]string, 0, len(declared)+2)
	for _, method := range declared {
		allowed = append(allowed, method)
		if method == http.MethodGet {
			allowed = append(allowed, http.MethodHead)
		}
	}
	allowed = append(allowed, http.MethodOptions)
	return Status(http.StatusOK, NoBody()).
		WithHeader("Allow", strings.Join(allowed, ",")).
		WithHeader("Accept-Patch", "")
}

// serve invokes the handler and routes any error through the advice chain.
func (m mapping) serve(writer http.ResponseWriter, request *Request) {
	entity, err := m.handler(request)
	if err == nil {
		entity.write(writer)
		return
	}
	for _, advice := range m.advice {
		if handled, ok := advice(err); ok {
			handled.write(writer)
			return
		}
	}
	defaultErrorPage(err, request).write(writer)
}

// defaultErrorPage maps an unhandled framework exception to the status Spring
// Boot's ErrorController would report for it.
//
// Every one of these arises after the request line has been read but before the
// body has been drained, so Tomcat marks the connection for closing — except
// the 415, which is decided from the headers alone.
func defaultErrorPage(err error, request *Request) ResponseEntity {
	path := request.URL.Path
	var unsupported *unsupportedMediaTypeError
	if errors.As(err, &unsupported) {
		return errorResponse(http.StatusUnsupportedMediaType, path).
			WithHeader("Accept", strings.Join(unsupported.supported, ", "))
	}
	var unreadable *httpMessageNotReadableError
	var missing *missingRequestParamError
	var mismatch *typeMismatchError
	if errors.As(err, &unreadable) || errors.As(err, &missing) || errors.As(err, &mismatch) {
		return errorResponse(http.StatusBadRequest, path).Closing()
	}
	return errorResponse(http.StatusInternalServerError, path).Closing()
}

// splitPath returns a path's non-empty segments.
func splitPath(path string) []string {
	var segments []string
	for _, segment := range strings.Split(path, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	return segments
}

// bind matches a mapping's segments against a request's, returning the path
// variables. A `{name}` segment binds exactly one non-empty segment.
func bind(pattern, actual []string) (map[string]string, bool) {
	if len(pattern) != len(actual) {
		return nil, false
	}
	variables := map[string]string{}
	for i, segment := range pattern {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			variables[segment[1:len(segment)-1]] = actual[i]
			continue
		}
		if segment != actual[i] {
			return nil, false
		}
	}
	return variables, true
}
