package controllers

import (
	"eventexplorer/models"
	"eventexplorer/services"
)

type LocationController struct {
	BaseController
	Locations services.LocationService
}

func (c *LocationController) Prepare() {
	if c.Locations == nil {
		c.Locations = services.DefaultLocationService()
	}
}

func (c *LocationController) Autocomplete() {
	input := c.GetString("input")
	sessionToken := c.GetString("sessionToken")

	suggestions, err := c.Locations.Autocomplete(c.Ctx.Request.Context(), input, sessionToken)
	if err != nil {
		c.FailJSON(err)
		return
	}

	c.JSONData(models.AutocompleteAPIResponse{
		Suggestions: suggestions,
		Attribution: models.GoogleAttribution,
	})
}

func (c *LocationController) PlaceDetails() {
	placeID := c.Ctx.Input.Param(":placeId")
	sessionToken := c.GetString("sessionToken")

	city, err := c.Locations.PlaceDetails(c.Ctx.Request.Context(), placeID, sessionToken)
	if err != nil {
		c.FailJSON(err)
		return
	}
	c.JSONData(city)
}
