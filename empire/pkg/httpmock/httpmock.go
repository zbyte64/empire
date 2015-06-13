package httpmock

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

// ServeReplay is an http.Handler
// It contains a list of handlers and calls the next handler in the list for each incoming request.
//
// If a request is received and no more handlers are available, ServeReplay calls NoneLeft.
// The default behavior of NoneLeft is to call t.Errorf with some information about the request.
type ServeReplay struct {
	t        *testing.T
	i        int
	Handlers []http.Handler
	NoneLeft func(*testing.T, *http.Request)
}

func defaultNoneLeft(t *testing.T, r *http.Request) {
	t.Errorf("http request: %s %s; no more handlers to call", r.Method, r.URL.Path)
}

// NewServeReplay returns a new ServeReplay.
func NewServeReplay(t *testing.T) *ServeReplay {
	return &ServeReplay{
		t:        t,
		Handlers: make([]http.Handler, 0),
		NoneLeft: defaultNoneLeft,
	}
}

// Add appends a handler to ServReplay's handler list.
// It returns itself to allow chaining.
func (h *ServeReplay) Add(handler http.Handler) *ServeReplay {
	h.Handlers = append(h.Handlers, handler)
	return h
}

// ServeHTTP dispatches the request to the next handler in the list.
func (h *ServeReplay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.i >= len(h.Handlers) {
		h.NoneLeft(h.t, r)
	} else {
		h.Handlers[h.i].ServeHTTP(w, r)
		h.i++
	}
}

// PathHandler will fail if the request doesn't match based on reqStr.
// If it matches it returns the given response status and body.
//
// reqStr should be of the form `METHOD /path Header:Value Header:Value`.
func PathHandler(t *testing.T, reqStr string, body *string, respStatus int, respBody string) http.Handler {
	method, uri, headers, err := parseReq(reqStr)
	if err != nil {
		t.Fatal(err)
	}

	return RequestHandler(t, method, uri, headers, body, respStatus, respBody)
}

// RequestHandler will fail if the request doesn't match on the given method, URI, headers, and body
func RequestHandler(t *testing.T, method string, uri string, headers map[string]string, body *string, respStatus int, respBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		match := true

		// Match method
		if method != r.Method {
			match = false
		}

		// Match request URI
		if uri != r.URL.RequestURI() {
			match = false
		}

		// Match headers
		for k, v := range headers {
			if r.Header.Get(k) != v {
				match = false
			}
		}

		// Match body
		var actualReqBody []byte
		if body != nil {
			var err error
			if actualReqBody, err = ioutil.ReadAll(r.Body); err != nil {
				t.Fatal(err)
			}

			if string(actualReqBody) != *body {
				match = false
			}
		}

		if match {
			w.WriteHeader(respStatus)
			w.Write([]byte(respBody))
		} else {
			http.NotFoundHandler().ServeHTTP(w, r)
			t.Errorf("http request did not match.\ngot:\n%s\nwanted:\n%s", formatReq(r, string(actualReqBody)), formatHTTPReq(method, uri, headers, body))
		}
	})
}

func formatHTTPReq(method string, uri string, headers map[string]string, body *string) string {
	s := make([]string, 0)
	s = append(s, fmt.Sprintf("%s %s", method, uri))

	for k, v := range headers {
		s = append(s, fmt.Sprintf("%s: %s", k, v))
	}
	s = append(s, "")

	if body != nil {
		s = append(s, *body)
	}

	return strings.Join(s, "\n")
}

func formatReq(r *http.Request, body string) string {
	headers := make(map[string]string, 0)
	for k, _ := range r.Header {
		headers[k] = r.Header.Get(k)
	}
	return formatHTTPReq(r.Method, r.URL.RequestURI(), headers, &body)
}

// parseReq parses method, uri and headers from a string formatted in the following way:
//
// "METHOD /path Header:Value Header:Value"
func parseReq(s string) (meth string, uri string, headers map[string]string, err error) {
	headers = make(map[string]string, 0)

	parts := strings.Split(s, " ")

	if len(parts) < 2 {
		return meth, uri, headers, errors.New("invalid request string")
	}

	meth = parts[0]
	uri = parts[1]

	for _, kvp := range parts[2:] {
		h := strings.Split(kvp, ":")

		if len(h) != 2 {
			return meth, uri, headers, errors.New("invalid header string")
		}
		headers[h[0]] = h[1]
	}

	return meth, uri, headers, nil
}
