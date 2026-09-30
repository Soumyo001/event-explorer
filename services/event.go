package services

import (
	"context"
	"eventexplorer/config"
	"eventexplorer/models"
	"strings"
	"sync"
)

var (
	initDefaultEventsOnce sync.Once
	defaultEventService   EventService
)

type EventService interface {
	ListByCity(ctx context.Context, city, countryCode string) []models.EventSection
	GetEvent(ctx context.Context, eventID string) (models.EventDetail, error)
}

type eventService struct {
	provider EventProvider
	perPage  int
}

func sectionErrorMessage(c models.Category) string {
	return strings.ToLower(c.String()) + " events could not be loaded right now."
}

func NewEventService(provider EventProvider, perPage int) EventService {
	if perPage <= 0 {
		perPage = 6
	}
	return &eventService{
		provider: provider,
		perPage:  perPage,
	}
}

func (s *eventService) ListByCity(ctx context.Context, city, countryCode string) []models.EventSection {
	sections := make([]models.EventSection, 0, len(models.AllCategories))

	for _, category := range models.AllCategories {
		events, err := s.provider.FetchEvents(ctx, city, countryCode, category, s.perPage)
		if err != nil {
			sections = append(sections, models.NewFailedEventSection(
				category, sectionErrorMessage(category)))
			continue
		}
		cards := models.ToEventCards(events, category, s.perPage)
		sections = append(sections, models.NewEventSection(category, cards, false))
	}

	return sections
}

func (s *eventService) GetEvent(ctx context.Context, eventID string) (models.EventDetail, error) {
	event, err := s.provider.FetchEvent(ctx, eventID)
	if err != nil {
		return models.EventDetail{}, err
	}
	return event.ToDetail(), nil
}

func DefaultEventService() EventService {
	initDefaultEventsOnce.Do(func() {
		cfg := config.GetConfig()
		defaultEventService = NewEventService(DefaultEventProvider(), cfg.EventsPerCategory)
	})
	return defaultEventService
}
