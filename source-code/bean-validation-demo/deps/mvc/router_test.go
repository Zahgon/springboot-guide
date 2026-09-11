package mvc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The router's own dispatch behaviour: the three ways a request can miss, plus
// the two methods Spring answers without a mapping of their own. All of it was
// previously supplied by Spring MVC and is now this package's responsibility.

func testRouter() *Router {
	router := New()
	router.Handle(http.MethodGet, "/api/hello", func(*Request) (ResponseEntity, error) {
		return OK(Text("Hello")), nil
	})
	router.Handle(http.MethodPost, "/api/persons", func(*Request) (ResponseEntity, error) {
		return OK(JSON(map[string]string{"k": "v"})), nil
	})
	router.Handle(http.MethodGet, "/api/persons/{id}", func(request *Request) (ResponseEntity, error) {
		return OK(Text(request.PathVariable("id"))), nil
	})
	router.Handle(http.MethodPut, "/api/persons", func(*Request) (ResponseEntity, error) {
		return OK(Text("put")), nil
	})
	return router
}

func do(t *testing.T, method, target string) *http.Response {
	t.Helper()
	recorder := httptest.NewRecorder()
	testRouter().ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	return recorder.Result()
}

func TestRouterBindsPathVariables(t *testing.T) {
	response := do(t, http.MethodGet, "/api/persons/42")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", response.StatusCode)
	}
	body := make([]byte, 2)
	_, _ = response.Body.Read(body)
	if string(body) != "42" {
		t.Errorf("body = %q, want %q", body, "42")
	}
}

func TestRouterReturns404WithVaryHeadersForAnUnmappedPath(t *testing.T) {
	response := do(t, http.MethodGet, "/api/nope")
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", response.StatusCode)
	}
	vary := response.Header.Values("Vary")
	want := []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"}
	if strings.Join(vary, "|") != strings.Join(want, "|") {
		t.Errorf("Vary = %v, want %v", vary, want)
	}
}

func TestRouterReturns405WithACommaSpaceAllowHeader(t *testing.T) {
	// DefaultHandlerExceptionResolver joins with ", ".
	for _, testCase := range []struct{ method, target, allow string }{
		{http.MethodDelete, "/api/persons", "POST, PUT"},
		{http.MethodPatch, "/api/persons", "POST, PUT"},
		{http.MethodPost, "/api/hello", "GET"},
		{http.MethodHead, "/api/persons", "POST, PUT"},
	} {
		response := do(t, testCase.method, testCase.target)
		if response.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, want 405", testCase.method, testCase.target, response.StatusCode)
		}
		if got := response.Header.Get("Allow"); got != testCase.allow {
			t.Errorf("%s %s: Allow = %q, want %q", testCase.method, testCase.target, got, testCase.allow)
		}
	}
}

func TestRouterMatchesATrailingSlashAgainstTheBarePath(t *testing.T) {
	// `/api/persons/` reaches the /api/persons mapping rather than binding an
	// empty {id}, so it is a 405 and not a 200.
	response := do(t, http.MethodGet, "/api/persons/")
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", response.StatusCode)
	}
	if got := response.Header.Get("Allow"); got != "POST, PUT" {
		t.Errorf("Allow = %q, want %q", got, "POST, PUT")
	}
	response = do(t, http.MethodGet, "/api/hello/")
	if response.StatusCode != http.StatusOK {
		t.Errorf("/api/hello/ status = %d, want 200", response.StatusCode)
	}
}

func TestRouterAnswersHEADFromTheGETMapping(t *testing.T) {
	response := do(t, http.MethodHead, "/api/hello")
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", response.StatusCode)
	}
	if got := response.Header.Get("Content-Type"); got != TextPlainUTF8 {
		t.Errorf("Content-Type = %q, want %q", got, TextPlainUTF8)
	}
}

func TestRouterAnswersOPTIONSWithACommaJoinedAllowList(t *testing.T) {
	// HttpOptionsHandler uses HttpHeaders.setAllow, which joins with a bare
	// comma — a different separator from the 405 above. HEAD follows GET, and
	// OPTIONS is appended last.
	for _, testCase := range []struct{ target, allow string }{
		{"/api/hello", "GET,HEAD,OPTIONS"},
		{"/api/persons", "POST,PUT,OPTIONS"},
		{"/api/persons/1", "GET,HEAD,OPTIONS"},
	} {
		response := do(t, http.MethodOptions, testCase.target)
		if response.StatusCode != http.StatusOK {
			t.Errorf("OPTIONS %s: status = %d, want 200", testCase.target, response.StatusCode)
		}
		if got := response.Header.Get("Allow"); got != testCase.allow {
			t.Errorf("OPTIONS %s: Allow = %q, want %q", testCase.target, got, testCase.allow)
		}
		if _, present := response.Header["Accept-Patch"]; !present {
			t.Errorf("OPTIONS %s: no Accept-Patch header", testCase.target)
		}
		if got := response.Header.Get("Content-Type"); got != "" {
			t.Errorf("OPTIONS %s: Content-Type = %q, want none", testCase.target, got)
		}
	}
}

func TestRouterReturns404ForOPTIONSOnAnUnmappedPath(t *testing.T) {
	if got := do(t, http.MethodOptions, "/api/nope").StatusCode; got != http.StatusNotFound {
		t.Errorf("status = %d, want 404", got)
	}
}

func TestResponseFramingFollowsTheBodyKind(t *testing.T) {
	// A String body carries a Content-Length; a Jackson-serialised body is
	// streamed chunked. The difference is visible on the wire.
	text := do(t, http.MethodGet, "/api/persons/7")
	if got := text.Header.Get("Content-Length"); got != "1" {
		t.Errorf("text Content-Length = %q, want %q", got, "1")
	}
	if got := text.Header.Get("Transfer-Encoding"); got != "" {
		t.Errorf("text Transfer-Encoding = %q, want none", got)
	}
	object := do(t, http.MethodPost, "/api/persons")
	if got := object.Header.Get("Transfer-Encoding"); got != "chunked" {
		t.Errorf("json Transfer-Encoding = %q, want %q", got, "chunked")
	}
}

func TestJSONDoesNotEscapeHTMLCharacters(t *testing.T) {
	// Jackson leaves <, > and & alone; encoding/json escapes them by default,
	// which would change every body containing one.
	body := JSON(map[string]string{"k": "a<b>&c"})
	if got, want := string(body.bytes), `{"k":"a<b>&c"}`; got != want {
		t.Errorf("JSON = %s, want %s", got, want)
	}
}

func TestErrorResponseCarriesSpringBootsDefaultAttributes(t *testing.T) {
	response := do(t, http.MethodGet, "/api/nope")
	payload := make([]byte, 512)
	n, _ := response.Body.Read(payload)
	body := string(payload[:n])
	for _, fragment := range []string{
		`"status":404`, `"error":"Not Found"`, `"message":""`, `"path":"/api/nope"`, `"timestamp":"`,
	} {
		if !strings.Contains(body, fragment) {
			t.Errorf("body %s does not contain %s", body, fragment)
		}
	}
	// The attributes appear in BasicErrorController's own order.
	if !strings.HasPrefix(body, `{"timestamp":"`) {
		t.Errorf("body does not start with timestamp: %s", body)
	}
}

func TestFrameworkExceptionMessagesMatchSpring(t *testing.T) {
	// Recorded from the original with server.error.include-message=always. The
	// shipped configuration leaves the property at `never`, so these strings do
	// not reach a client; they are the text a log or debugger shows.
	for _, testCase := range []struct {
		err  error
		want string
	}{
		{
			&unsupportedMediaTypeError{received: "text/plain;charset=UTF-8", supported: jsonMediaTypes},
			"Content type 'text/plain;charset=UTF-8' not supported",
		},
		{
			&missingRequestParamError{name: "name"},
			"Required request parameter 'name' for method parameter type String is not present",
		},
		{
			&typeMismatchError{value: "abc"},
			"Failed to convert value of type 'java.lang.String' to required type " +
				`'java.lang.Integer'; nested exception is java.lang.NumberFormatException: ` +
				`For input string: "abc"`,
		},
		{
			&httpMessageNotReadableError{reason: missingBodyReason},
			"Required request body is missing",
		},
		{
			&MethodArgumentNotValidError{FieldErrors: []FieldError{{Field: "name"}, {Field: "sex"}}},
			"Validation failed for argument with 2 error(s)",
		},
	} {
		if got := testCase.err.Error(); got != testCase.want {
			t.Errorf("Error() = %q, want %q", got, testCase.want)
		}
	}
}

// The response-framing details below are part of the contract recorded in
// instruction.md §R3/§R4/§R6 and were verified against the running original by
// the differential in truth.md §9. They are asserted here as well because a
// differential is a one-time act while a test is the regression net: mutation
// testing showed each of these could be broken without any test noticing.

func TestUnsupportedMediaTypeAdvertisesTheConsumableTypes(t *testing.T) {
	// Spring lists what MappingJackson2HttpMessageConverter can read.
	router := New()
	router.Handle(http.MethodPost, "/api/persons",
		ValidJSONBody(func(*struct{}) (ResponseEntity, error) { return OK(Text("")), nil }))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/persons", strings.NewReader("{}"))
	request.Header.Set("Content-Type", "text/plain")
	router.ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", response.StatusCode)
	}
	if got, want := response.Header.Get("Accept"), "application/json, application/*+json"; got != want {
		t.Errorf("Accept = %q, want %q", got, want)
	}
}

func TestTextResponsesCarryTheCharsetParameter(t *testing.T) {
	// Spring writes a String body as `text/plain;charset=UTF-8` — with the
	// charset, and with no space after the semicolon.
	response := do(t, http.MethodGet, "/api/persons/7")
	if got, want := response.Header.Get("Content-Type"), "text/plain;charset=UTF-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestErrorBodyTimestampHasMillisecondsAndAUTCOffset(t *testing.T) {
	// Spring Boot's ISO-8601 serialiser emits `2026-09-08T06:33:40.757+00:00`.
	// Dropping the milliseconds or using `Z` would both still parse as a time.
	response := do(t, http.MethodGet, "/api/nope")
	payload := make([]byte, 512)
	n, _ := response.Body.Read(payload)
	var body struct {
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(payload[:n], &body); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}\+00:00$`).MatchString(body.Timestamp) {
		t.Errorf("timestamp = %q, want the form 2026-09-08T06:33:40.757+00:00", body.Timestamp)
	}
	if _, err := time.Parse(errorTimestampLayout, body.Timestamp); err != nil {
		t.Errorf("timestamp %q does not parse under the declared layout: %v", body.Timestamp, err)
	}
}

func TestClosingMarksTheConnectionForClose(t *testing.T) {
	// Tomcat sends `Connection: close` when it replies without having drained
	// the request body, which is every validation failure on a POST.
	recorder := httptest.NewRecorder()
	Status(http.StatusBadRequest, Text("x")).Closing().write(recorder)
	if got := recorder.Result().Header.Get("Connection"); got != "close" {
		t.Errorf("Connection = %q, want %q", got, "close")
	}
	recorder = httptest.NewRecorder()
	Status(http.StatusBadRequest, Text("x")).write(recorder)
	if got := recorder.Result().Header.Get("Connection"); got != "" {
		t.Errorf("Connection = %q on a non-closing entity, want none", got)
	}
}
