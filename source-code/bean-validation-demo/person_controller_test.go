package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/entity"
)

// Port of PersonControllerTest.java. httptest.Server stands in for
// @SpringBootTest + @AutoConfigureMockMvc: it runs the real router over a real
// socket, so the assertions see the bytes a client would.

// mockMvc starts the application's router on a loopback listener.
func mockMvc(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(newRouter())
	t.Cleanup(server.Close)
	return server
}

// perform issues one request and returns the status and body.
func perform(t *testing.T, server *httptest.Server, method, path, contentType, body string) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request, err := http.NewRequest(method, server.URL+path, reader)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("performing request: %v", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading response: %v", err)
	}
	return response.StatusCode, string(payload)
}

// jsonPath reads a top-level key out of a JSON object body, the equivalent of
// MockMvcResultMatchers.jsonPath("key").
func jsonPath(t *testing.T, body, key string) string {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal([]byte(body), &decoded); err != nil {
		t.Fatalf("body is not a JSON object: %v (%s)", err, body)
	}
	value, ok := decoded[key]
	if !ok {
		t.Fatalf("no JSON path %q in %s", key, body)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("JSON path %q is %T, want string", key, value)
	}
	return text
}

// 验证出现参数不合法的情况抛出异常并且可以正确被捕获
func TestShouldCheckPersonValue(t *testing.T) {
	server := mockMvc(t)
	personRequest := entity.NewPersonRequestBuilder().Sex("Man22").
		ClassID("82938390").
		Region("Shanghai").
		PhoneNumber("1816313815").Build()
	encoded, err := json.Marshal(personRequest)
	if err != nil {
		t.Fatalf("serialising request: %v", err)
	}

	_, body := perform(t, server, http.MethodPost, "/api/persons", "application/json", string(encoded))

	for _, expectation := range []struct{ path, want string }{
		{"sex", "sex 值不在可选范围"},
		{"name", "name 不能为空"},
		{"region", "Region 值不在可选范围内"},
		{"phoneNumber", "phoneNumber 格式不正确"},
	} {
		if got := jsonPath(t, body, expectation.path); got != expectation.want {
			t.Errorf("jsonPath(%q) = %q, want %q", expectation.path, got, expectation.want)
		}
	}

	// The original delegated this ordering to java.util.HashMap, so it was
	// covered by the JDK's tests and by nothing in this project. It is now this
	// port's own code and needs an assertion of its own.
	if !strings.HasPrefix(body, `{"phoneNumber":`) {
		t.Errorf("field errors are not in java.util.HashMap order: %s", body)
	}
	want := `{"phoneNumber":"phoneNumber 格式不正确","sex":"sex 值不在可选范围",` +
		`"name":"name 不能为空","region":"Region 值不在可选范围内"}`
	if body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}

func TestShouldCheckPathVariable(t *testing.T) {
	server := mockMvc(t)

	status, body := perform(t, server, http.MethodGet, "/api/persons/6", "application/json", "")

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if want := "getPersonByID.id: 超过 id 的范围了"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestShouldCheckRequestParamValue2(t *testing.T) {
	server := mockMvc(t)

	status, body := perform(t, server, http.MethodPut,
		"/api/persons?name=snailclimbsnailclimb", "application/json", "")

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if want := "getPersonByName.name: 超过 name 的范围了"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

// 手动校验对象
func TestCheckPersonManually(t *testing.T) {
	personRequest := entity.NewPersonRequestBuilder().Sex("Man22").
		ClassID("82938390").Build()

	violations := beanvalidation.Validate(personRequest)

	for _, violation := range violations {
		fmt.Println(violation.Message)
	}

	// The Java test printed the messages and asserted nothing. Its intent — that
	// validating a bean directly, outside any Spring plumbing, finds the same
	// violations the HTTP layer reports — is made explicit here rather than left
	// to a reader of stdout. The bean under test is unchanged.
	//
	// One caveat on the comparison below: it pins the *order* too, and the
	// original does not promise one. javax.validation.Validator.validate returns
	// a Set, whose iteration order is unspecified; this port returns declaration
	// order, which is a property of the shim rather than of the contract. It is
	// asserted because determinism is worth keeping, not because the original
	// guarantees it — and nothing observable depends on it: the 400 body is
	// ordered by HashMap bucket, and no two of these five field names collide,
	// so the order violations are found in cannot reach a response.
	got := make([]string, len(violations))
	for i, violation := range violations {
		got[i] = violation.PropertyPath + "=" + violation.Message
	}
	want := []string{
		"name=name 不能为空",
		"sex=sex 值不在可选范围",
		"region=Region 值不在可选范围内",
		"phoneNumber=phoneNumber 不能为空",
	}
	if len(got) != len(want) {
		t.Fatalf("violations = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("violation %d = %q, want %q", i, got[i], want[i])
		}
	}
}
