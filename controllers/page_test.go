package controllers

import (
	"net/http"
	"strings"
	"testing"

	"eventexplorer/models"
	"eventexplorer/services"

	beego "github.com/beego/beego/v2/server/web"
)

func pageApp(svc services.EventService) *beego.HttpServer {
	app := newApp()
	app.Router("/", &PageController{Events: svc}, "get:Home")
	app.Router("/events", &PageController{Events: svc}, "get:Listing")
	app.Router("/events/:eventId", &PageController{Events: svc}, "get:Details")
	return app
}

func TestHomeRendersTheSearchPage(t *testing.T) {
	rec := serve(t, pageApp(&stubEventService{}), http.MethodGet, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "city-search") {
		t.Error("the search form should be rendered")
	}

	if !strings.Contains(body, models.GoogleAttribution) {
		t.Error("the Google attribution is missing from the home page")
	}

	if !strings.Contains(body, "site-header") || !strings.Contains(body, "site-footer") {
		t.Error("the shared header and footer should be rendered")
	}
}

func TestListingRendersBothSections(t *testing.T) {
	svc := &stubEventService{sections: []models.EventSection{
		models.NewEventSection(models.CategoryMusic, []models.EventCard{
			{ID: "m1", Name: "Jazz Night", DetailsURL: "/events/m1", DateLabel: "Wed, 10 Feb 2027"},
		}, false),
		models.NewEventSection(models.CategorySports, []models.EventCard{
			{ID: "s1", Name: "City Hoops", DetailsURL: "/events/s1", DateLabel: "Thu, 11 Feb 2027"},
		}, false),
	}}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events?city=Toronto&countryCode=CA")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"Toronto", "Music", "Sports", "Jazz Night", "City Hoops", "/events/m1"} {
		if !strings.Contains(body, want) {
			t.Errorf("the listing should contain %q", want)
		}
	}
}

func TestListingShowsAPartialFailure(t *testing.T) {
	svc := &stubEventService{sections: []models.EventSection{
		models.NewEventSection(models.CategoryMusic, []models.EventCard{
			{ID: "m1", Name: "Jazz Night", DetailsURL: "/events/m1"},
		}, false),
		models.NewFailedEventSection(models.CategorySports, "sports events could not be loaded right now."),
	}}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events?city=Toronto&countryCode=CA")
	body := rec.Body.String()

	if !strings.Contains(body, "Jazz Night") {
		t.Error("the working section must still render")
	}
	if !strings.Contains(body, "could not be loaded") {
		t.Error("the failed section should show its error message")
	}
}

func TestListingShowsTheEmptyState(t *testing.T) {
	svc := &stubEventService{sections: []models.EventSection{
		models.NewEventSection(models.CategoryMusic, nil, false),
		models.NewEventSection(models.CategorySports, nil, false),
	}}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events?city=Dhaka&countryCode=BD")

	if rec.Code != http.StatusOK {
		t.Fatalf("an empty result is not an error, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "No ") {
		t.Error("expected an empty-state message")
	}
}

func TestListingMarksCachedSections(t *testing.T) {
	svc := &stubEventService{sections: []models.EventSection{
		models.NewEventSection(models.CategoryMusic, []models.EventCard{{ID: "m1", Name: "Jazz"}}, true),
		models.NewEventSection(models.CategorySports, []models.EventCard{{ID: "s1", Name: "Hoops"}}, true),
	}}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events?city=Toronto&countryCode=CA")

	if !strings.Contains(rec.Body.String(), "From cache") {
		t.Error("a cached section should be marked, it is the visible evidence of reuse")
	}
}

func TestListingRequiresBothParameters(t *testing.T) {
	cases := []string{"/events", "/events?city=Toronto", "/events?countryCode=CA", "/events?city=+&countryCode=CA"}

	for _, target := range cases {
		rec := serve(t, pageApp(&stubEventService{}), http.MethodGet, target)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", target, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "Choose a city") {
			t.Errorf("%s: expected the error page body", target)
		}
	}
}

func TestDetailsRendersTheEvent(t *testing.T) {
	svc := &stubEventService{detail: models.EventDetail{
		ID:          "ev-1",
		Name:        "The Weekend Sound",
		DateLabel:   "Wed, 10 Feb 2027",
		VenueName:   "Sample Riverside Hall",
		Description: "An evening of live music.",
		RedirectURL: "/redirect/ev-1",
	}}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events/ev-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, want := range []string{"The Weekend Sound", "Sample Riverside Hall", "An evening of live music."} {
		if !strings.Contains(body, want) {
			t.Errorf("the details page should contain %q", want)
		}
	}

	if !strings.Contains(body, "/redirect/ev-1") {
		t.Error("the View Tickets link should point at the redirect route")
	}
	if strings.Contains(body, "ticketmaster.com") {
		t.Error("the provider url must never reach the page")
	}
}

func TestDetailsWorksAsADirectLink(t *testing.T) {
	svc := &stubEventService{detail: models.EventDetail{ID: "ev-1", Name: "Direct", RedirectURL: "/redirect/ev-1"}}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events/ev-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("a direct visit should render, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `href="/"`) {
		t.Error("with no referrer the back link should fall back to home")
	}
}

func TestDetailsBackLinkUsesTheReferringListing(t *testing.T) {
	svc := &stubEventService{detail: models.EventDetail{ID: "ev-1", Name: "Event"}}

	rec := serveWithReferer(t, pageApp(svc), http.MethodGet, "/events/ev-1",
		"http://localhost:8080/events?city=Toronto&countryCode=CA")

	if !strings.Contains(rec.Body.String(), "/events?city=Toronto&amp;countryCode=CA") {
		t.Error("the back link should return to the referring listing")
	}
}

func TestDetailsBackLinkIgnoresAnExternalReferrer(t *testing.T) {
	svc := &stubEventService{detail: models.EventDetail{ID: "ev-1", Name: "Event"}}

	cases := []string{
		"https://evil.example/events?city=Toronto&countryCode=CA",
		"http://localhost:8080/somewhere-else?city=Toronto&countryCode=CA",
		"not a url",
	}

	for _, referer := range cases {
		rec := serveWithReferer(t, pageApp(svc), http.MethodGet, "/events/ev-1", referer)
		if strings.Contains(rec.Body.String(), "evil.example") {
			t.Errorf("%s: an external referrer must never be used as a link", referer)
		}
	}
}

func TestDetailsUnknownEventIs404(t *testing.T) {
	svc := &stubEventService{err: models.ErrNotFound}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events/nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Not found") {
		t.Error("expected the styled error page")
	}

	if strings.Contains(body, "View tickets") {
		t.Error("a failed lookup must not render the details page")
	}
}

func TestDetailsUpstreamFailureIs502(t *testing.T) {
	svc := &stubEventService{err: models.ErrUpstreamUnavailable}

	rec := serve(t, pageApp(svc), http.MethodGet, "/events/ev-1")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}
}
