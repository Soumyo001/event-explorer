package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"eventexplorer/models"
)

type stubProvider struct {
	mu sync.Mutex

	// per-category canned answers
	eventsByCategory map[models.Category][]models.TMEvent
	errsByCategory   map[models.Category]error

	// single-event answers
	event    models.TMEvent
	eventErr error

	// delay lets a test prove the two category calls overlap
	delay time.Duration

	// call counters used to prove the cache prevented a refetch
	listCalls  int32
	eventCalls int32
}

func newStubProvider() *stubProvider {
	return &stubProvider{
		eventsByCategory: make(map[models.Category][]models.TMEvent),
		errsByCategory:   make(map[models.Category]error),
	}
}

func (s *stubProvider) FetchEvents(ctx context.Context, city, countryCode string, category models.Category, size int) ([]models.TMEvent, error) {
	atomic.AddInt32(&s.listCalls, 1)

	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err, ok := s.errsByCategory[category]; ok {
		return nil, err
	}
	return s.eventsByCategory[category], nil
}

func (s *stubProvider) FetchEvent(ctx context.Context, eventID string) (models.TMEvent, error) {
	atomic.AddInt32(&s.eventCalls, 1)

	if s.eventErr != nil {
		return models.TMEvent{}, s.eventErr
	}
	return s.event, nil
}

func (s *stubProvider) ListCallCount() int  { return int(atomic.LoadInt32(&s.listCalls)) }
func (s *stubProvider) EventCallCount() int { return int(atomic.LoadInt32(&s.eventCalls)) }

// testEvent builds a minimal usable event.
func testEvent(id, name string) models.TMEvent {
	e := models.TMEvent{
		ID:   id,
		Name: name,
		URL:  "https://www.ticketmaster.com/event/" + id,
		Dates: models.TMDates{
			Start: models.TMStart{LocalDate: "2027-03-01", LocalTime: "20:00:00"},
		},
		Classifications: []models.TMClassification{
			{Primary: true, Segment: models.TMNamed{Name: "Music"}},
		},
	}
	e.Embedded.Venues = []models.TMVenue{{
		Name: "Test Hall",
		City: models.TMNamed{Name: "Toronto"},
	}}
	return e
}
