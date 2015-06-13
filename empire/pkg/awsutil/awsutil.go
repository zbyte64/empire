package awsutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/remind101/empire/empire/pkg/httpmock"
)

// Request represents an expected AWS API Operation.
type Request struct {
	RequestURI string
	Operation  string
	Body       string
}

func (r *Request) String() string {
	body := formatBody(strings.NewReader(r.Body))
	return fmt.Sprintf("RequestURI: %s\nOperation: %s\nBody: %s", r.RequestURI, r.Operation, body)
}

// Response represents a predefined response.
type Response struct {
	StatusCode int
	Body       string
}

// Cycle represents a request-response cycle.
type Cycle struct {
	Request  Request
	Response Response
}

// NewHandler returns a new Handler instance.
func NewHandler(t *testing.T, c []Cycle) http.Handler {
	m := httpmock.NewServeReplay(t)

	for _, cycle := range c {
		m.Add(&cycleHandler{t: t, cycle: cycle})
	}

	return m
}

// Handler is an http.Handler that will play back a cycle.
type cycleHandler struct {
	t     *testing.T
	cycle Cycle
}

func (h *cycleHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.t.Fatal(err)
	}

	match := Request{
		RequestURI: r.URL.RequestURI(),
		Operation:  r.Header.Get("X-Amz-Target"),
		Body:       string(b),
	}

	if h.cycle.Request.Body == "ignore" {
		match.Body = h.cycle.Request.Body
	}

	if h.cycle.Request.String() == match.String() {
		w.WriteHeader(h.cycle.Response.StatusCode)
		io.WriteString(w, h.cycle.Response.Body)
	} else {
		w.WriteHeader(404)
		h.t.Log("Request does not match next cycle.")
		h.t.Log(h.cycle.Request.String())
		h.t.Log(match.String())
		h.t.Fail()
	}
}

func formatBody(r io.Reader) string {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		panic(err)
	}

	s, err := formatJSON(bytes.NewReader(b))
	if err == nil {
		return s
	}

	return string(b)
}

func formatJSON(r io.Reader) (string, error) {
	var body map[string]interface{}
	if err := json.NewDecoder(r).Decode(&body); err != nil {
		return "", err
	}

	raw, err := json.MarshalIndent(&body, "", "  ")
	if err != nil {
		return "", err
	}

	return string(raw), nil
}
