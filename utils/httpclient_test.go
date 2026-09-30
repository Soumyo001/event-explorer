package utils

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type stubDoer struct {
	response *http.Response
	err      error
	request  *http.Request
}

func (s *stubDoer) Do(req *http.Request) (*http.Response, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return s.response, nil
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestGetReturnsStatusAndBody(t *testing.T) {
	doer := &stubDoer{response: jsonResponse(200, `{"ok":true}`)}
	client := NewClientWithDoer(doer, time.Second)

	res, err := client.Get(context.Background(), "https://example.test/data", map[string]string{
		"X-Custom": "value",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.OK() || res.StatusCode != 200 {
		t.Errorf("unexpected status: %d", res.StatusCode)
	}
	if string(res.Body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", res.Body)
	}
	if doer.request.Header.Get("X-Custom") != "value" {
		t.Error("custom headers should reach the request")
	}
	if doer.request.Header.Get("Accept") != "application/json" {
		t.Error("Accept should default to application/json")
	}
}

func TestPostJSONSendsTheEncodedBody(t *testing.T) {
	doer := &stubDoer{response: jsonResponse(200, `{}`)}
	client := NewClientWithDoer(doer, time.Second)

	payload := map[string]string{"input": "toronto"}
	if _, err := client.PostJSON(context.Background(), "https://example.test/x", nil, payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doer.request.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", doer.request.Method)
	}
	if doer.request.Header.Get("Content-Type") != "application/json" {
		t.Error("PostJSON must set the JSON content type")
	}

	sent, _ := io.ReadAll(doer.request.Body)
	if !strings.Contains(string(sent), `"input":"toronto"`) {
		t.Errorf("unexpected request body: %s", sent)
	}
}

func TestPostJSONRejectsUnencodablePayloads(t *testing.T) {
	client := NewClientWithDoer(&stubDoer{response: jsonResponse(200, `{}`)}, time.Second)

	// channel cannot be marshalled to JSON.
	_, err := client.PostJSON(context.Background(), "https://example.test/x", nil, make(chan int))
	if err == nil {
		t.Fatal("expected an encoding error")
	}
}

func TestNonSuccessStatusIsReturnedNotErrored(t *testing.T) {
	doer := &stubDoer{response: jsonResponse(403, `{"error":{"status":"PERMISSION_DENIED"}}`)}
	client := NewClientWithDoer(doer, time.Second)

	res, err := client.Get(context.Background(), "https://example.test/x", nil)
	if err != nil {
		t.Fatalf("a 403 should not be a transport error, got %v", err)
	}
	if res.OK() {
		t.Error("403 must not report OK")
	}
	if !strings.Contains(string(res.Body), "PERMISSION_DENIED") {
		t.Error("the error body should be readable by the caller")
	}
}

func TestTransportErrorsAreSanitised(t *testing.T) {
	doer := &stubDoer{err: &url.Error{
		Op:  "Get",
		URL: "https://app.ticketmaster.com/discovery/v2/events.json?apikey=SUPERSECRET",
		Err: errors.New("connection refused"),
	}}
	client := NewClientWithDoer(doer, time.Second)

	_, err := client.Get(context.Background(), "https://example.test/x", nil)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "SUPERSECRET") {
		t.Fatalf("the api key leaked into the error: %v", err)
	}
	if !strings.Contains(err.Error(), "REDACTED") {
		t.Errorf("expected the url to be redacted, got %v", err)
	}
}

func TestOversizedResponsesAreRejected(t *testing.T) {
	oversized := strings.Repeat("a", maxResponseBytes+10)
	doer := &stubDoer{response: jsonResponse(200, oversized)}
	client := NewClientWithDoer(doer, time.Second)

	if _, err := client.Get(context.Background(), "https://example.test/x", nil); err == nil {
		t.Fatal("expected an oversized response to be rejected rather than truncated")
	}
}

func TestRequestBuildFailureIsReported(t *testing.T) {
	client := NewClientWithDoer(&stubDoer{response: jsonResponse(200, `{}`)}, time.Second)
	if _, err := client.Get(context.Background(), "https://example.test/\x7f", nil); err == nil {
		t.Fatal("expected a request build error")
	}
}

func TestTimeoutsAreEnforced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer server.Close()

	client := NewClient(30 * time.Millisecond)

	_, err := client.Get(context.Background(), server.URL, nil)
	if err == nil {
		t.Fatal("expected the request to time out")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("the error chain should still show a deadline: %v", err)
	}
}

func TestNewClientFallsBackToADefaultTimeout(t *testing.T) {
	if c := NewClient(0); c.timeout != 10*time.Second {
		t.Errorf("expected the default timeout, got %s", c.timeout)
	}
	if c := NewClientWithDoer(nil, -time.Second); c.timeout != 10*time.Second {
		t.Errorf("expected the default timeout, got %s", c.timeout)
	}
}

func TestDecodeJSON(t *testing.T) {
	var out struct {
		Name string `json:"name"`
	}

	res := Response{StatusCode: 200, Body: []byte(`{"name":"Toronto"}`)}
	if err := res.DecodeJSON(&out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "Toronto" {
		t.Errorf("unexpected decode result: %+v", out)
	}

	if err := (Response{}).DecodeJSON(&out); err == nil {
		t.Error("an empty body should be an error")
	}
	if err := (Response{Body: []byte("not json")}).DecodeJSON(&out); err == nil {
		t.Error("malformed json should be an error")
	}
}

func TestResponseOKBoundaries(t *testing.T) {
	cases := map[int]bool{199: false, 200: true, 204: true, 299: true, 300: false, 500: false}
	for status, want := range cases {
		if got := (Response{StatusCode: status}).OK(); got != want {
			t.Errorf("status %d: OK() = %v, want %v", status, got, want)
		}
	}
}
