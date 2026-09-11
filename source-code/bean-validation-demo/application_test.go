package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/controller"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/deps/beanvalidation"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/entity"
	"github.com/CodingDocs/springboot-guide/source-code/bean-validation-demo/validation"
)

// End-to-end assertions over the application's whole HTTP surface.
//
// The original repository had no such tests: Spring MVC, Hibernate Validator
// and Jackson supplied all of this, and were covered by their own projects'
// suites. The port implements it, so the port tests it. Every expected value
// below was recorded from the running Spring Boot application.

func TestHelloReturnsPlainText(t *testing.T) {
	server := mockMvc(t)

	status, body := perform(t, server, http.MethodGet, "/api/hello", "", "")

	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if body != "Hello" {
		t.Errorf("body = %q, want %q", body, "Hello")
	}
	if got := controller.NewHelloWorldController().Hello(); got != "Hello" {
		t.Errorf("Hello() = %q, want %q", got, "Hello")
	}
}

func TestSaveEchoesAValidRequestInDeclarationOrder(t *testing.T) {
	server := mockMvc(t)
	valid := `{"classId":"1","name":"n","sex":"Man","region":"China","phoneNumber":"18163138155"}`

	status, body := perform(t, server, http.MethodPost, "/api/persons", "application/json", valid)

	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", status, body)
	}
	// Jackson serialises a POJO in field-declaration order, which is a
	// different rule from the HashMap order of a validation failure.
	if body != valid {
		t.Errorf("body = %s, want %s", body, valid)
	}
}

func TestSaveAcceptsEveryRegionAndSexTheValidatorsAllow(t *testing.T) {
	server := mockMvc(t)
	for _, testCase := range []struct{ sex, region, phone string }{
		{"Man", "China", "18163138155"},
		{"Woman", "China-Taiwan", "13000000000"},
		{"UGM", "China-HongKong", "14500000000"},
	} {
		body := `{"classId":"1","name":"n","sex":"` + testCase.sex +
			`","region":"` + testCase.region + `","phoneNumber":"` + testCase.phone + `"}`
		if status, got := perform(t, server, http.MethodPost, "/api/persons", "application/json", body); status != http.StatusOK {
			t.Errorf("%s/%s/%s: status = %d, want 200: %s", testCase.sex, testCase.region, testCase.phone, status, got)
		}
	}
}

func TestSaveReportsEveryMissingFieldInHashMapOrder(t *testing.T) {
	server := mockMvc(t)

	status, body := perform(t, server, http.MethodPost, "/api/persons", "application/json", "{}")

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
	want := `{"classId":"classId 不能为空","phoneNumber":"phoneNumber 不能为空",` +
		`"sex":"sex 不能为空","name":"name 不能为空","region":"Region 值不在可选范围内"}`
	if body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}

func TestSaveReportsTheSizeDefaultMessageForAnOverLongName(t *testing.T) {
	server := mockMvc(t)
	long := `{"classId":"1","name":"0123456789012345678901234567890123","sex":"Man",` +
		`"region":"China","phoneNumber":"18163138155"}`

	status, body := perform(t, server, http.MethodPost, "/api/persons", "application/json", long)

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
	// English, because @Size(max = 33) supplies no message of its own.
	if want := `{"name":"size must be between 0 and 33"}`; body != want {
		t.Errorf("body = %s, want %s", body, want)
	}
}

func TestSaveTreatsAnAbsentFieldDifferentlyFromAnEmptyOne(t *testing.T) {
	server := mockMvc(t)
	base := `{"classId":"1","name":%s,"sex":"Man","region":"China","phoneNumber":"18163138155"}`

	// An empty name is a present String: @NotNull passes, @Size passes.
	status, _ := perform(t, server, http.MethodPost, "/api/persons", "application/json",
		fmtBody(base, `""`))
	if status != http.StatusOK {
		t.Errorf(`"name":"" gave %d, want 200`, status)
	}
	// An explicit null is the same as omitting the key.
	status, body := perform(t, server, http.MethodPost, "/api/persons", "application/json",
		fmtBody(base, "null"))
	if status != http.StatusBadRequest || body != `{"name":"name 不能为空"}` {
		t.Errorf(`"name":null gave %d %s`, status, body)
	}
}

func TestSaveRejectsAnUnsupportedContentType(t *testing.T) {
	server := mockMvc(t)

	status, body := perform(t, server, http.MethodPost, "/api/persons", "text/plain", "{}")

	if status != http.StatusUnsupportedMediaType {
		t.Errorf("status = %d, want 415", status)
	}
	if body == "" {
		t.Error("415 carried no error body")
	}
	// A vendor +json type is consumable.
	if status, _ = perform(t, server, http.MethodPost, "/api/persons", "application/vnd.api+json",
		`{"classId":"1","name":"n","sex":"Man","region":"China","phoneNumber":"18163138155"}`); status != http.StatusOK {
		t.Errorf("application/vnd.api+json gave %d, want 200", status)
	}
}

func TestSaveRejectsAnUnreadableBody(t *testing.T) {
	server := mockMvc(t)
	for _, body := range []string{"{", "null", "[]", `"x"`, ""} {
		status, _ := perform(t, server, http.MethodPost, "/api/persons", "application/json", body)
		if status != http.StatusBadRequest {
			t.Errorf("body %q gave %d, want 400", body, status)
		}
	}
}

func TestGetPersonByIDAppliesTheMaxConstraintOnlyFromAbove(t *testing.T) {
	server := mockMvc(t)
	for _, testCase := range []struct {
		path   string
		status int
		body   string
	}{
		{"/api/persons/5", http.StatusOK, "5"},
		{"/api/persons/0", http.StatusOK, "0"},
		{"/api/persons/-3", http.StatusOK, "-3"},
		{"/api/persons/0x5", http.StatusOK, "5"},
		{"/api/persons/6", http.StatusBadRequest, "getPersonByID.id: 超过 id 的范围了"},
		{"/api/persons/2147483647", http.StatusBadRequest, "getPersonByID.id: 超过 id 的范围了"},
	} {
		status, body := perform(t, server, http.MethodGet, testCase.path, "", "")
		if status != testCase.status || body != testCase.body {
			t.Errorf("GET %s = %d %q, want %d %q", testCase.path, status, body, testCase.status, testCase.body)
		}
	}
}

func TestGetPersonByIDReportsAConversionFailureBeforeValidation(t *testing.T) {
	server := mockMvc(t)
	// Out of Integer range never reaches @Max, so the answer is the framework's
	// error page rather than the constraint message.
	status, body := perform(t, server, http.MethodGet, "/api/persons/2147483648", "", "")
	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", status)
	}
	if body == "getPersonByID.id: 超过 id 的范围了" {
		t.Error("an unconvertible id produced the @Max message instead of the error page")
	}
}

func TestGetPersonByNameJoinsRepeatedParameters(t *testing.T) {
	server := mockMvc(t)
	// Spring converts the String[] the container supplies into the declared
	// String by joining with a comma, so two short values can exceed @Size(6).
	status, body := perform(t, server, http.MethodPut, "/api/persons?name=ab&name=cd", "", "")
	if status != http.StatusOK || body != "ab,cd" {
		t.Errorf("= %d %q, want 200 %q", status, body, "ab,cd")
	}
	status, body = perform(t, server, http.MethodPut, "/api/persons?name=abc&name=def", "", "")
	if status != http.StatusBadRequest || body != "getPersonByName.name: 超过 name 的范围了" {
		t.Errorf("= %d %q, want the @Size message", status, body)
	}
}

func TestGetPersonByNameRequiresTheParameterButAllowsItEmpty(t *testing.T) {
	server := mockMvc(t)
	status, body := perform(t, server, http.MethodPut, "/api/persons?name=", "", "")
	if status != http.StatusOK || body != "" {
		t.Errorf("empty name = %d %q, want 200 and an empty body", status, body)
	}
	status, _ = perform(t, server, http.MethodPut, "/api/persons", "", "")
	if status != http.StatusBadRequest {
		t.Errorf("missing name = %d, want 400", status)
	}
}

func TestPhoneNumberValidatorAcceptsNullAndTheDocumentedPrefixes(t *testing.T) {
	if !validation.IsValidPhoneNumber(nil) {
		t.Error("a null phone number was rejected; @NotNull is what rejects absence")
	}
	for _, number := range []string{
		"13000000000", "14500000000", "15000000000", "15500000000",
		"16500000000", "16600000000", "17000000000", "18163138155",
		"19100000000", "19800000000", "19900000000",
	} {
		if !validation.IsValidPhoneNumber(&number) {
			t.Errorf("rejected %q", number)
		}
	}
	for _, number := range []string{
		"", "1816313815", "181631381555", "14400000000", "15400000000",
		"16700000000", "19200000000", "23000000000", "18163138155\n", "abcdefghijk",
	} {
		if validation.IsValidPhoneNumber(&number) {
			t.Errorf("accepted %q", number)
		}
	}
}

func TestRegionValidatorRejectsNull(t *testing.T) {
	// RegionValidator asks a HashSet whether it contains the value and never
	// null-guards, so an absent region fails even without a @NotNull.
	if validation.IsValidRegion(nil) {
		t.Error("a null region was accepted")
	}
	for _, region := range []string{"China", "China-Taiwan", "China-HongKong"} {
		if !validation.IsValidRegion(&region) {
			t.Errorf("rejected %q", region)
		}
	}
	for _, region := range []string{"", "china", "Shanghai", "China-Macau"} {
		if validation.IsValidRegion(&region) {
			t.Errorf("accepted %q", region)
		}
	}
}

func TestPersonBuilderAndAccessors(t *testing.T) {
	built := entity.NewPersonRequestBuilder().
		ClassID("c").Name("n").Sex("Man").Region("China").PhoneNumber("18163138155").Build()
	if built.Name == nil || *built.Name != "n" {
		t.Errorf("Name = %v, want \"n\"", built.Name)
	}
	if got := beanvalidation.Validate(built); len(got) != 0 {
		t.Errorf("a fully built request had violations: %v", got)
	}

	person := &entity.Person{}
	if person.GetGroup() != nil {
		t.Error("a fresh Person has a group")
	}
	person.SetGroup("group1")
	if person.GetGroup() == nil || *person.GetGroup() != "group1" {
		t.Errorf("GetGroup = %v, want \"group1\"", person.GetGroup())
	}
}

func TestResolvePortPrefersTheArgumentThenTheEnvironment(t *testing.T) {
	if got := resolvePort(nil, ""); got != defaultPort {
		t.Errorf("resolvePort = %q, want %q", got, defaultPort)
	}
	if got := resolvePort(nil, "9000"); got != "9000" {
		t.Errorf("resolvePort = %q, want %q", got, "9000")
	}
	if got := resolvePort([]string{"--server.port=9001"}, "9000"); got != "9001" {
		t.Errorf("resolvePort = %q, want %q", got, "9001")
	}
	if got := resolvePort([]string{"--other", "--server.port="}, "9000"); got != "9000" {
		t.Errorf("an empty --server.port was taken as a port: %q", got)
	}
}

// fmtBody substitutes one %s placeholder without pulling fmt into every caller.
func fmtBody(template, value string) string {
	for i := 0; i+1 < len(template); i++ {
		if template[i] == '%' && template[i+1] == 's' {
			return template[:i] + value + template[i+2:]
		}
	}
	return template
}

// rawResponse serves one request straight through the application's router with
// a recorder, so the assertions see headers exactly as written — an http.Client
// consumes hop-by-hop headers such as Connection before a caller can inspect them.
func rawResponse(t *testing.T, method, target, contentType, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	request := httptest.NewRequest(method, target, reader)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	recorder := httptest.NewRecorder()
	newRouter().ServeHTTP(recorder, request)
	return recorder.Result()
}

func TestValidationFailuresMarkTheConnectionForClose(t *testing.T) {
	// Tomcat closes a connection whose request body it never finished reading,
	// so every validation failure carries `Connection: close` while the 200 that
	// consumed its body does not. Asserted through the real advice chain: a
	// handler that forgot to mark the response would pass every other test.
	for _, testCase := range []struct {
		name, method, target, ctype, body string
		wantClose                         bool
	}{
		{"field validation failure", http.MethodPost, "/api/persons", "application/json", `{}`, true},
		{"method-level violation", http.MethodGet, "/api/persons/6", "", "", true},
		{"request-param violation", http.MethodPut, "/api/persons?name=snailclimbsnailclimb", "", "", true},
		{"successful post", http.MethodPost, "/api/persons", "application/json",
			`{"classId":"1","name":"n","sex":"Man","region":"China","phoneNumber":"18163138155"}`, false},
		{"successful get", http.MethodGet, "/api/hello", "", "", false},
	} {
		got := rawResponse(t, testCase.method, testCase.target, testCase.ctype, testCase.body).
			Header.Get("Connection")
		if testCase.wantClose && got != "close" {
			t.Errorf("%s: Connection = %q, want %q", testCase.name, got, "close")
		}
		if !testCase.wantClose && got != "" {
			t.Errorf("%s: Connection = %q, want none", testCase.name, got)
		}
	}
}

func TestResponseContentTypesMatchTheOriginal(t *testing.T) {
	for _, testCase := range []struct {
		name, method, target, ctype, body, want string
	}{
		{"hello", http.MethodGet, "/api/hello", "", "", "text/plain;charset=UTF-8"},
		{"echoed person", http.MethodPost, "/api/persons", "application/json",
			`{"classId":"1","name":"n","sex":"Man","region":"China","phoneNumber":"18163138155"}`,
			"application/json"},
		{"field errors", http.MethodPost, "/api/persons", "application/json", `{}`, "application/json"},
		{"violation message", http.MethodGet, "/api/persons/6", "", "", "text/plain;charset=UTF-8"},
		{"framework error page", http.MethodGet, "/api/nope", "", "", "application/json"},
	} {
		got := rawResponse(t, testCase.method, testCase.target, testCase.ctype, testCase.body).
			Header.Get("Content-Type")
		if got != testCase.want {
			t.Errorf("%s: Content-Type = %q, want %q", testCase.name, got, testCase.want)
		}
	}
}
