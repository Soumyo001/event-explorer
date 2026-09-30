package routers

import (
	"eventexplorer/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	// Page routes
	beego.Router("/", &controllers.PageController{}, "get:Home")
	beego.Router("/events", &controllers.PageController{}, "get:Listing")
	beego.Router("/events/:eventId", &controllers.PageController{}, "get:Details")

	// redirect route
	beego.Router("/redirect/:eventId", &controllers.RedirectController{}, "get:Ticket")

	// API routes
	beego.Router("/api/locations/autocomplete", &controllers.LocationController{}, "get:Autocomplete")
	beego.Router("/api/locations/:placeId", &controllers.LocationController{}, "get:PlaceDetails")
}
