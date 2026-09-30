package routers

import (
	"eventexplorer/controllers"

	beego "github.com/beego/beego/v2/server/web"
)

func init() {
	// Page routes
	beego.Router("/", &controllers.PageController{}, "get:Home")

	// API routes
	beego.Router("/api/locations/autocomplete", &controllers.LocationController{}, "get:Autocomplete")
	beego.Router("/api/locations/:placeId", &controllers.LocationController{}, "get:PlaceDetails")
}
