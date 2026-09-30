package controllers

import (
	"eventexplorer/models"
	"eventexplorer/services"
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
