package controllers

import (
	"fmt"
	"strings"

	"eventexplorer/cache"
	"eventexplorer/models"

	"github.com/beego/beego/v2/core/logs"
)

type CacheController struct {
	BaseController
	Store *cache.Cache
}

func (c *CacheController) Prepare() {
	if c.Store == nil {
		c.Store = cache.Shared()
	}
	if c.Store == nil {
		panic("cache store is not initialised")
	}
}

func (c *CacheController) Keys() {
	keys := c.Store.Keys()
	stats := c.Store.Stats()

	c.JSONData(models.CacheKeysResponse{
		Count:  len(keys),
		Keys:   keys,
		Hits:   stats.Hits,
		Misses: stats.Misses,
		TTL:    c.Store.TTL().String(),
	})
}

func (c *CacheController) ClearAll() {
	removed := c.Store.Clear()
	logs.Info("cache invalidation: all entries cleared (%d removed)", removed)

	c.JSONData(models.CacheInvalidateResponse{
		Scope:   "all",
		Removed: removed,
		Message: fmt.Sprintf("Cleared the whole cache, %d entries removed.", removed),
	})
}

func (c *CacheController) ClearKey() {
	key := strings.TrimSpace(c.Ctx.Input.Param(":cacheKey"))
	if key == "" {
		c.JSONError(400, "invalid_input", "A cache key is required.")
		return
	}

	if !c.Store.Delete(key) {
		c.JSONError(404, "not_found", "No live cache entry for that key.")
		return
	}

	c.JSONData(models.CacheInvalidateResponse{
		Scope:   "key",
		Target:  key,
		Removed: 1,
		Message: "Cache entry removed.",
	})
}

func (c *CacheController) ClearCity() {
	city := strings.TrimSpace(c.GetString("city"))
	countryCode := strings.TrimSpace(c.GetString("countryCode"))

	if city == "" || countryCode == "" {
		c.JSONError(400, "invalid_input", "Both city and countryCode are required.")
		return
	}

	prefix := cache.EventsKey(city, countryCode, "")
	removed := c.Store.DeletePrefix(prefix)
	logs.Info("cache invalidation: %s/%s cleared (%d removed)", city, countryCode, removed)

	c.JSONData(models.CacheInvalidateResponse{
		Scope:   "city",
		Target:  city + "/" + strings.ToUpper(countryCode),
		Removed: removed,
		Message: fmt.Sprintf("Removed %d cached entries for this city.", removed),
	})
}

func (c *CacheController) ClearEvent() {
	eventID := strings.TrimSpace(c.Ctx.Input.Param(":eventId"))
	if eventID == "" {
		c.JSONError(400, "invalid_input", "An event id is required.")
		return
	}

	key := cache.EventKey(eventID)
	if !c.Store.Delete(key) {
		c.JSONError(404, "not_found", "That event is not currently cached.")
		return
	}

	logs.Info("cache invalidation: event %q cleared", eventID)

	c.JSONData(models.CacheInvalidateResponse{
		Scope:   "event",
		Target:  eventID,
		Removed: 1,
		Message: "Cached event details removed.",
	})
}
