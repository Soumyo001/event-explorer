package models

import (
	"strings"
	"testing"
)

func TestCategoryHelpers(t *testing.T) {
	if CategoryMusic.String() != "Music" || CategoryMusic.Slug() != "music" {
		t.Error("Music category helpers are wrong")
	}
	if !CategoryMusic.Valid() || !CategorySports.Valid() {
		t.Error("both supported categories should be valid")
	}
	if Category("Theatre").Valid() {
		t.Error("an unsupported category must not be valid")
	}
}

func TestParseCategory(t *testing.T) {
	cases := map[string]Category{
		"music":    CategoryMusic,
		"  MUSIC ": CategoryMusic,
		"Sports":   CategorySports,
	}

	for input, want := range cases {
		got, ok := ParseCategory(input)
		if !ok || got != want {
			t.Errorf("ParseCategory(%q) = %q, %v", input, got, ok)
		}
	}

	if _, ok := ParseCategory("theatre"); ok {
		t.Error("an unknown category should not parse")
	}
	if got := ParseCategoryOrEmpty("theatre"); got != "" {
		t.Errorf("expected an empty category, got %q", got)
	}
}

func TestPlaceholderImagePerCategory(t *testing.T) {
	if !strings.Contains(PlaceholderImage(CategoryMusic), "music") {
		t.Error("expected the music placeholder")
	}
	if !strings.Contains(PlaceholderImage(CategorySports), "sports") {
		t.Error("expected the sports placeholder")
	}
	if !strings.Contains(PlaceholderImage(""), "event") {
		t.Error("expected the neutral placeholder for an unknown category")
	}
}

func TestEventDetailHasDescription(t *testing.T) {
	if (EventDetail{Description: "x"}).HasDescription() != true {
		t.Error("expected true when a description is present")
	}
	if (EventDetail{}).HasDescription() != false {
		t.Error("expected false when the description is empty")
	}
}

func TestNewEventSectionNeverHoldsANilSlice(t *testing.T) {
	s := NewEventSection(CategoryMusic, nil, false)

	if s.Events == nil {
		t.Fatal("events should be an empty slice, not nil")
	}
	if s.Title != "Music" {
		t.Errorf("unexpected title: %q", s.Title)
	}
	if !s.IsEmpty() {
		t.Error("a section with no events and no error is empty")
	}
	if s.HasError() {
		t.Error("a successful section has no error")
	}
}

func TestNewFailedEventSection(t *testing.T) {
	s := NewFailedEventSection(CategorySports, "sports events could not be loaded")

	if !s.HasError() {
		t.Error("a failed section must report an error")
	}
	if s.IsEmpty() {
		t.Error("a failed section is not the same as an empty one")
	}
	if s.Count() != 0 {
		t.Errorf("expected no events, got %d", s.Count())
	}
}

func TestEventSectionCount(t *testing.T) {
	s := NewEventSection(CategoryMusic, []EventCard{{ID: "a"}, {ID: "b"}}, true)

	if s.Count() != 2 {
		t.Errorf("expected 2 events, got %d", s.Count())
	}
	if !s.Cached {
		t.Error("the cached flag should be carried through")
	}
}

func TestListingPageDataAllFailed(t *testing.T) {
	both := ListingPageData{Sections: []EventSection{
		NewFailedEventSection(CategoryMusic, "x"),
		NewFailedEventSection(CategorySports, "y"),
	}}
	if !both.AllFailed() {
		t.Error("expected AllFailed when every section errored")
	}

	// partial-failure case
	partial := ListingPageData{Sections: []EventSection{
		NewEventSection(CategoryMusic, []EventCard{{ID: "a"}}, false),
		NewFailedEventSection(CategorySports, "y"),
	}}
	if partial.AllFailed() {
		t.Error("one working section means the page has not wholly failed")
	}

	if !(ListingPageData{}).AllFailed() {
		t.Error("no sections at all counts as a failure")
	}
}

func TestSampleCityValidation(t *testing.T) {
	good := SampleCity{City: "Toronto", CountryCode: "CA"}
	if !good.Valid() {
		t.Error("a city with a two-letter code is valid")
	}

	bad := map[string]SampleCity{
		"no city":    {CountryCode: "CA"},
		"blank city": {City: "   ", CountryCode: "CA"},
		"short code": {City: "Toronto", CountryCode: "C"},
		"long code":  {City: "Toronto", CountryCode: "CAN"},
		"no code":    {City: "Toronto"},
	}
	for name, c := range bad {
		if c.Valid() {
			t.Errorf("%s: expected this entry to be rejected", name)
		}
	}
}

func TestSampleCityDisplayLabelAndURL(t *testing.T) {
	withLabel := SampleCity{City: "New York", CountryCode: "US", Label: "NYC"}
	if got := withLabel.DisplayLabel(); got != "NYC" {
		t.Errorf("expected the label, got %q", got)
	}

	withoutLabel := SampleCity{City: "New York", CountryCode: "US"}
	if got := withoutLabel.DisplayLabel(); got != "New York" {
		t.Errorf("expected a fallback to the city name, got %q", got)
	}

	want := "/events?city=New+York&countryCode=US"
	if got := withoutLabel.ListingURL(); got != want {
		t.Errorf("listing url should be escaped:\n got %q\nwant %q", got, want)
	}
}

func TestValidCitiesFiltersAndNormalises(t *testing.T) {
	set := SampleCitySet{Cities: []SampleCity{
		{City: "  Toronto ", CountryCode: " ca "},
		{City: "", CountryCode: "GB"},
		{City: "Dhaka", CountryCode: "BD", HasEvents: false},
	}}

	got := set.ValidCities()
	if len(got) != 2 {
		t.Fatalf("expected the invalid entry to be dropped, got %d", len(got))
	}
	if got[0].City != "Toronto" || got[0].CountryCode != "CA" {
		t.Errorf("entry was not normalised: %+v", got[0])
	}
	if got[1].HasEvents {
		t.Error("the empty-result city should keep HasEvents false")
	}

	if (SampleCitySet{}).ValidCities() == nil {
		t.Error("expected an empty slice rather than nil")
	}
}

func TestHomePageDataHasSampleCities(t *testing.T) {
	if (HomePageData{}).HasSampleCities() {
		t.Error("expected false with no sample cities")
	}
	withCities := HomePageData{SampleCities: []SampleCity{{City: "Toronto", CountryCode: "CA"}}}
	if !withCities.HasSampleCities() {
		t.Error("expected true when sample cities are present")
	}
}
