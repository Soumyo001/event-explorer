package controllers

import (
	"context"

	"eventexplorer/models"
)

type stubEventService struct {
	sections []models.EventSection
	detail   models.EventDetail
	raw      models.TMEvent
	err      error
}

func (s *stubEventService) ListByCity(ctx context.Context, city, countryCode string) []models.EventSection {
	return s.sections
}

func (s *stubEventService) GetEvent(ctx context.Context, eventID string) (models.EventDetail, error) {
	if s.err != nil {
		return models.EventDetail{}, s.err
	}
	return s.detail, nil
}

func (s *stubEventService) RawEvent(ctx context.Context, eventID string) (models.TMEvent, error) {
	if s.err != nil {
		return models.TMEvent{}, s.err
	}
	return s.raw, nil
}

type stubLocationService struct {
	suggestions []models.CitySuggestion
	city        models.SelectedCity
	err         error
}

func (s *stubLocationService) Autocomplete(ctx context.Context, input, sessionToken string) ([]models.CitySuggestion, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.suggestions, nil
}

func (s *stubLocationService) PlaceDetails(ctx context.Context, placeID, sessionToken string) (models.SelectedCity, error) {
	if s.err != nil {
		return models.SelectedCity{}, s.err
	}
	return s.city, nil
}

type stubTicketValidator struct {
	target string
	err    error
}

func (s *stubTicketValidator) Validate(rawURL string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.target != "" {
		return s.target, nil
	}
	return rawURL, nil
}
