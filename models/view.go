package models

import (
	"net/url"
	"strings"
)

type Category string

const (
	CategoryMusic  Category = "Music"
	CategorySports Category = "Sports"
)

const GoogleAttribution = "Powered by Google"

var AllCategories = []Category{CategoryMusic, CategorySports}

func (c Category) String() string { return string(c) }
func (c Category) Slug() string   { return strings.ToLower(string(c)) }
func (c Category) Valid() bool {
	return c == CategoryMusic || c == CategorySports
}

func ParseCategory(s string) (Category, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "music":
		return CategoryMusic, true
	case "sports":
		return CategorySports, true
	default:
		return "", false
	}
}

func ParseCategoryOrEmpty(s string) Category {
	c, _ := ParseCategory(s)
	return c
}

func PlaceholderImage(c Category) string {
	switch c {
	case CategoryMusic:
		return "/static/img/placeholder-music.svg"
	case CategorySports:
		return "/static/img/placeholder-sports.svg"
	default:
		return "/static/img/placeholder-event.svg"
	}
}

// location view model
type CitySuggestion struct {
	PlaceID       string `json:"placeId"`
	Text          string `json:"text"`
	MainText      string `json:"mainText"`
	SecondaryText string `json:"secondaryText"`
}
type AutocompleteAPIResponse struct {
	Suggestions []CitySuggestion `json:"suggestions"`
	Attribution string           `json:"attribution"`
}
type SelectedCity struct {
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
}
type APIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// event view model
type EventDetail struct {
	ID            string
	Name          string
	ImageURL      string
	DateLabel     string
	TimeLabel     string
	Timezone      string
	VenueName     string
	VenueAddress  string
	VenueLocation string
	Description   string
	Category      string
	RedirectURL   string
}

func (e EventDetail) HasDescription() bool {
	return e.Description != ""
}

// one section per category
type EventCard struct {
	ID            string
	Name          string
	ImageURL      string
	DateLabel     string
	TimeLabel     string
	HasDate       bool
	VenueName     string
	VenueLocation string
	DetailsURL    string
}
type EventSection struct {
	Category Category
	Title    string
	Events   []EventCard
	Err      string
	Cached   bool
}

func (e EventSection) HasError() bool {
	return e.Err != ""
}
func (e EventSection) IsEmpty() bool {
	return e.Err == "" && len(e.Events) == 0
}
func (e EventSection) Count() int {
	return len(e.Events)
}

func NewEventSection(c Category, events []EventCard, cached bool) EventSection {
	if events == nil {
		events = []EventCard{}
	}
	return EventSection{
		Category: c,
		Title:    c.String(),
		Events:   events,
		Cached:   cached,
	}
}

func NewFailedEventSection(c Category, message string) EventSection {
	return EventSection{
		Category: c,
		Title:    c.String(),
		Err:      message,
		Events:   []EventCard{},
	}
}

type SampleCity struct {
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Label       string `json:"label"`
	Note        string `json:"note"`
	HasEvents   bool   `json:"hasEvents"`
}

func (s SampleCity) Valid() bool {
	return strings.TrimSpace(s.City) != "" && len(strings.TrimSpace(s.CountryCode)) == 2
}

func (s SampleCity) DisplayLabel() string {
	if strings.TrimSpace(s.Label) != "" {
		return s.Label
	}
	return s.City
}

func (s SampleCity) ListingURL() string {
	q := url.Values{}
	q.Set("city", s.City)
	q.Set("countryCode", s.CountryCode)
	return "/events?" + q.Encode()
}

type SampleCitySet struct {
	Note   string       `json:"note"`
	Cities []SampleCity `json:"cities"`
}

func (s SampleCitySet) ValidCities() []SampleCity {
	out := make([]SampleCity, 0, len(s.Cities))
	for _, c := range s.Cities {
		if !c.Valid() {
			continue
		}
		c.City = strings.TrimSpace(c.City)
		c.CountryCode = strings.ToUpper(strings.TrimSpace(c.CountryCode))
		out = append(out, c)
	}
	return out
}

// template structs (page data)
type HomePageData struct {
	Title        string
	Attribution  string
	SampleCities []SampleCity
}

func (d HomePageData) HasSampleCities() bool {
	return len(d.SampleCities) > 0
}

type ListingPageData struct {
	Title       string
	City        string
	CountryCode string
	Sections    []EventSection
}

func (p ListingPageData) AllFailed() bool {
	if len(p.Sections) == 0 {
		return true
	}

	for _, s := range p.Sections {
		if !s.HasError() {
			return false
		}
	}
	return true
}

type DetailsPageData struct {
	Title   string
	Event   EventDetail
	BackURL string
}

type ErrorPageData struct {
	Title      string
	StatusCode int
	Heading    string
	Message    string
	BackURL    string
}
