package controllers

import (
	"net/http"
	"testing"

	"eventexplorer/models"
)

func TestStatusForErrorMapsEveryDomainError(t *testing.T) {
	cases := map[string]struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		"invalid input": {models.ErrInvalidInput, http.StatusBadRequest, "invalid_input"},
		"not found":     {models.ErrNotFound, http.StatusNotFound, "not_found"},
		"no city":       {models.ErrNoCityComponent, http.StatusUnprocessableEntity, "no_city"},
		"bad ticket":    {models.ErrInvalidTicketURL, http.StatusBadGateway, "invalid_ticket_link"},
		"upstream":      {models.ErrUpstreamUnavailable, http.StatusBadGateway, "upstream_unavailable"},
	}

	for name, c := range cases {
		status, code, message := statusForError(c.err)
		if status != c.wantStatus {
			t.Errorf("%s: status = %d, want %d", name, status, c.wantStatus)
		}
		if code != c.wantCode {
			t.Errorf("%s: code = %q, want %q", name, code, c.wantCode)
		}
		if message == "" {
			t.Errorf("%s: the visitor-facing message must not be empty", name)
		}
	}
}

func TestStatusForErrorDefaultsSafely(t *testing.T) {
	status, code, message := statusForError(errUnknown{})

	if status != http.StatusBadGateway || code != "upstream_unavailable" {
		t.Errorf("unexpected default: %d %q", status, code)
	}
	if message == "" {
		t.Error("the default message must not be empty")
	}
}

type errUnknown struct{}

func (errUnknown) Error() string { return "something the app does not recognise" }

func TestHeadingPerStatus(t *testing.T) {
	cases := map[int]string{
		http.StatusBadRequest:          "Something was missing",
		http.StatusNotFound:            "Not found",
		http.StatusUnprocessableEntity: "City unavailable",
		http.StatusBadGateway:          "Service unavailable",
		http.StatusTeapot:              "Service unavailable",
	}

	for status, want := range cases {
		if got := heading(status); got != want {
			t.Errorf("status %d: heading = %q, want %q", status, got, want)
		}
	}
}
