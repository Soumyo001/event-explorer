package controllers

import (
	"net/http"
	"testing"

	"eventexplorer/models"
	"eventexplorer/services"

	beego "github.com/beego/beego/v2/server/web"
)

func redirectApp(events services.EventService, links services.TicketLinkValidator) *beego.HttpServer {
	app := newApp()
	app.Router("/redirect/:eventId",
		&RedirectController{Events: events, Links: links}, "get:Ticket")
	return app
}

func TestTicketRedirectIssuesA302(t *testing.T) {
	events := &stubEventService{raw: models.TMEvent{
		ID:  "ev-1",
		URL: "https://www.ticketmaster.com/event/ev-1",
	}}
	links := &stubTicketValidator{}

	rec := serve(t, redirectApp(events, links), http.MethodGet, "/redirect/ev-1")

	if rec.Code != http.StatusFound {
		t.Fatalf("expected a 302, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://www.ticketmaster.com/event/ev-1" {
		t.Errorf("unexpected redirect target: %q", got)
	}
}

func TestTicketRedirectRefusesAnUnapprovedLink(t *testing.T) {
	events := &stubEventService{raw: models.TMEvent{ID: "ev-1", URL: "https://evil.example/x"}}
	links := &stubTicketValidator{err: models.ErrInvalidTicketURL}

	rec := serve(t, redirectApp(events, links), http.MethodGet, "/redirect/ev-1")

	if rec.Code == http.StatusFound {
		t.Fatal("an unapproved link must not produce a redirect")
	}
	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "" {
		t.Error("no Location header should be sent for a rejected link")
	}
}

func TestTicketRedirectUnknownEventIs404(t *testing.T) {
	events := &stubEventService{err: models.ErrNotFound}

	rec := serve(t, redirectApp(events, &stubTicketValidator{}), http.MethodGet, "/redirect/nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "" {
		t.Error("no redirect should be issued for an unknown event")
	}
}

func TestTicketRedirectUpstreamFailureIs502(t *testing.T) {
	events := &stubEventService{err: models.ErrUpstreamUnavailable}

	rec := serve(t, redirectApp(events, &stubTicketValidator{}), http.MethodGet, "/redirect/ev-1")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}

func TestTicketRedirectIgnoresAVisitorSuppliedDestination(t *testing.T) {
	events := &stubEventService{raw: models.TMEvent{
		ID:  "ev-1",
		URL: "https://www.ticketmaster.com/event/ev-1",
	}}
	links := &stubTicketValidator{}

	rec := serve(t, redirectApp(events, links), http.MethodGet,
		"/redirect/ev-1?url=https://evil.example&next=https://evil.example")

	if got := rec.Header().Get("Location"); got != "https://www.ticketmaster.com/event/ev-1" {
		t.Fatalf("the redirect must come from the event, got %q", got)
	}
}
