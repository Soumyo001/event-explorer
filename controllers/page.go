package controllers

import (
	"eventexplorer/models"
	"eventexplorer/services"
)

type PageController struct {
	BaseController
}

func (p *PageController) Home() {
	p.Data["Page"] = models.HomePageData{
		Title:        "Event Explorer",
		Attribution:  models.GoogleAttribution,
		SampleCities: services.SampleCities(),
	}
	p.TplName = "home.tpl"
}
