package models

import (
	"testing"
)

func sampleEvent() TMEvent {
	return TMEvent{
		ID:   "ev-1",
		Name: "The Weekend Sound",
		URL:  "https://www.ticketmaster.ca/event/ev-1",
		Images: []TMImage{
			{Ratio: "16_9", URL: "https://img/large.jpg", Width: 2048},
			{Ratio: "16_9", URL: "https://img/small.jpg", Width: 1024},
			{Ratio: "3_2", URL: "https://img/wide.jpg", Width: 3000},
		},
		Dates: TMDates{
			Start:    TMStart{LocalDate: "2027-02-10", LocalTime: "19:30:00"},
			Timezone: "America/Toronto",
		},
		Classifications: []TMClassification{
			{Primary: true, Segment: TMNamed{Name: "Music"}},
		},
		Info: "An evening of live music.",
		Embedded: TMEventEmbedded{
			Venues: []TMVenue{{
				Name:    "Sample Riverside Hall",
				City:    TMNamed{Name: "Toronto"},
				State:   TMState{StateCode: "ON"},
				Country: TMCountry{CountryCode: "CA"},
				Address: TMAddress{Line1: "1 River Road"},
			}},
		},
	}
}

func TestEventsNeverReturnsNil(t *testing.T) {
	var empty TMEventsResponse
	if got := empty.Events(); got == nil || len(got) != 0 {
		t.Fatalf("expected an empty slice, got %#v", got)
	}

	filled := TMEventsResponse{}
	filled.Embedded.Events = []TMEvent{{ID: "a"}}
	if len(filled.Events()) != 1 {
		t.Fatal("expected the embedded events to come through")
	}
}

func TestTMErrorResponseMessage(t *testing.T) {
	withErrors := TMErrorResponse{Errors: []TMError{{Detail: "Invalid apikey"}}}
	if got := withErrors.Message(); got != "Invalid apikey" {
		t.Errorf("expected the error detail, got %q", got)
	}

	withFault := TMErrorResponse{Fault: &TMFault{FaultString: "Rate limit exceeded"}}
	if got := withFault.Message(); got != "Rate limit exceeded" {
		t.Errorf("expected the fault string, got %q", got)
	}

	if got := (TMErrorResponse{}).Message(); got == "" {
		t.Error("Message must never be empty")
	}
}

func TestBestImageURLPrefersTheWidest16by9(t *testing.T) {
	e := sampleEvent()
	if got := e.BestImageURL(); got != "https://img/large.jpg" {
		t.Errorf("expected the widest 16:9 image, got %q", got)
	}
}

func TestBestImageURLSkipsFallbacksAndBlanks(t *testing.T) {
	e := TMEvent{Images: []TMImage{
		{Ratio: "16_9", URL: "", Width: 4000},
		{Ratio: "16_9", URL: "https://img/placeholder.jpg", Width: 4000, Fallback: true},
		{Ratio: "4_3", URL: "https://img/usable.jpg", Width: 800},
	}}

	if got := e.BestImageURL(); got != "https://img/usable.jpg" {
		t.Errorf("expected the only usable image, got %q", got)
	}
}

func TestBestImageURLEmptyWhenNothingUsable(t *testing.T) {
	e := TMEvent{Images: []TMImage{{URL: "https://img/x.jpg", Fallback: true}}}
	if got := e.BestImageURL(); got != "" {
		t.Errorf("expected an empty string so a placeholder is used, got %q", got)
	}
}

func TestPrimaryVenueIsZeroWhenAbsent(t *testing.T) {
	if v := (TMEvent{}).PrimaryVenue(); v.Name != "" {
		t.Errorf("expected a zero venue, got %+v", v)
	}
}

func TestStartTimeParsesDateAndTime(t *testing.T) {
	e := sampleEvent()

	got, ok := e.StartTime()
	if !ok {
		t.Fatal("expected a usable start time")
	}
	if got.Hour() != 19 || got.Minute() != 30 {
		t.Errorf("time parsed wrong: %s", got)
	}
	if got.Year() != 2027 || got.Month() != 2 || got.Day() != 10 {
		t.Errorf("date parsed wrong: %s", got)
	}
}

func TestStartTimeFallsBackToDateOnly(t *testing.T) {
	e := sampleEvent()
	e.Dates.Start.LocalTime = ""

	got, ok := e.StartTime()
	if !ok {
		t.Fatal("a date without a time should still parse")
	}
	if got.Hour() != 0 {
		t.Errorf("expected midnight when no time is given, got %s", got)
	}
}

func TestStartTimeRejectsUnknownDates(t *testing.T) {
	cases := map[string]TMStart{
		"date TBA":   {LocalDate: "2027-02-10", DateTBA: true},
		"date TBD":   {LocalDate: "2027-02-10", DateTBD: true},
		"no date":    {},
		"bad format": {LocalDate: "10-02-2027"},
	}

	for name, start := range cases {
		e := TMEvent{Dates: TMDates{Start: start}}
		if _, ok := e.StartTime(); ok {
			t.Errorf("%s: expected no usable start time", name)
		}
	}
}

func TestDateLabelAndTimeLabel(t *testing.T) {
	e := sampleEvent()

	if got := e.DateLabel(); got != "Wed, 10 Feb 2027" {
		t.Errorf("unexpected date label: %q", got)
	}
	if got := e.TimeLabel(); got != "7:30 PM" {
		t.Errorf("unexpected time label: %q", got)
	}

	unknown := TMEvent{Dates: TMDates{Start: TMStart{DateTBA: true}}}
	if got := unknown.DateLabel(); got != "Date TBA" {
		t.Errorf("expected the TBA label, got %q", got)
	}
}

func TestTimeLabelEmptyWhenTimeIsUnknown(t *testing.T) {
	cases := map[string]TMStart{
		"time TBA":         {LocalDate: "2027-02-10", LocalTime: "19:30:00", TimeTBA: true},
		"no specific time": {LocalDate: "2027-02-10", LocalTime: "19:30:00", NoSpecificTime: true},
		"missing time":     {LocalDate: "2027-02-10"},
		"unparseable":      {LocalDate: "2027-02-10", LocalTime: "half seven"},
	}

	for name, start := range cases {
		e := TMEvent{Dates: TMDates{Start: start}}
		if got := e.TimeLabel(); got != "" {
			t.Errorf("%s: expected an empty label so the line is omitted, got %q", name, got)
		}
	}
}

func TestLocationLabel(t *testing.T) {
	e := sampleEvent()
	if got := e.LocationLabel(); got != "Toronto, ON" {
		t.Errorf("expected city and state, got %q", got)
	}

	// No state code, so the country stands in.
	e.Embedded.Venues[0].State.StateCode = ""
	if got := e.LocationLabel(); got != "Toronto, CA" {
		t.Errorf("expected city and country, got %q", got)
	}

	if got := (TMEvent{}).LocationLabel(); got != "" {
		t.Errorf("expected an empty label with no venue, got %q", got)
	}
}

func TestVenueAddress(t *testing.T) {
	if got := sampleEvent().VenueAddress(); got != "1 River Road" {
		t.Errorf("unexpected address: %q", got)
	}
	if got := (TMEvent{}).VenueAddress(); got != "" {
		t.Errorf("expected an empty address, got %q", got)
	}
}

func TestDescriptionTextPrefersDescriptionThenInfo(t *testing.T) {
	e := TMEvent{Description: "Full description", Info: "Info", PleaseNote: "Note"}
	if got := e.DescriptionText(); got != "Full description" {
		t.Errorf("description should win, got %q", got)
	}

	e.Description = "   "
	if got := e.DescriptionText(); got != "Info" {
		t.Errorf("info should be next, got %q", got)
	}

	e.Info = ""
	if got := e.DescriptionText(); got != "Note" {
		t.Errorf("pleaseNote should be last, got %q", got)
	}

	if got := (TMEvent{}).DescriptionText(); got != "" {
		t.Errorf("expected an empty description, got %q", got)
	}
}

func TestCategoryNamePrefersThePrimaryClassification(t *testing.T) {
	e := TMEvent{Classifications: []TMClassification{
		{Primary: false, Segment: TMNamed{Name: "Sports"}},
		{Primary: true, Segment: TMNamed{Name: "Music"}},
	}}
	if got := e.CategoryName(); got != "Music" {
		t.Errorf("expected the primary classification, got %q", got)
	}

	e.Classifications[1].Primary = false
	if got := e.CategoryName(); got != "Sports" {
		t.Errorf("expected the first named segment, got %q", got)
	}

	if got := (TMEvent{}).CategoryName(); got != "" {
		t.Errorf("expected an empty category, got %q", got)
	}
}

func TestToCardBuildsTheListingShape(t *testing.T) {
	card := sampleEvent().ToCard(CategoryMusic)

	if card.ID != "ev-1" || card.Name != "The Weekend Sound" {
		t.Errorf("identity fields wrong: %+v", card)
	}
	if card.DetailsURL != "/events/ev-1" {
		t.Errorf("unexpected details url: %q", card.DetailsURL)
	}
	if !card.HasDate {
		t.Error("expected HasDate to be true for a dated event")
	}
	if card.VenueName != "Sample Riverside Hall" || card.VenueLocation != "Toronto, ON" {
		t.Errorf("venue fields wrong: %+v", card)
	}
}

func TestToCardUsesThePlaceholderWhenNoImage(t *testing.T) {
	e := sampleEvent()
	e.Images = nil

	if got := e.ToCard(CategorySports).ImageURL; got != PlaceholderImage(CategorySports) {
		t.Errorf("expected the sports placeholder, got %q", got)
	}
}

func TestToDetailPointsAtOurOwnRedirectRoute(t *testing.T) {
	d := sampleEvent().ToDetail()

	if d.RedirectURL != "/redirect/ev-1" {
		t.Errorf("the ticket link must go through our route, got %q", d.RedirectURL)
	}
	if d.Category != "Music" || d.Timezone != "America/Toronto" {
		t.Errorf("detail fields wrong: %+v", d)
	}
	if d.Description != "An evening of live music." {
		t.Errorf("description should come from Info here, got %q", d.Description)
	}
}

func TestToEventCardsRespectsTheLimit(t *testing.T) {
	events := make([]TMEvent, 10)
	for i := range events {
		events[i] = TMEvent{ID: "e"}
	}

	if got := ToEventCards(events, CategoryMusic, 6); len(got) != 6 {
		t.Errorf("expected the list to be capped at 6, got %d", len(got))
	}
	if got := ToEventCards(events, CategoryMusic, 0); len(got) != 10 {
		t.Errorf("a limit of 0 should mean no cap, got %d", len(got))
	}
	if got := ToEventCards(nil, CategoryMusic, 6); got == nil {
		t.Error("expected an empty slice rather than nil")
	}
}
