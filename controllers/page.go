package controllers

import (
	"eventexplorer/models"
	"eventexplorer/services"
	"net/url"
	"strings"
)

type PageController struct {
	BaseController
	Events services.EventService
}

func (p *PageController) Prepare() {
	if p.Events == nil {
		p.Events = services.DefaultEventService()
	}
	if p.Events == nil {
		panic("event service is not initialised")
	}
}

func (p *PageController) backToListing() string {
	ref := p.Ctx.Request.Referer()
	if ref == "" {
		return "/"
	}

	u, err := url.Parse(ref)
	if err != nil || u.Path != "/events" {
		return "/"
	}

	city := strings.TrimSpace(u.Query().Get("city"))
	country := strings.TrimSpace(u.Query().Get("countryCode"))
	if city == "" || country == "" {
		return "/"
	}
	queryParam := url.Values{}
	queryParam.Set("city", city)
	queryParam.Set("countryCode", country)
	return "/events?" + queryParam.Encode()
}

func (p *PageController) Home() {
	p.Data["Page"] = models.HomePageData{
		Title:        "Event Explorer",
		Attribution:  models.GoogleAttribution,
		SampleCities: services.SampleCities(),
	}
	p.TplName = "home.tpl"
}

func (p *PageController) Listing() {
	city := strings.TrimSpace(p.GetString("city"))
	countryCode := strings.ToUpper(strings.TrimSpace(p.GetString("countryCode")))

	if city == "" || countryCode == "" {
		p.RenderError(400, "Choose a city first", "Pick a city from the search suggestions to see what is on.", "/")
		return
	}

	sections := p.Events.ListByCity(p.Ctx.Request.Context(), city, countryCode)

	p.Data["Page"] = models.ListingPageData{
		Title:       "Events in " + city,
		City:        city,
		CountryCode: countryCode,
		Sections:    sections,
	}
	p.TplName = "listing.tpl"
}

func (p *PageController) Details() {
	eventID := strings.TrimSpace(p.Ctx.Input.Param(":eventId"))

	event, err := p.Events.GetEvent(p.Ctx.Request.Context(), eventID)
	if err != nil {
		p.FailPage(err, p.backToListing())
		return
	}

	p.Data["Page"] = models.DetailsPageData{
		Title:   event.Name,
		Event:   event,
		BackURL: p.backToListing(),
	}
	p.TplName = "details.tpl"
}
