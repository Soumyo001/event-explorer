package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"eventexplorer/models"
	"eventexplorer/services"

	beego "github.com/beego/beego/v2/server/web"
)

func locationApp(svc services.LocationService) *beego.HttpServer {
	app := newApp()
	app.Handlers.Add("/api/locations/autocomplete",
		&LocationController{Locations: svc}, beego.WithRouterMethods(&LocationController{}, "get:Autocomplete"))
	app.Handlers.Add("/api/locations/:placeId",
		&LocationController{Locations: svc}, beego.WithRouterMethods(&LocationController{}, "get:PlaceDetails"))
	return app
}

func TestAutocompleteReturnsSuggestionsAndAttribution(t *testing.T) {
	svc := &stubLocationService{suggestions: []models.CitySuggestion{
		{PlaceID: "p1", MainText: "Toronto", SecondaryText: "ON, Canada"},
	}}

	rec := serve(t, locationApp(svc), http.MethodGet, "/api/locations/autocomplete?input=toronto&sessionToken=s1")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}

	var body models.AutocompleteAPIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not valid json: %v", err)
	}
	if len(body.Suggestions) != 1 || body.Suggestions[0].PlaceID != "p1" {
		t.Errorf("unexpected suggestions: %+v", body.Suggestions)
	}

	if body.Attribution != models.GoogleAttribution {
		t.Errorf("the attribution is required, got %q", body.Attribution)
	}
}

func TestAutocompleteReturnsAnEmptyListNotNull(t *testing.T) {
	rec := serve(t, locationApp(&stubLocationService{}), http.MethodGet, "/api/locations/autocomplete?input=zzz&sessionToken=s1")

	if strings.Contains(rec.Body.String(), `"suggestions":null`) {
		t.Errorf("expected an empty array, got %s", rec.Body)
	}
}

func TestAutocompleteShortInputIsABadRequest(t *testing.T) {
	svc := &stubLocationService{err: models.ErrInvalidInput}

	rec := serve(t, locationApp(svc), http.MethodGet, "/api/locations/autocomplete?input=to&sessionToken=s1")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var body models.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("the error response should be json: %v", err)
	}
	if body.Error != "invalid_input" {
		t.Errorf("unexpected error code: %q", body.Error)
	}
}

func TestAutocompleteProviderFailureIsABadGateway(t *testing.T) {
	svc := &stubLocationService{err: models.ErrUpstreamUnavailable}

	rec := serve(t, locationApp(svc), http.MethodGet, "/api/locations/autocomplete?input=toronto&sessionToken=s1")

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", rec.Code)
	}

	if strings.Contains(rec.Body.String(), "PERMISSION_DENIED") {
		t.Error("provider detail leaked into the response")
	}
}

func TestPlaceDetailsReturnsTheSelectedCity(t *testing.T) {
	svc := &stubLocationService{city: models.SelectedCity{City: "Toronto", CountryCode: "CA"}}

	rec := serve(t, locationApp(svc), http.MethodGet, "/api/locations/ChIJ123?sessionToken=s1")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}

	var city models.SelectedCity
	if err := json.Unmarshal(rec.Body.Bytes(), &city); err != nil {
		t.Fatalf("response was not valid json: %v", err)
	}
	if city.City != "Toronto" || city.CountryCode != "CA" {
		t.Errorf("unexpected city: %+v", city)
	}
}

func TestPlaceDetailsNotFoundIs404(t *testing.T) {
	svc := &stubLocationService{err: models.ErrNotFound}

	rec := serve(t, locationApp(svc), http.MethodGet, "/api/locations/nope?sessionToken=s1")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPlaceDetailsWithNoCityIs422(t *testing.T) {
	svc := &stubLocationService{err: models.ErrNoCityComponent}

	rec := serve(t, locationApp(svc), http.MethodGet, "/api/locations/p1?sessionToken=s1")

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", rec.Code)
	}
}
