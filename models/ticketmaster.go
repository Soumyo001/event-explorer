package models

import (
	"strings"
	"time"
)

type TMNamed struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type TMState struct {
	Name      string `json:"name"`
	StateCode string `json:"stateCode"`
}
type TMCountry struct {
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
}
type TMAddress struct {
	Line1 string `json:"line1"`
	Line2 string `json:"line2"`
}
type TMImage struct {
	Ratio    string `json:"ratio"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Fallback bool   `json:"fallback"`
}

// TM date schema
type TMStart struct {
	LocalDate      string `json:"localDate"`
	LocalTime      string `json:"localTime"`
	DateTime       string `json:"dateTime"`
	DateTBD        bool   `json:"dateTBD"`
	DateTBA        bool   `json:"dateTBA"`
	TimeTBA        bool   `json:"timeTBA"`
	NoSpecificTime bool   `json:"noSpecificTime"`
}
type TMStatus struct {
	Code string `json:"code"`
}
type TMDates struct {
	Start            TMStart  `json:"start"`
	Timezone         string   `json:"timezone"`
	Status           TMStatus `json:"status"`
	SpanMultipleDays bool     `json:"spanMultipleDays"`
}

// TM Venue schema
type TMVenue struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	City    TMNamed   `json:"city"`
	State   TMState   `json:"state"`
	Country TMCountry `json:"country"`
	Address TMAddress `json:"address"`
}
type TMEventEmbedded struct {
	Venues []TMVenue `json:"venues"`
}

// TM Classification schema
type TMClassification struct {
	Primary bool    `json:"primary"`
	Segment TMNamed `json:"segment"`
	Genre   TMNamed `json:"genre"`
}

// TM Event schema
type TMEvent struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Type            string             `json:"type"`
	URL             string             `json:"url"`
	Locale          string             `json:"locale"`
	Images          []TMImage          `json:"images"`
	Dates           TMDates            `json:"dates"`
	Classifications []TMClassification `json:"classifications"`
	Info            string             `json:"info"`
	PleaseNote      string             `json:"pleaseNote"`
	Description     string             `json:"description"`
	Embedded        TMEventEmbedded    `json:"_embedded"`
}

// out _embedded block of a list response
type TMEventsEmbedded struct {
	Events []TMEvent `json:"events"`
}

// TM page schema
type TMPage struct {
	Size          int `json:"size"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
	Number        int `json:"number"`
}

// response schema
type TMEventsResponse struct {
	Embedded TMEventsEmbedded `json:"_embedded"`
	Page     TMPage           `json:"page"`
}

func (r TMEventsResponse) Events() []TMEvent {
	if r.Embedded.Events == nil {
		return []TMEvent{}
	}
	return r.Embedded.Events
}

// error schema
type TMError struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}
type TMFault struct {
	FaultString string `json:"faultstring"`
}
type TMErrorResponse struct {
	Errors []TMError `json:"errors"`
	Fault  *TMFault  `json:"fault,omitempty"`
}

func (e TMErrorResponse) Message() string {
	if len(e.Errors) > 0 && e.Errors[0].Detail != "" {
		return e.Errors[0].Detail
	}
	if e.Fault != nil && e.Fault.FaultString != "" {
		return e.Fault.FaultString
	}
	return "unknown ticketmaster error"
}

func (e TMEvent) BestImageURL() string {
	var best TMImage
	for _, img := range e.Images {
		if img.URL == "" || img.Fallback {
			continue
		}
		if img.Ratio == "16_9" && img.Width >= 640 {
			if best.Ratio != "16_9" || img.Width < best.Width || best.Width == 0 {
				best = img
			}
			continue
		}
		if best.Ratio != "16_9" && img.Width > best.Width {
			best = img
		}
	}
	return best.URL
}

func (e TMEvent) PrimaryVenue() TMVenue {
	if len(e.Embedded.Venues) == 0 {
		return TMVenue{}
	}
	return e.Embedded.Venues[0]
}

func (e TMEvent) StartTime() (time.Time, bool) {
	s := e.Dates.Start
	if s.DateTBA || s.DateTBD || s.LocalDate == "" {
		return time.Time{}, false
	}
	if s.LocalTime != "" && !s.TimeTBA && !s.NoSpecificTime {
		if t, err := time.Parse("2006-01-02 15:04:05", s.LocalDate+" "+s.LocalTime); err == nil {
			return t, true
		}
	}
	if t, err := time.Parse("2006-01-02", s.LocalDate); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func (e TMEvent) DateLabel() string {
	t, ok := e.StartTime()
	if !ok {
		return "Date TBA"
	}
	return t.Format("Mon, 02 Jan 2006")
}

func (e TMEvent) TimeLabel() string {
	s := e.Dates.Start
	if s.TimeTBA || s.NoSpecificTime || s.LocalTime == "" {
		return ""
	}
	t, err := time.Parse("15:04:05", s.LocalTime)
	if err != nil {
		return ""
	}
	return t.Format("3:04 PM")
}

func (e TMEvent) LocationLabel() string {
	v := e.PrimaryVenue()
	parts := make([]string, 0, 2)
	if v.City.Name != "" {
		parts = append(parts, v.City.Name)
	}
	switch {
	case v.State.StateCode != "":
		parts = append(parts, v.State.StateCode)
	case v.Country.CountryCode != "":
		parts = append(parts, v.Country.CountryCode)
	}
	return strings.Join(parts, ", ")
}

func (e TMEvent) VenueAddress() string {
	return strings.TrimSpace(e.PrimaryVenue().Address.Line1)
}

func (e TMEvent) DescriptionText() string {
	for _, candidate := range []string{e.Description, e.Info, e.PleaseNote} {
		if t := strings.TrimSpace(candidate); t != "" {
			return t
		}
	}
	return ""
}

func (e TMEvent) CategoryName() string {
	for _, c := range e.Classifications {
		if c.Primary && c.Segment.Name != "" {
			return c.Segment.Name
		}
	}
	for _, c := range e.Classifications {
		if c.Segment.Name != "" {
			return c.Segment.Name
		}
	}
	return ""
}

// map response to view model
func (e TMEvent) ToCard(category Category) EventCard {
	image := e.BestImageURL()
	if image == "" {
		image = PlaceholderImage(category)
	}
	_, hasDate := e.StartTime()
	return EventCard{
		ID:            e.ID,
		Name:          e.Name,
		ImageURL:      image,
		DateLabel:     e.DateLabel(),
		TimeLabel:     e.TimeLabel(),
		HasDate:       hasDate,
		VenueName:     e.PrimaryVenue().Name,
		VenueLocation: e.LocationLabel(),
		DetailsURL:    "/events/" + e.ID,
	}
}

func (e TMEvent) ToDetail() EventDetail {
	category := ParseCategoryOrEmpty(e.CategoryName())
	image := e.BestImageURL()
	if image == "" {
		image = PlaceholderImage(category)
	}
	return EventDetail{
		ID:            e.ID,
		Name:          e.Name,
		ImageURL:      image,
		DateLabel:     e.DateLabel(),
		TimeLabel:     e.TimeLabel(),
		Timezone:      e.Dates.Timezone,
		VenueName:     e.PrimaryVenue().Name,
		VenueAddress:  e.VenueAddress(),
		VenueLocation: e.LocationLabel(),
		Description:   e.DescriptionText(),
		Category:      e.CategoryName(),
		RedirectURL:   "/redirect/" + e.ID,
	}
}

func ToEventCards(events []TMEvent, category Category, limit int) []EventCard {
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	cards := make([]EventCard, 0, len(events))
	for _, e := range events {
		cards = append(cards, e.ToCard(category))
	}
	return cards
}
