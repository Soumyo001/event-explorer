package services

import (
	"context"
	"eventexplorer/config"
	"eventexplorer/models"
	"eventexplorer/utils"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

var (
	initDefaultProviderOnce sync.Once
	defaultEventProvider    EventProvider
)

type EventProvider interface {
	FetchEvents(ctx context.Context, city, countryCode string, category models.Category, size int) ([]models.TMEvent, error)
	FetchEvent(ctx context.Context, eventID string) (models.TMEvent, error)
}

type TicketmasterService struct {
	client  *utils.Client
	apiKey  string
	baseURL string
}

func ticketmasterErrorMessage(res utils.Response) string {
	var e models.TMErrorResponse
	if err := res.DecodeJSON(&e); err != nil {
		return "unreadable error body"
	}
	return e.Message()
}

func NewTicketmasterService(cfg *config.Config) *TicketmasterService {
	return &TicketmasterService{
		client:  utils.NewClient(cfg.HTTPTimeout),
		apiKey:  cfg.TicketmasterAPIKey,
		baseURL: strings.TrimRight(cfg.TicketmasterBase, "/"),
	}
}

func NewTicketmasterServiceWith(client *utils.Client, apiKey, baseURL string) *TicketmasterService {
	return &TicketmasterService{
		client:  client,
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func (s *TicketmasterService) FetchEvents(ctx context.Context, city, countryCode string, category models.Category, size int) ([]models.TMEvent, error) {
	city = strings.TrimSpace(city)
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))

	if city == "" || countryCode == "" {
		return nil, fmt.Errorf("%w: city and countryCode are required", models.ErrInvalidInput)
	}
	if !category.Valid() {
		return nil, fmt.Errorf("%w: unsupported category %q", models.ErrInvalidInput, category)
	}
	if s.apiKey == "" {
		logs.Error("ticketmaster api key not configured")
		return nil, models.ErrUpstreamUnavailable
	}

	queryParams := url.Values{}
	queryParams.Set("apikey", s.apiKey)
	queryParams.Set("city", city)
	queryParams.Set("countryCode", countryCode)
	queryParams.Set("classificationName", category.String())
	queryParams.Set("size", strconv.Itoa(size))

	endpoint := s.baseURL + "/events.json?" + queryParams.Encode()

	res, err := s.client.Get(ctx, endpoint, nil)
	if err != nil {
		logs.Error("ticketmaster events request failed (%s/%s): %v", city, category, err)
		return nil, models.ErrUpstreamUnavailable
	}

	if !res.OK() {
		logs.Error("ticketmaster events returned %d for %s/%s: %s", res.StatusCode, city, category, ticketmasterErrorMessage(res))
		return nil, models.ErrUpstreamUnavailable
	}

	var decoded models.TMEventsResponse
	if err := res.DecodeJSON(&decoded); err != nil {
		logs.Error("ticketmaster events decode failed (%s/%s): %v", city, category, err)
		return nil, models.ErrUpstreamUnavailable
	}

	return decoded.Events(), nil
}

func (s *TicketmasterService) FetchEvent(ctx context.Context, eventID string) (models.TMEvent, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return models.TMEvent{}, fmt.Errorf("%w: event id is required", models.ErrInvalidInput)
	}
	if s.apiKey == "" {
		logs.Error("ticketmaster api key is not configured")
		return models.TMEvent{}, models.ErrUpstreamUnavailable
	}

	endpoint := s.baseURL + "/events/" + url.PathEscape(eventID) + ".json?apikey=" + url.QueryEscape(s.apiKey)

	res, err := s.client.Get(ctx, endpoint, nil)
	if err != nil {
		logs.Error("ticketmaster event request failed (%s): %v", eventID, err)
		return models.TMEvent{}, models.ErrUpstreamUnavailable
	}

	if res.StatusCode == 404 || res.StatusCode == 400 {
		logs.Warn("ticketmaster event %q not found (%d)", eventID, res.StatusCode)
		return models.TMEvent{}, models.ErrNotFound
	}
	if !res.OK() {
		logs.Error("ticketmaster event returned %d for %s: %s", res.StatusCode, eventID, ticketmasterErrorMessage(res))
		return models.TMEvent{}, models.ErrUpstreamUnavailable
	}

	var event models.TMEvent
	if err := res.DecodeJSON(&event); err != nil {
		logs.Error("ticketmaster event decode failed (%s): %v", eventID, err)
		return models.TMEvent{}, models.ErrUpstreamUnavailable
	}
	if event.ID == "" {
		logs.Warn("ticketmaster event %q returned an empty payload", eventID)
		return models.TMEvent{}, models.ErrNotFound
	}

	return event, nil
}

func DefaultEventProvider() EventProvider {
	initDefaultProviderOnce.Do(func() {
		defaultEventProvider = NewTicketmasterService(config.GetConfig())
	})
	return defaultEventProvider
}
