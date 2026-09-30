package controllers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"eventexplorer/cache"
	"eventexplorer/models"

	beego "github.com/beego/beego/v2/server/web"
)

func cacheApp(store *cache.Cache) *beego.HttpServer {
	app := newApp()
	app.Router("/api/cache", &CacheController{Store: store}, "get:Keys")
	app.Router("/api/cache", &CacheController{Store: store}, "delete:ClearAll")
	app.Router("/api/cache/city", &CacheController{Store: store}, "delete:ClearCity")
	app.Router("/api/cache/event/:eventId", &CacheController{Store: store}, "delete:ClearEvent")
	app.Router("/api/cache/key/:cacheKey", &CacheController{Store: store}, "delete:ClearKey")
	return app
}

func populatedCache() *cache.Cache {
	c := cache.New(time.Minute)
	c.Set(cache.EventsKey("Toronto", "CA", "Music"), []models.TMEvent{{ID: "m1"}})
	c.Set(cache.EventsKey("Toronto", "CA", "Sports"), []models.TMEvent{{ID: "s1"}})
	c.Set(cache.EventKey("ev-1"), models.TMEvent{ID: "ev-1"})
	return c
}

func TestCacheKeysListsWhatIsHeld(t *testing.T) {
	rec := serve(t, cacheApp(populatedCache()), http.MethodGet, "/api/cache")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body models.CacheKeysResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response was not valid json: %v", err)
	}
	if body.Count != 3 || len(body.Keys) != 3 {
		t.Errorf("expected three live entries, got %+v", body)
	}
	if body.TTL == "" {
		t.Error("the ttl should be reported")
	}
}

func TestCacheClearAllEmptiesTheStore(t *testing.T) {
	store := populatedCache()

	rec := serve(t, cacheApp(store), http.MethodDelete, "/api/cache")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body models.CacheInvalidateResponse
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Scope != "all" || body.Removed != 3 {
		t.Errorf("unexpected response: %+v", body)
	}
	if store.Len() != 0 {
		t.Errorf("the cache should be empty, %d entries remain", store.Len())
	}
}

func TestCacheClearKeyRemovesOneEntry(t *testing.T) {
	store := populatedCache()

	rec := serve(t, cacheApp(store), http.MethodDelete, "/api/cache/key/events:toronto:CA:music")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	if _, ok := store.Get(cache.EventsKey("Toronto", "CA", "Music")); ok {
		t.Error("the key should have been removed")
	}
	if _, ok := store.Get(cache.EventsKey("Toronto", "CA", "Sports")); !ok {
		t.Error("the other category should survive")
	}
}

func TestCacheClearKeyUnknownIs404(t *testing.T) {
	rec := serve(t, cacheApp(populatedCache()), http.MethodDelete, "/api/cache/key/nosuchkey")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestCacheClearCityRemovesBothCategories(t *testing.T) {
	store := populatedCache()

	rec := serve(t, cacheApp(store), http.MethodDelete, "/api/cache/city?city=Toronto&countryCode=CA")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body models.CacheInvalidateResponse
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Removed != 2 {
		t.Errorf("expected both categories removed, got %d", body.Removed)
	}

	if _, ok := store.Get(cache.EventKey("ev-1")); !ok {
		t.Error("clearing a city should not drop cached event details")
	}
}

func TestCacheClearCityRequiresBothParameters(t *testing.T) {
	cases := []string{
		"/api/cache/city?city=Toronto",
		"/api/cache/city?countryCode=CA",
		"/api/cache/city",
	}

	for _, target := range cases {
		rec := serve(t, cacheApp(populatedCache()), http.MethodDelete, target)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: expected 400, got %d", target, rec.Code)
		}
	}
}

func TestCacheClearEventRemovesTheDetailsEntry(t *testing.T) {
	store := populatedCache()

	rec := serve(t, cacheApp(store), http.MethodDelete, "/api/cache/event/ev-1")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body)
	}
	if _, ok := store.Get(cache.EventKey("ev-1")); ok {
		t.Error("the cached event should have been removed")
	}

	if store.Len() != 2 {
		t.Errorf("expected the two listing entries to remain, got %d", store.Len())
	}
}

func TestCacheClearEventUnknownIs404(t *testing.T) {
	rec := serve(t, cacheApp(populatedCache()), http.MethodDelete, "/api/cache/event/nope")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
