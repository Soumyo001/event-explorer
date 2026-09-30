package services

import (
	"errors"
	"testing"

	"eventexplorer/models"
)

func newTestValidator() TicketLinkValidator {
	return NewTicketLinkValidator([]string{
		"ticketmaster.com",
		"ticketmaster.ca",
		"livenation.com",
	})
}

func TestValidateAcceptsApprovedHTTPSHosts(t *testing.T) {
	v := newTestValidator()

	cases := []string{
		"https://ticketmaster.com/event/123",
		"https://www.ticketmaster.com/event/123",
		"https://concerts.livenation.com/show/9",
		"https://ticketmaster.ca/event/abc?utm_source=api",
	}

	for _, raw := range cases {
		got, err := v.Validate(raw)
		if err != nil {
			t.Errorf("expected %q to be accepted, got %v", raw, err)
			continue
		}
		if got != raw {
			t.Errorf("validator changed the url: %q -> %q", raw, got)
		}
	}
}

func TestValidateRejectsUnapprovedHosts(t *testing.T) {
	v := newTestValidator()

	cases := map[string]string{
		"different domain":     "https://evil.example/tickets",
		"suffix lookalike":     "https://evil-ticketmaster.com/event/1",
		"approved as a prefix": "https://ticketmaster.com.evil.example/event/1",
		"embedded in path":     "https://evil.example/ticketmaster.com/event/1",
	}

	for name, raw := range cases {
		if _, err := v.Validate(raw); !errors.Is(err, models.ErrInvalidTicketURL) {
			t.Errorf("%s: expected rejection for %q, got %v", name, raw, err)
		}
	}
}

func TestValidateRejectsNonHTTPSSchemes(t *testing.T) {
	v := newTestValidator()

	cases := []string{
		"http://ticketmaster.com/event/1",
		"ftp://ticketmaster.com/event/1",
		"javascript:alert(1)",
		"//ticketmaster.com/event/1",
	}

	for _, raw := range cases {
		if _, err := v.Validate(raw); !errors.Is(err, models.ErrInvalidTicketURL) {
			t.Errorf("expected rejection for %q, got %v", raw, err)
		}
	}
}

func TestValidateRejectsCredentialsInURL(t *testing.T) {
	v := newTestValidator()

	if _, err := v.Validate("https://ticketmaster.com@abcd.example/x"); !errors.Is(err, models.ErrInvalidTicketURL) {
		t.Fatalf("expected credentials to be rejected, got %v", err)
	}
}

func TestValidateRejectsEmptyAndMalformedInput(t *testing.T) {
	v := newTestValidator()

	cases := []string{"", "   ", "https://", "not a url at all", "https:// space.com"}

	for _, raw := range cases {
		if _, err := v.Validate(raw); !errors.Is(err, models.ErrInvalidTicketURL) {
			t.Errorf("expected rejection for %q, got %v", raw, err)
		}
	}
}

func TestValidateIsCaseInsensitiveOnHostAndScheme(t *testing.T) {
	v := newTestValidator()

	if _, err := v.Validate("HTTPS://WWW.TICKETMASTER.COM/event/1"); err != nil {
		t.Fatalf("expected an uppercase url to be accepted, got %v", err)
	}
}

func TestNewTicketLinkValidatorNormalisesTheAllowlist(t *testing.T) {
	v := NewTicketLinkValidator([]string{" TicketMaster.com ", "", "   "})

	if _, err := v.Validate("https://ticketmaster.com/e/1"); err != nil {
		t.Fatalf("allowlist entries should be trimmed and lowercased, got %v", err)
	}
}

func TestEmptyAllowlistRejectsEverything(t *testing.T) {
	v := NewTicketLinkValidator(nil)

	if _, err := v.Validate("https://ticketmaster.com/e/1"); !errors.Is(err, models.ErrInvalidTicketURL) {
		t.Fatal("an empty allowlist must reject every host")
	}
}
