package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"eventexplorer/models"
	"eventexplorer/utils"
)

func newTestTicketmaster(handler http.HandlerFunc) (*TicketmasterService, *httptest.Server) {
	server := httptest.NewServer(handler)
	svc := NewTicketmasterServiceWith(
		utils.NewClient(2*time.Second), "test-key", server.URL)
	return svc, server
}

func TestFetchEventsSendsTheRequiredQueryParameters(t *testing.T) {
	var got *http.Request

	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Write([]byte(`{"_embedded":{"events":[]},"page":{"totalElements":0}}`))
	})
	defer server.Close()

	if _, err := svc.FetchEvents(context.Background(), "Toronto", "ca", models.CategoryMusic, 6); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := got.URL.Query()
	if q.Get("city") != "Toronto" {
		t.Errorf("city not sent: %q", q.Get("city"))
	}
	if q.Get("countryCode") != "CA" {
		t.Errorf("the country code should be uppercased, got %q", q.Get("countryCode"))
	}
	if q.Get("classificationName") != "Music" {
		t.Errorf("classificationName not sent: %q", q.Get("classificationName"))
	}
	if q.Get("size") != "6" {
		t.Errorf("the assignment requires size=6, got %q", q.Get("size"))
	}
	if q.Get("apikey") != "test-key" {
		t.Error("the api key should travel as a query parameter")
	}
	if got.URL.Path != "/events.json" {
		t.Errorf("unexpected path: %q", got.URL.Path)
	}
}

func TestFetchEventsDecodesEvents(t *testing.T) {
	body := `{"_embedded":{"events":[
		{"id":"e1","name":"Jazz Night","url":"https://www.ticketmaster.com/e1"},
		{"id":"e2","name":"Rock Show","url":"https://www.ticketmaster.com/e2"}
	]},"page":{"totalElements":2}}`

	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	defer server.Close()

	events, err := svc.FetchEvents(context.Background(), "Toronto", "CA", models.CategoryMusic, 6)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 || events[0].ID != "e1" {
		t.Fatalf("unexpected events: %+v", events)
	}
}

func TestFetchEventsEmptyResultIsNotAnError(t *testing.T) {
	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"page":{"totalElements":0}}`))
	})
	defer server.Close()

	events, err := svc.FetchEvents(context.Background(), "Dhaka", "BD", models.CategoryMusic, 6)
	if err != nil {
		t.Fatalf("an empty result should not error, got %v", err)
	}
	if events == nil {
		t.Error("expected an empty slice rather than nil")
	}
	if len(events) != 0 {
		t.Errorf("expected no events, got %d", len(events))
	}
}

func TestFetchEventsRejectsBadInput(t *testing.T) {
	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the provider should not be called for invalid input")
	})
	defer server.Close()

	cases := map[string]struct {
		city, country string
		category      models.Category
	}{
		"no city":          {"", "CA", models.CategoryMusic},
		"no country":       {"Toronto", "", models.CategoryMusic},
		"unknown category": {"Toronto", "CA", models.Category("Theatre")},
	}

	for name, c := range cases {
		_, err := svc.FetchEvents(context.Background(), c.city, c.country, c.category, 6)
		if !errors.Is(err, models.ErrInvalidInput) {
			t.Errorf("%s: expected ErrInvalidInput, got %v", name, err)
		}
	}
}

func TestFetchEventsHandlesProviderFailure(t *testing.T) {
	cases := map[int]string{
		401: `{"fault":{"faultstring":"Invalid ApiKey"}}`,
		429: `{"errors":[{"detail":"Rate limit exceeded"}]}`,
		500: `{}`,
	}

	for status, body := range cases {
		svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(body))
		})

		_, err := svc.FetchEvents(context.Background(), "Toronto", "CA", models.CategoryMusic, 6)
		if !errors.Is(err, models.ErrUpstreamUnavailable) {
			t.Errorf("status %d: expected ErrUpstreamUnavailable, got %v", status, err)
		}
		server.Close()
	}
}

func TestFetchEventsHandlesMalformedJSON(t *testing.T) {
	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"_embedded": not json`))
	})
	defer server.Close()

	_, err := svc.FetchEvents(context.Background(), "Toronto", "CA", models.CategoryMusic, 6)
	if !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestFetchEventsRequiresAnAPIKey(t *testing.T) {
	svc := NewTicketmasterServiceWith(utils.NewClient(time.Second), "", "https://example.test")

	_, err := svc.FetchEvents(context.Background(), "Toronto", "CA", models.CategoryMusic, 6)
	if !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable without a key, got %v", err)
	}
}

func TestFetchEventBuildsTheRightPath(t *testing.T) {
	var got *http.Request

	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Write([]byte(`{"id":"ev-1","name":"Jazz"}`))
	})
	defer server.Close()

	if _, err := svc.FetchEvent(context.Background(), "ev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.URL.Path != "/events/ev-1.json" {
		t.Errorf("unexpected path: %q", got.URL.Path)
	}
	if got.URL.Query().Get("apikey") != "test-key" {
		t.Error("the api key should be sent")
	}
}

func TestFetchEventDecodesTheEvent(t *testing.T) {
	body := `{"id":"ev-1","name":"Jazz Night","url":"https://www.ticketmaster.com/ev-1",
		"dates":{"start":{"localDate":"2027-03-01","localTime":"20:00:00"}}}`

	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	defer server.Close()

	event, err := svc.FetchEvent(context.Background(), "ev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID != "ev-1" || event.Name != "Jazz Night" {
		t.Errorf("unexpected event: %+v", event)
	}
	if event.URL == "" {
		t.Error("the ticket url must be decoded, the redirect depends on it")
	}
}

func TestFetchEventInvalidIDReturnsNotFound(t *testing.T) {
	for _, status := range []int{400, 404} {
		svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			w.Write([]byte(`{"errors":[{"detail":"Resource not found"}]}`))
		})

		_, err := svc.FetchEvent(context.Background(), "does-not-exist")
		if !errors.Is(err, models.ErrNotFound) {
			t.Errorf("status %d: expected ErrNotFound, got %v", status, err)
		}
		server.Close()
	}
}

func TestFetchEventEmptyPayloadIsNotFound(t *testing.T) {
	svc, server := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	})
	defer server.Close()

	if _, err := svc.FetchEvent(context.Background(), "ev-1"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestFetchEventRejectsAnEmptyID(t *testing.T) {
	svc := NewTicketmasterServiceWith(utils.NewClient(time.Second), "k", "https://example.test")

	if _, err := svc.FetchEvent(context.Background(), "  "); !errors.Is(err, models.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestFetchEventRequiresAnAPIKey(t *testing.T) {
	svc := NewTicketmasterServiceWith(utils.NewClient(time.Second), "", "https://example.test")

	if _, err := svc.FetchEvent(context.Background(), "ev-1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestFetchEventHandlesProviderFailureAndBadJSON(t *testing.T) {
	failing, s1 := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
		w.Write([]byte(`{}`))
	})
	defer s1.Close()
	if _, err := failing.FetchEvent(context.Background(), "ev-1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable for a 503, got %v", err)
	}

	malformed, s2 := newTestTicketmaster(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id": broken`))
	})
	defer s2.Close()
	if _, err := malformed.FetchEvent(context.Background(), "ev-1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Errorf("expected ErrUpstreamUnavailable for bad json, got %v", err)
	}
}

func TestTicketmasterConstructorsTrimTheBaseURL(t *testing.T) {
	svc := NewTicketmasterServiceWith(utils.NewClient(time.Second), "k", "https://example.test/")
	if svc.baseURL != "https://example.test" {
		t.Errorf("the trailing slash should be trimmed, got %q", svc.baseURL)
	}
}

func TestTicketmasterErrorMessageFallsBack(t *testing.T) {
	unreadable := utils.Response{StatusCode: 500, Body: []byte("not json")}
	if got := ticketmasterErrorMessage(unreadable); got == "" {
		t.Error("the message must never be empty")
	}
}
