package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"eventexplorer/cache"
	"eventexplorer/models"
)

func newTestService(p EventProvider) (EventService, *cache.Cache) {
	c := cache.New(time.Minute)
	return NewEventService(p, c, 6), c
}

func TestListByCityReturnsBothCategories(t *testing.T) {
	p := newStubProvider()
	p.eventsByCategory[models.CategoryMusic] = []models.TMEvent{testEvent("m1", "Jazz Night")}
	p.eventsByCategory[models.CategorySports] = []models.TMEvent{testEvent("s1", "City Hoops")}

	svc, _ := newTestService(p)
	sections := svc.ListByCity(context.Background(), "Toronto", "CA")

	if len(sections) != 2 {
		t.Fatalf("expected one section per category, got %d", len(sections))
	}

	if sections[0].Category != models.CategoryMusic || sections[1].Category != models.CategorySports {
		t.Errorf("sections came back in the wrong order: %v, %v",
			sections[0].Category, sections[1].Category)
	}
	if sections[0].Count() != 1 || sections[1].Count() != 1 {
		t.Errorf("expected one event per section, got %d and %d",
			sections[0].Count(), sections[1].Count())
	}
	if sections[0].HasError() || sections[1].HasError() {
		t.Error("neither section should report an error")
	}
}

func TestListByCityKeepsTheWorkingSectionWhenOneFails(t *testing.T) {
	p := newStubProvider()
	p.eventsByCategory[models.CategoryMusic] = []models.TMEvent{testEvent("m1", "Jazz Night")}
	p.errsByCategory[models.CategorySports] = models.ErrUpstreamUnavailable

	svc, _ := newTestService(p)
	sections := svc.ListByCity(context.Background(), "Toronto", "CA")

	music, sports := sections[0], sections[1]

	if music.HasError() {
		t.Error("music succeeded and must still render")
	}
	if music.Count() != 1 {
		t.Errorf("expected the music events, got %d", music.Count())
	}
	if !sports.HasError() {
		t.Fatal("sports failed and must report an error")
	}
	if sports.Count() != 0 {
		t.Error("a failed section carries no events")
	}
}

func TestListByCityWhenBothCategoriesFail(t *testing.T) {
	p := newStubProvider()
	p.errsByCategory[models.CategoryMusic] = models.ErrUpstreamUnavailable
	p.errsByCategory[models.CategorySports] = models.ErrUpstreamUnavailable

	svc, _ := newTestService(p)
	sections := svc.ListByCity(context.Background(), "Toronto", "CA")

	page := models.ListingPageData{Sections: sections}
	if !page.AllFailed() {
		t.Error("the page should report a whole-page failure")
	}
}

func TestListByCityEmptyResultIsNotAnError(t *testing.T) {
	p := newStubProvider()

	svc, _ := newTestService(p)
	sections := svc.ListByCity(context.Background(), "Dhaka", "BD")

	for _, s := range sections {
		if s.HasError() {
			t.Errorf("%s: an empty result is not a failure", s.Category)
		}
		if !s.IsEmpty() {
			t.Errorf("%s: expected the empty state", s.Category)
		}
	}
}

func TestListByCityCapsEachSectionAtTheConfiguredSize(t *testing.T) {
	p := newStubProvider()
	many := make([]models.TMEvent, 12)
	for i := range many {
		many[i] = testEvent("e", "Event")
	}
	p.eventsByCategory[models.CategoryMusic] = many
	p.eventsByCategory[models.CategorySports] = many

	svc, _ := newTestService(p)
	sections := svc.ListByCity(context.Background(), "Toronto", "CA")

	for _, s := range sections {
		if s.Count() != 6 {
			t.Errorf("%s: expected six events, got %d", s.Category, s.Count())
		}
	}
}

func TestListByCityUsesTheCacheOnASecondCall(t *testing.T) {
	p := newStubProvider()
	p.eventsByCategory[models.CategoryMusic] = []models.TMEvent{testEvent("m1", "Jazz")}
	p.eventsByCategory[models.CategorySports] = []models.TMEvent{testEvent("s1", "Hoops")}

	svc, _ := newTestService(p)

	first := svc.ListByCity(context.Background(), "Toronto", "CA")
	if p.ListCallCount() != 2 {
		t.Fatalf("the first call should reach the provider twice, got %d", p.ListCallCount())
	}
	for _, s := range first {
		if s.Cached {
			t.Errorf("%s: the first call is not cached", s.Category)
		}
	}

	second := svc.ListByCity(context.Background(), "Toronto", "CA")
	if p.ListCallCount() != 2 {
		t.Errorf("the second call should be served from cache, provider hit %d times",
			p.ListCallCount())
	}
	for _, s := range second {
		if !s.Cached {
			t.Errorf("%s: expected the section to be marked cached", s.Category)
		}
		if s.Count() != 1 {
			t.Errorf("%s: cached data should still render", s.Category)
		}
	}
}

func TestFailedSectionsAreNotCached(t *testing.T) {
	p := newStubProvider()
	p.eventsByCategory[models.CategoryMusic] = []models.TMEvent{testEvent("m1", "Jazz")}
	p.errsByCategory[models.CategorySports] = models.ErrUpstreamUnavailable

	svc, c := newTestService(p)
	svc.ListByCity(context.Background(), "Toronto", "CA")

	if _, ok := c.Get(cache.EventsKey("Toronto", "CA", "Sports")); ok {
		t.Error("a failed fetch must not be cached")
	}
	if _, ok := c.Get(cache.EventsKey("Toronto", "CA", "Music")); !ok {
		t.Error("a successful fetch should be cached")
	}
}

func TestCacheEntriesOfTheWrongTypeAreDiscarded(t *testing.T) {
	p := newStubProvider()
	p.eventsByCategory[models.CategoryMusic] = []models.TMEvent{testEvent("m1", "Jazz")}

	svc, c := newTestService(p)

	// Poison the key
	c.Set(cache.EventsKey("Toronto", "CA", "Music"), "not an event list")

	sections := svc.ListByCity(context.Background(), "Toronto", "CA")
	if sections[0].HasError() {
		t.Fatal("the service should have refetched rather than failed")
	}
	if sections[0].Count() != 1 {
		t.Errorf("expected the refetched event, got %d", sections[0].Count())
	}
}

func TestListByCityFetchesCategoriesConcurrently(t *testing.T) {
	p := newStubProvider()
	p.delay = 120 * time.Millisecond
	p.eventsByCategory[models.CategoryMusic] = []models.TMEvent{testEvent("m1", "Jazz")}
	p.eventsByCategory[models.CategorySports] = []models.TMEvent{testEvent("s1", "Hoops")}

	svc, _ := newTestService(p)

	start := time.Now()
	svc.ListByCity(context.Background(), "Toronto", "CA")
	elapsed := time.Since(start)

	if elapsed > 200*time.Millisecond {
		t.Errorf("the categories look sequential, took %s for two %s requests",
			elapsed, p.delay)
	}
}

func TestGetEventReturnsTheDetailShape(t *testing.T) {
	p := newStubProvider()
	p.event = testEvent("ev-9", "The Weekend Sound")

	svc, _ := newTestService(p)
	detail, err := svc.GetEvent(context.Background(), "ev-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if detail.ID != "ev-9" || detail.Name != "The Weekend Sound" {
		t.Errorf("unexpected detail: %+v", detail)
	}
	if detail.RedirectURL != "/redirect/ev-9" {
		t.Errorf("the ticket link must go through our route, got %q", detail.RedirectURL)
	}
}

func TestGetEventPropagatesNotFound(t *testing.T) {
	p := newStubProvider()
	p.eventErr = models.ErrNotFound

	svc, _ := newTestService(p)
	if _, err := svc.GetEvent(context.Background(), "nope"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetEventPropagatesUpstreamFailure(t *testing.T) {
	p := newStubProvider()
	p.eventErr = models.ErrUpstreamUnavailable

	svc, _ := newTestService(p)
	if _, err := svc.GetEvent(context.Background(), "ev-1"); !errors.Is(err, models.ErrUpstreamUnavailable) {
		t.Fatalf("expected ErrUpstreamUnavailable, got %v", err)
	}
}

func TestGetEventIsCached(t *testing.T) {
	p := newStubProvider()
	p.event = testEvent("ev-9", "Cached Event")

	svc, _ := newTestService(p)

	if _, err := svc.GetEvent(context.Background(), "ev-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := svc.GetEvent(context.Background(), "ev-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.EventCallCount() != 1 {
		t.Errorf("the second lookup should come from cache, provider hit %d times",
			p.EventCallCount())
	}
}

func TestRawEventSharesTheDetailsCacheEntry(t *testing.T) {
	p := newStubProvider()
	p.event = testEvent("ev-9", "Shared")

	svc, _ := newTestService(p)

	if _, err := svc.GetEvent(context.Background(), "ev-9"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := svc.RawEvent(context.Background(), "ev-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if raw.URL == "" {
		t.Error("the raw event should carry the provider ticket url")
	}
	if p.EventCallCount() != 1 {
		t.Errorf("RawEvent should reuse the cached event, provider hit %d times",
			p.EventCallCount())
	}
}

func TestRawEventPropagatesErrors(t *testing.T) {
	p := newStubProvider()
	p.eventErr = models.ErrNotFound

	svc, _ := newTestService(p)
	if _, err := svc.RawEvent(context.Background(), "nope"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRawEventDiscardsAPoisonedCacheEntry(t *testing.T) {
	p := newStubProvider()
	p.event = testEvent("ev-9", "Recovered")

	svc, c := newTestService(p)
	c.Set(cache.EventKey("ev-9"), 12345)

	got, err := svc.RawEvent(context.Background(), "ev-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "ev-9" {
		t.Errorf("expected a refetched event, got %+v", got)
	}
}

func TestGetEventDiscardsAPoisonedCacheEntry(t *testing.T) {
	p := newStubProvider()
	p.event = testEvent("ev-9", "Recovered")

	svc, c := newTestService(p)
	c.Set(cache.EventKey("ev-9"), "garbage")

	got, err := svc.GetEvent(context.Background(), "ev-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Recovered" {
		t.Errorf("expected a refetched event, got %+v", got)
	}
}

func TestNewEventServiceDefaults(t *testing.T) {
	svc := NewEventService(newStubProvider(), cache.New(time.Minute), 0)
	impl, ok := svc.(*eventService)
	if !ok {
		t.Fatal("unexpected implementation type")
	}
	if impl.perPage != 6 {
		t.Errorf("expected a default of 6, got %d", impl.perPage)
	}

	withShared := NewEventService(newStubProvider(), nil, 6).(*eventService)
	if withShared.cache == nil {
		t.Error("a nil cache should fall back to the shared cache")
	}
}

func TestSectionErrorMessageNamesTheCategory(t *testing.T) {
	if got := sectionErrorMessage(models.CategorySports); got == "" {
		t.Fatal("the message must not be empty")
	}
	if got := sectionErrorMessage(models.CategoryMusic); got[:5] != "music" {
		t.Errorf("expected the category name, got %q", got)
	}
}
