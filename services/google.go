package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"eventexplorer/config"
	"eventexplorer/models"
	"eventexplorer/utils"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

const minAutocompleteChars = 3

var (
	initDefaultLocationOnce sync.Once
	defaultLocationService  LocationService
)

type LocationService interface {
	Autocomplete(ctx context.Context, input, sessionToken string) ([]models.CitySuggestion, error)
	PlaceDetails(ctx context.Context, placeID, sessionToken string) (models.SelectedCity, error)
}

type GoogleService struct {
	client  *utils.Client
	apiKey  string
	baseURL string
}

func googleErrorMessage(res utils.Response) string {
	var e models.GoogleErrorResponse
	if err := res.DecodeJSON(&e); err != nil {
		return "unreadable error body"
	}
	return e.Message()
}

func (s *GoogleService) headers(extra map[string]string) map[string]string {
	h := map[string]string{"X-Goog-Api-Key": s.apiKey}
	for k, v := range extra {
		h[k] = v
	}
	return h
}

func (s *GoogleService) Autocomplete(ctx context.Context, input, sessionToken string) ([]models.CitySuggestion, error) {
	input = strings.TrimSpace(input)
	if len([]rune(input)) < minAutocompleteChars {
		return nil, fmt.Errorf("%w: input must be at least %d characters",
			models.ErrInvalidInput, minAutocompleteChars)
	}
	if s.apiKey == "" {
		logs.Error("google autocomplete: places api key is not configured")
		return nil, models.ErrUpstreamUnavailable
	}

	endpoint := s.baseURL + "/v1/places:autocomplete"
	body := models.NewAutocompleteRequest(input, sessionToken)

	res, err := s.client.PostJSON(ctx, endpoint, s.headers(nil), body)
	if err != nil {
		logs.Error("google autocomplete request failed: %v", err)
		return nil, models.ErrUpstreamUnavailable
	}
	if !res.OK() {
		logs.Error("google autocomplete returned %d: %s", res.StatusCode, googleErrorMessage(res))
		return nil, models.ErrUpstreamUnavailable
	}

	var decoded models.GoogleAutocompleteResponse
	if err := res.DecodeJSON(&decoded); err != nil {
		logs.Error("google autocomplete decode failed: %v", err)
		return nil, models.ErrUpstreamUnavailable
	}

	return decoded.ToCitySuggestions(), nil
}

func (s *GoogleService) PlaceDetails(ctx context.Context, placeID, sessionToken string) (models.SelectedCity, error) {
	placeID = strings.TrimSpace(placeID)
	if placeID == "" {
		return models.SelectedCity{}, fmt.Errorf("%w: place id is required", models.ErrInvalidInput)
	}
	if s.apiKey == "" {
		logs.Error("google place details: places api key is not configured")
		return models.SelectedCity{}, models.ErrUpstreamUnavailable
	}

	endpoint := s.baseURL + "/v1/places/" + url.PathEscape(placeID)
	if sessionToken != "" {
		endpoint += "?sessionToken=" + url.QueryEscape(sessionToken)
	}

	headers := s.headers(map[string]string{
		"X-Goog-FieldMask": "addressComponents",
	})

	res, err := s.client.Get(ctx, endpoint, headers)
	if err != nil {
		logs.Error("google place details request failed: %v", err)
		return models.SelectedCity{}, models.ErrUpstreamUnavailable
	}
	if res.StatusCode == 404 {
		logs.Warn("google place details: place %q not found", placeID)
		return models.SelectedCity{}, models.ErrNotFound
	}
	if !res.OK() {
		logs.Error("google place details returned %d: %s", res.StatusCode, googleErrorMessage(res))
		return models.SelectedCity{}, models.ErrUpstreamUnavailable
	}

	var decoded models.GooglePlaceDetailsResponse
	if err := res.DecodeJSON(&decoded); err != nil {
		logs.Error("google place details decode failed: %v", err)
		return models.SelectedCity{}, models.ErrUpstreamUnavailable
	}

	city, err := decoded.ToSelectedCity()
	if err != nil {
		logs.Warn("google place details: no city component for %q", placeID)
		return models.SelectedCity{}, err
	}
	return city, nil
}

var _ LocationService = (*GoogleService)(nil) // check & validation

func NewGoogleService(cfg *config.Config) *GoogleService {
	return &GoogleService{
		client:  utils.NewClient(cfg.HTTPTimeout),
		apiKey:  cfg.GoogleAPIKey,
		baseURL: strings.TrimRight(cfg.GooglePlacesBase, "/"),
	}
}

func NewGoogleServiceWith(client *utils.Client, apiKey, baseURL string) *GoogleService {
	return &GoogleService{
		client:  client,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func NewSessionToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		logs.Error("session token generation failed: %v", err)
		return "fallback-session-token"
	}
	return hex.EncodeToString(b)
}

func DefaultLocationService() LocationService {
	initDefaultLocationOnce.Do(func() {
		defaultLocationService = NewGoogleService(config.GetConfig())
	})
	return defaultLocationService
}
