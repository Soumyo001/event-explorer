package models

import (
	"errors"
	"testing"
)

func TestNewAutocompleteRequestTrimsAndSetsCitiesType(t *testing.T) {
	req := NewAutocompleteRequest("  toronto  ", "tok-1")

	if req.Input != "toronto" {
		t.Errorf("input should be trimmed, got %q", req.Input)
	}
	if len(req.IncludedPrimaryTypes) != 1 || req.IncludedPrimaryTypes[0] != CitiesPrimaryType {
		t.Errorf("expected the cities collection, got %v", req.IncludedPrimaryTypes)
	}
	if req.SessionToken != "tok-1" {
		t.Errorf("session token was not carried through, got %q", req.SessionToken)
	}
}

func TestToCitySuggestionsMapsPredictions(t *testing.T) {
	resp := GoogleAutocompleteResponse{
		Suggestions: []GoogleSuggestion{
			{PlacePrediction: &GooglePlacePrediction{
				PlaceID: "place-1",
				Text:    GoogleText{Text: "Toronto, ON, Canada"},
				StructuredFormat: GoogleStructuredFormat{
					MainText:      GoogleText{Text: "Toronto"},
					SecondaryText: GoogleText{Text: "ON, Canada"},
				},
			}},
		},
	}

	got := resp.ToCitySuggestions()
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(got))
	}
	if got[0].PlaceID != "place-1" || got[0].MainText != "Toronto" {
		t.Errorf("suggestion mapped wrong: %+v", got[0])
	}
	if got[0].SecondaryText != "ON, Canada" {
		t.Errorf("secondary text missing: %+v", got[0])
	}
}

func TestToCitySuggestionsSkipsUnusableEntries(t *testing.T) {
	resp := GoogleAutocompleteResponse{
		Suggestions: []GoogleSuggestion{
			{PlacePrediction: nil},
			{PlacePrediction: &GooglePlacePrediction{PlaceID: "   "}},
			{PlacePrediction: &GooglePlacePrediction{
				PlaceID: "keep-me",
				Text:    GoogleText{Text: "Dhaka"},
			}},
		},
	}

	got := resp.ToCitySuggestions()
	if len(got) != 1 || got[0].PlaceID != "keep-me" {
		t.Fatalf("expected only the usable prediction, got %+v", got)
	}
	if got[0].MainText != "Dhaka" {
		t.Errorf("expected a fallback to Text, got %q", got[0].MainText)
	}
}

func TestToCitySuggestionsNeverReturnsNil(t *testing.T) {
	got := GoogleAutocompleteResponse{}.ToCitySuggestions()
	if got == nil {
		t.Fatal("an empty response should still produce an empty slice, not nil")
	}
	if len(got) != 0 {
		t.Fatalf("expected no suggestions, got %d", len(got))
	}
}

func TestAddressComponentHasType(t *testing.T) {
	c := GoogleAddressComponent{Types: []string{"locality", "political"}}

	if !c.HasType("locality") {
		t.Error("expected locality to match")
	}
	if c.HasType("country") {
		t.Error("country should not match")
	}
}

func TestToSelectedCityExtractsCityAndCountry(t *testing.T) {
	resp := GooglePlaceDetailsResponse{
		AddressComponents: []GoogleAddressComponent{
			{LongText: "Ontario", ShortText: "ON", Types: []string{"administrative_area_level_1"}},
			{LongText: "Toronto", ShortText: "Toronto", Types: []string{"locality"}},
			{LongText: "Canada", ShortText: "ca", Types: []string{"country"}},
		},
	}

	city, err := resp.ToSelectedCity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if city.City != "Toronto" {
		t.Errorf("locality should win over the admin area, got %q", city.City)
	}
	if city.CountryCode != "CA" {
		t.Errorf("country code should be uppercased, got %q", city.CountryCode)
	}
}

func TestToSelectedCityFallsBackThroughPriorityTypes(t *testing.T) {
	resp := GooglePlaceDetailsResponse{
		AddressComponents: []GoogleAddressComponent{
			{LongText: "Reading", Types: []string{"postal_town"}},
			{LongText: "United Kingdom", ShortText: "GB", Types: []string{"country"}},
		},
	}

	city, err := resp.ToSelectedCity()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if city.City != "Reading" || city.CountryCode != "GB" {
		t.Errorf("unexpected result: %+v", city)
	}
}

func TestToSelectedCityErrorsWhenIncomplete(t *testing.T) {
	cases := map[string]GooglePlaceDetailsResponse{
		"no components": {},
		"no country": {AddressComponents: []GoogleAddressComponent{
			{LongText: "Toronto", Types: []string{"locality"}},
		}},
		"no city": {AddressComponents: []GoogleAddressComponent{
			{LongText: "Canada", ShortText: "CA", Types: []string{"country"}},
		}},
	}

	for name, resp := range cases {
		if _, err := resp.ToSelectedCity(); !errors.Is(err, ErrNoCityComponent) {
			t.Errorf("%s: expected ErrNoCityComponent, got %v", name, err)
		}
	}
}

func TestGoogleErrorResponseMessage(t *testing.T) {
	withMessage := GoogleErrorResponse{Error: GoogleErrorFormat{Message: "API key not valid"}}
	if got := withMessage.Message(); got != "API key not valid" {
		t.Errorf("expected the message, got %q", got)
	}

	statusOnly := GoogleErrorResponse{Error: GoogleErrorFormat{Status: "PERMISSION_DENIED"}}
	if got := statusOnly.Message(); got != "PERMISSION_DENIED" {
		t.Errorf("expected the status as a fallback, got %q", got)
	}

	if got := (GoogleErrorResponse{}).Message(); got == "" {
		t.Error("Message must never be empty")
	}
}
