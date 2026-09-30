package utils

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestRedactURLHidesCredentials(t *testing.T) {
	raw := "https://app.ticketmaster.com/discovery/v2/events.json?apikey=SECRET&city=Toronto"

	got := RedactURL(raw)
	if strings.Contains(got, "SECRET") {
		t.Fatalf("the key survived redaction: %s", got)
	}
	if !strings.Contains(got, "REDACTED") {
		t.Errorf("expected a REDACTED marker, got %s", got)
	}
	if !strings.Contains(got, "city=Toronto") {
		t.Errorf("harmless parameters should be preserved, got %s", got)
	}
}

func TestRedactURLCoversEverySensitiveParameter(t *testing.T) {
	raw := "https://example.test/x?apikey=a&key=b&api_key=c&sessionToken=d"

	got := RedactURL(raw)
	for _, secret := range []string{"=a", "=b", "=c", "=d"} {
		if strings.Contains(got, secret) {
			t.Errorf("a sensitive value survived: %s", got)
		}
	}
}

func TestRedactURLHandlesGarbage(t *testing.T) {
	if got := RedactURL("://not a url"); got != "invalid-url" {
		t.Errorf("expected the safe placeholder, got %q", got)
	}
}

func TestRedactURLLeavesCleanURLsUsable(t *testing.T) {
	raw := "https://places.googleapis.com/v1/places:autocomplete"
	if got := RedactURL(raw); got != raw {
		t.Errorf("a url with no secrets should be unchanged, got %q", got)
	}
}

func TestSanitizeErrorRebuildsURLErrors(t *testing.T) {
	original := &url.Error{
		Op:  "Get",
		URL: "https://app.ticketmaster.com/x?apikey=SECRET",
		Err: context.DeadlineExceeded,
	}

	got := SanitizeError(original)
	if strings.Contains(got.Error(), "SECRET") {
		t.Fatalf("the key leaked: %v", got)
	}

	if !errors.Is(got, context.DeadlineExceeded) {
		t.Error("the wrapped error should still be reachable with errors.Is")
	}

	var urlErr *url.Error
	if !errors.As(got, &urlErr) {
		t.Fatal("the result should still be a *url.Error")
	}
	if urlErr.Op != "Get" {
		t.Errorf("the operation should be preserved, got %q", urlErr.Op)
	}
}

func TestSanitizeErrorPassesOtherErrorsThrough(t *testing.T) {
	plain := errors.New("something else went wrong")
	if got := SanitizeError(plain); got != plain {
		t.Error("a non-url error should be returned unchanged")
	}
	if SanitizeError(nil) != nil {
		t.Error("nil should stay nil")
	}
}
