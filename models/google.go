package models

import (
	"strings"
)

const CitiesPrimaryType = "(cities)"

// request schema
type AutocompleteRequest struct {
	Input                string   `json:"input"`
	IncludedPrimaryTypes []string `json:"includedPrimaryTypes,omitempty"`
	SessionToken         string   `json:"sessionToken,omitempty"`
}

func NewAutocompleteRequest(input, sessionToken string) AutocompleteRequest {
	return AutocompleteRequest{
		Input:                strings.TrimSpace(input),
		IncludedPrimaryTypes: []string{CitiesPrimaryType},
		SessionToken:         sessionToken,
	}
}

// response schema
// google autocomplete response
type GoogleText struct {
	Text string `json:"text"`
}
type GoogleStructuredFormat struct {
	MainText      GoogleText `json:"mainText"`
	SecondaryText GoogleText `json:"secondaryText"`
}
type GooglePlacePrediction struct {
	Place            string                 `json:"place"`
	PlaceID          string                 `json:"placeId"`
	Text             GoogleText             `json:"text"`
	StructuredFormat GoogleStructuredFormat `json:"structuredFormat"`
	Types            []string               `json:"types"`
}
type GoogleSuggestion struct {
	PlacePrediction *GooglePlacePrediction `json:"placePrediction,omitempty"`
}
type GoogleAutocompleteResponse struct {
	Suggestions []GoogleSuggestion `json:"suggestions"`
}

func (r GoogleAutocompleteResponse) ToCitySuggestions() []CitySuggestion {
	out := make([]CitySuggestion, 0, len(r.Suggestions))
	for _, s := range r.Suggestions {
		p := s.PlacePrediction
		if p == nil || strings.TrimSpace(p.PlaceID) == "" {
			continue
		}

		main := p.StructuredFormat.MainText.Text
		if main == "" {
			main = p.Text.Text
		}

		out = append(out, CitySuggestion{
			PlaceID:       p.PlaceID,
			MainText:      main,
			SecondaryText: p.StructuredFormat.SecondaryText.Text,
			Text:          p.Text.Text,
		})
	}
	return out
}

// google places response
type GoogleAddressComponent struct {
	LongText     string   `json:"longText"`
	ShortText    string   `json:"shortText"`
	Types        []string `json:"types"`
	LanguageCode string   `json:"languageCode"`
}

func (c GoogleAddressComponent) HasType(t string) bool {
	for _, haveType := range c.Types {
		if haveType == t {
			return true
		}
	}
	return false
}

type GooglePlaceDetailsResponse struct {
	AddressComponents []GoogleAddressComponent `json:"addressComponents"`
}

var cityTypesByPriority = []string{
	"locality",
	"postal_town",
	"administrative_area_level_3",
	"administrative_area_level_2",
	"administrative_area_level_1",
}

func (r GooglePlaceDetailsResponse) ToSelectedCity() (SelectedCity, error) {
	var city, countryCode string

	for _, cityType := range cityTypesByPriority {
		for _, comp := range r.AddressComponents {
			if comp.HasType(cityType) && comp.LongText != "" {
				city = comp.LongText
				break
			}
		}
		if city != "" {
			break
		}
	}

	for _, comp := range r.AddressComponents {
		if comp.HasType("country") && comp.ShortText != "" {
			countryCode = strings.ToUpper(comp.ShortText)
			break
		}
	}

	if city == "" || countryCode == "" {
		return SelectedCity{}, ErrNoCityComponent
	}
	return SelectedCity{City: city, CountryCode: countryCode}, nil
}

type GoogleErrorFormat struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}
type GoogleErrorResponse struct {
	Error GoogleErrorFormat `json:"error"`
}

func (e GoogleErrorResponse) Message() string {
	if e.Error.Message != "" {
		return e.Error.Message
	}
	if e.Error.Status != "" {
		return e.Error.Status
	}
	return "unknown google places error"
}
