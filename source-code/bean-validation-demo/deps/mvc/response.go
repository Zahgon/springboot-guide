// Package mvc reproduces the slice of Spring MVC this application's HTTP
// surface depends on: request mapping and the errors a mismatch produces,
// @RequestBody binding, ResponseEntity, and Spring Boot's default error body.
//
// Everything here is observable at the socket. The status codes and bodies are
// the obvious part; the framing is not, and it is just as visible: Spring
// writes a String body with a Content-Length and streams a JSON body with
// chunked transfer encoding, and Tomcat closes a connection whose request body
// it never finished reading.
package mvc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

// Content types as Spring spells them, including the charset parameter it
// appends to text/plain and omits from application/json.
const (
	TextPlainUTF8   = "text/plain;charset=UTF-8"
	ApplicationJSON = "application/json"
)

// Body is a rendered response body together with the framing Spring uses for
// its kind.
type Body struct {
	contentType string
	bytes       []byte
	// chunked reports whether Spring streams this body rather than buffering
	// it. A String return value goes out with a Content-Length; a Jackson-
	// serialised object goes out chunked, because the converter writes into the
	// response stream without knowing the length in advance.
	chunked bool
}

// Text is a String return value: text/plain with a charset, and a Content-Length.
func Text(text string) Body {
	return Body{contentType: TextPlainUTF8, bytes: []byte(text)}
}

// NoBody is a response that carries no entity at all — no content type, and a
// Content-Length of zero. Spring's OPTIONS handler produces one.
func NoBody() Body { return Body{} }

// JSON is an object return value serialised by Jackson: application/json,
// streamed with chunked transfer encoding.
func JSON(value any) Body {
	encoded, err := marshal(value)
	if err != nil {
		// Jackson failing mid-stream produces a 500 from an already-committed
		// response. Nothing this application returns can fail to serialise, so
		// this stays a programming error rather than a response shape.
		panic("mvc: cannot serialise response body: " + err.Error())
	}
	return Body{contentType: ApplicationJSON, bytes: encoded, chunked: true}
}

// marshal encodes value the way Jackson does: compact, and without escaping
// `<`, `>` and `&`, which Go's encoding/json rewrites to < and friends by
// default and Jackson leaves alone.
func marshal(value any) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	// Encode appends a newline that Jackson does not write.
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

// ResponseEntity mirrors org.springframework.http.ResponseEntity.
type ResponseEntity struct {
	Status  int
	Headers http.Header
	Body    Body
	// closeConnection sets `Connection: close`, which Tomcat does when it is
	// about to reply without having drained the request body.
	closeConnection bool
}

// OK is ResponseEntity.ok().body(...).
func OK(body Body) ResponseEntity {
	return ResponseEntity{Status: http.StatusOK, Body: body}
}

// Status is ResponseEntity.status(...).body(...).
func Status(status int, body Body) ResponseEntity {
	return ResponseEntity{Status: status, Body: body}
}

// WithHeader returns a copy carrying an additional response header.
func (e ResponseEntity) WithHeader(name string, values ...string) ResponseEntity {
	headers := http.Header{}
	for key, existing := range e.Headers {
		headers[key] = append([]string(nil), existing...)
	}
	for _, value := range values {
		headers.Add(name, value)
	}
	e.Headers = headers
	return e
}

// Closing returns a copy that will send `Connection: close`. Tomcat sets it
// when it replies to a request whose body it has not finished reading, which is
// every validation failure on a POST and every rejected path variable.
func (e ResponseEntity) Closing() ResponseEntity {
	e.closeConnection = true
	return e
}

// write emits the entity, choosing the framing the body's kind implies.
func (e ResponseEntity) write(writer http.ResponseWriter) {
	header := writer.Header()
	for key, values := range e.Headers {
		for _, value := range values {
			header.Add(key, value)
		}
	}
	if e.Body.contentType != "" {
		header.Set("Content-Type", e.Body.contentType)
	}
	if e.closeConnection {
		header.Set("Connection", "close")
	}
	if e.Body.chunked {
		// Go decides the framing itself unless told; naming the encoding is how
		// a handler asks for the chunked stream Spring produces here.
		header.Set("Transfer-Encoding", "chunked")
	} else {
		header.Set("Content-Length", strconv.Itoa(len(e.Body.bytes)))
	}
	writer.WriteHeader(e.Status)
	if len(e.Body.bytes) > 0 {
		_, _ = writer.Write(e.Body.bytes)
	}
}
