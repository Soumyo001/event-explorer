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

	// Cache inspection and invalidation
	beego.Router("/api/cache", &controllers.CacheController{}, "get:Keys")
	beego.Router("/api/cache", &controllers.CacheController{}, "delete:ClearAll")
	beego.Router("/api/cache/city", &controllers.CacheController{}, "delete:ClearCity")
	beego.Router("/api/cache/event/:eventId", &controllers.CacheController{}, "delete:ClearEvent")
	beego.Router("/api/cache/key/:cacheKey", &controllers.CacheController{}, "delete:ClearKey")

	// API routes
	beego.Router("/api/locations/autocomplete", &controllers.LocationController{}, "get:Autocomplete")
	beego.Router("/api/locations/:placeId", &controllers.LocationController{}, "get:PlaceDetails")
}
