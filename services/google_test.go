package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"eventexplorer/models"
	"eventexplorer/utils"
)

func newTestGoogle(handler http.HandlerFunc) (*GoogleService, *httptest.Server) {
	server := httptest.NewServer(handler)
	svc := NewGoogleServiceWith(utils.NewClient(2*time.Second), "test-key", server.URL)
	return svc, server
}

func TestAutocompletePostsTheRequiredBodyAndHeader(t *testing.T) {
	var body models.AutocompleteRequest
	var apiKeyHeader string

	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		apiKeyHeader = r.Header.Get("X-Goog-Api-Key")
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		w.Write([]byte(`{"suggestions":[]}`))
	})
	defer server.Close()

	if _, err := svc.Autocomplete(context.Background(), "toronto", "session-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if apiKeyHeader != "test-key" {
		t.Error("the key must travel in the X-Goog-Api-Key header, not the query")
	}
	if body.Input != "toronto" {
		t.Errorf("input not sent: %q", body.Input)
	}
	if body.SessionToken != "session-1" {
		t.Errorf("session token not sent: %q", body.SessionToken)
	}
	if len(body.IncludedPrimaryTypes) != 1 || body.IncludedPrimaryTypes[0] != models.CitiesPrimaryType {
		t.Errorf("the cities collection is required, got %v", body.IncludedPrimaryTypes)
	}
}

func TestAutocompleteMapsSuggestions(t *testing.T) {
	body := `{"suggestions":[{"placePrediction":{
		"placeId":"ChIJ123",
		"text":{"text":"Toronto, ON, Canada"},
		"structuredFormat":{"mainText":{"text":"Toronto"},"secondaryText":{"text":"ON, Canada"}}
	}}]}`

	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})
	defer server.Close()

	got, err := svc.Autocomplete(context.Background(), "toronto", "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].PlaceID != "ChIJ123" || got[0].MainText != "Toronto" {
		t.Fatalf("unexpected suggestions: %+v", got)
	}
}

func TestAutocompleteRequiresAMinimumInputLength(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Google should not be called for a short input")
	})
	defer server.Close()

	for _, input := range []string{"", " ", "to", "  a "} {
		if _, err := svc.Autocomplete(context.Background(), input, "s1"); !errors.Is(err, models.ErrInvalidInput) {
			t.Errorf("%q: expected ErrInvalidInput, got %v", input, err)
		}
	}
}

func TestAutocompleteHandlesProviderFailure(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"code":403,"status":"PERMISSION_DENIED","message":"billing"}}`))
	})
	defer server.Close()

	if _, err := svc.Autocomplete(context.Background(), "toronto", "s1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestAutocompleteHandlesMalformedJSON(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"suggestions": broken`))
	})
	defer server.Close()

	if _, err := svc.Autocomplete(context.Background(), "toronto", "s1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestAutocompleteRequiresAnAPIKey(t *testing.T) {
	svc := NewGoogleServiceWith(utils.NewClient(time.Second), "", "https://example.test")

	if _, err := svc.Autocomplete(context.Background(), "toronto", "s1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable without a key, got %v", err)
	}
}

func TestPlaceDetailsSendsTheFieldMaskAndSessionToken(t *testing.T) {
	var got *http.Request

	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		got = r
		w.Write([]byte(`{"addressComponents":[
			{"longText":"Toronto","types":["locality"]},
			{"longText":"Canada","shortText":"CA","types":["country"]}
		]}`))
	})
	defer server.Close()

	city, err := svc.PlaceDetails(context.Background(), "ChIJ123", "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Header.Get("X-Goog-FieldMask") != "addressComponents" {
		t.Errorf("the field mask is required, got %q", got.Header.Get("X-Goog-FieldMask"))
	}
	if got.URL.Query().Get("sessionToken") != "session-1" {
		t.Error("the session token must be reused for the selection")
	}
	if got.URL.Path != "/v1/places/ChIJ123" {
		t.Errorf("unexpected path: %q", got.URL.Path)
	}
	if city.City != "Toronto" || city.CountryCode != "CA" {
		t.Errorf("unexpected city: %+v", city)
	}
}

func TestPlaceDetailsWorksWithoutASessionToken(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("sessionToken") {
			t.Error("no session token should be sent when none was given")
		}
		w.Write([]byte(`{"addressComponents":[
			{"longText":"Dhaka","types":["locality"]},
			{"longText":"Bangladesh","shortText":"BD","types":["country"]}
		]}`))
	})
	defer server.Close()

	if _, err := svc.PlaceDetails(context.Background(), "place-x", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceDetailsRejectsAnEmptyPlaceID(t *testing.T) {
	svc := NewGoogleServiceWith(utils.NewClient(time.Second), "k", "https://example.test")

	if _, err := svc.PlaceDetails(context.Background(), "  ", "s1"); !errors.Is(err, models.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPlaceDetailsNotFound(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":{"code":404,"status":"NOT_FOUND"}}`))
	})
	defer server.Close()

	if _, err := svc.PlaceDetails(context.Background(), "nope", "s1"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPlaceDetailsHandlesProviderFailure(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":{"status":"PERMISSION_DENIED"}}`))
	})
	defer server.Close()

	if _, err := svc.PlaceDetails(context.Background(), "p1", "s1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestPlaceDetailsHandlesMalformedJSON(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"addressComponents": broken`))
	})
	defer server.Close()

	if _, err := svc.PlaceDetails(context.Background(), "p1", "s1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestPlaceDetailsWithNoCityComponent(t *testing.T) {
	svc, server := newTestGoogle(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"addressComponents":[{"longText":"Canada","shortText":"CA","types":["country"]}]}`))
	})
	defer server.Close()

	if _, err := svc.PlaceDetails(context.Background(), "p1", "s1"); !errors.Is(err, models.ErrNoCityComponent) {
		t.Fatalf("expected ErrNoCityComponent, got %v", err)
	}
}

func TestPlaceDetailsRequiresAnAPIKey(t *testing.T) {
	svc := NewGoogleServiceWith(utils.NewClient(time.Second), "", "https://example.test")

	if _, err := svc.PlaceDetails(context.Background(), "p1", "s1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable without a key, got %v", err)
	}
}

func TestNewSessionTokenIsUniqueAndNonEmpty(t *testing.T) {
	first := NewSessionToken()
	second := NewSessionToken()

	if first == "" || second == "" {
		t.Fatal("a session token must not be empty")
	}
	if first == second {
		t.Error("each search should start a new session token")
	}
}

func TestGoogleErrorMessageFallsBack(t *testing.T) {
	if got := googleErrorMessage(utils.Response{Body: []byte("not json")}); got == "" {
		t.Error("the message must never be empty")
	}
}

func TestGoogleHeadersMergeExtras(t *testing.T) {
	svc := NewGoogleServiceWith(utils.NewClient(time.Second), "k", "https://example.test")

	h := svc.headers(map[string]string{"X-Extra": "v"})
	if h["X-Goog-Api-Key"] != "k" || h["X-Extra"] != "v" {
		t.Errorf("unexpected headers: %v", h)
	}
}
