package services

import (
	"context"
	"eventexplorer/cache"
	"eventexplorer/config"
	"eventexplorer/models"
	"strings"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

var (
	initDefaultEventsOnce sync.Once
	defaultEventService   EventService
)

type EventService interface {
	ListByCity(ctx context.Context, city, countryCode string) []models.EventSection
	GetEvent(ctx context.Context, eventID string) (models.EventDetail, error)
	RawEvent(ctx context.Context, eventID string) (models.TMEvent, error)
}

type eventService struct {
	provider EventProvider
	cache    *cache.Cache
	perPage  int
}

type sectionResult struct {
	category models.Category
	events   []models.TMEvent
	cached   bool
	err      error
}

func sectionErrorMessage(c models.Category) string {
	return strings.ToLower(c.String()) + " events could not be loaded right now."
}

func NewEventService(provider EventProvider, c *cache.Cache, perPage int) EventService {
	if perPage <= 0 {
		perPage = 6
	}
	if c == nil {
		c = cache.Shared()
	}
	return &eventService{
		provider: provider,
		cache:    c,
		perPage:  perPage,
	}
}

func (s *eventService) fetchSection(ctx context.Context, city, countryCode string, category models.Category) sectionResult {
	key := cache.EventsKey(city, countryCode, category.String())

	if cached, ok := s.cache.Get(key); ok {
		if events, valid := cached.([]models.TMEvent); valid {
			return sectionResult{category: category, events: events, cached: true}
		}

		logs.Warn("cache entry %q had an unexpected type, refetching", key)
		s.cache.Delete(key)
	}

	events, err := s.provider.FetchEvents(ctx, city, countryCode, category, s.perPage)
	if err != nil {
		logs.Error("fetch failed for %s/%s/%s: %v", city, countryCode, category, err)
		return sectionResult{category: category, err: err}
	}

	s.cache.Set(key, events)
	return sectionResult{category: category, events: events}
}

func (s *eventService) ListByCity(ctx context.Context, city, countryCode string) []models.EventSection {
	categories := models.AllCategories
	results := make(chan sectionResult, len(categories))

	var wg sync.WaitGroup

	for _, category := range categories {
		wg.Add(1)
		go func(c models.Category) {
			defer wg.Done()
			results <- s.fetchSection(ctx, city, countryCode, c)
		}(category)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	collected := make(map[models.Category]sectionResult, len(categories))
	for res := range results {
		collected[res.category] = res
	}

	sections := make([]models.EventSection, 0, len(categories))
	for _, category := range categories {
		res, ok := collected[category]
		if !ok || res.err != nil {
			sections = append(sections,
				models.NewFailedEventSection(category, sectionErrorMessage(category)))
			continue
		}
		cards := models.ToEventCards(res.events, category, s.perPage)
		sections = append(sections, models.NewEventSection(category, cards, res.cached))
	}

	return sections
}

func (s *eventService) GetEvent(ctx context.Context, eventID string) (models.EventDetail, error) {
	eventID = strings.TrimSpace(eventID)
	key := cache.EventKey(eventID)

	if cached, ok := s.cache.Get(key); ok {
		if event, valid := cached.(models.TMEvent); valid {
			return event.ToDetail(), nil
		}
		logs.Warn("cache entry %q had an unexpected type, refetching", key)
		s.cache.Delete(key)
	}

	event, err := s.provider.FetchEvent(ctx, eventID)
	if err != nil {
		return models.EventDetail{}, err
	}

	s.cache.Set(key, event)
	return event.ToDetail(), nil
}

func (s *eventService) RawEvent(ctx context.Context, eventID string) (models.TMEvent, error) {
	eventID = strings.TrimSpace(eventID)
	key := cache.EventKey(eventID)

	if cached, ok := s.cache.Get(key); ok {
		if event, valid := cached.(models.TMEvent); valid {
			return event, nil
		}
		s.cache.Delete(key)
	}

	event, err := s.provider.FetchEvent(ctx, eventID)
	if err != nil {
		return models.TMEvent{}, err
	}

	s.cache.Set(key, event)
	return event, nil
}

var _ EventService = (*eventService)(nil)

func DefaultEventService() EventService {
	initDefaultEventsOnce.Do(func() {
		cfg := config.GetConfig()
		defaultEventService = NewEventService(
			DefaultEventProvider(), cache.Shared(), cfg.EventsPerCategory)
	})
	return defaultEventService
}
