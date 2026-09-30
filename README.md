# Event Explorer

City search and event discovery built with Beego (Go), HTML, CSS and vanilla
JavaScript. Pick a city, browse Music and Sports events, open an event and
continue to the ticket provider.

## Requirements

- Go 1.21 or newer
- A Google Places API (New) key, on a project with billing enabled
- A Ticketmaster Discovery API v2 consumer key

## Setup

```bash
git clone <repository-url>
cd eventexplorer
go mod download

cp .env.example .env
# open .env and add your two API keys
```

API keys are read from `.env`, which is gitignored and never committed.
`.env.example` lists every variable with safe defaults.

## Running

```bash
go run main.go
# or, with live reload:
bee run
```

The site is served at http://localhost:8080

## Running the tests

All tests use mocked provider responses and pass without API keys.

```bash
go test ./...                                          # all packages
go test ./... -race                                    # race detector
go test ./... -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out | tail -n 1           # total coverage
go tool cover -html=coverage.out -o coverage.html      # line-by-line report
```

Current coverage: **91.0%** of statements. See `TEST_OUTPUT.md` for the full
passing run.

## Project structure

```
eventexplorer/
├── routers/router.go     route registration
├── controllers/          request handling and rendering
│   ├── page.go           /  /events  /events/:eventId   (SSR pages)
│   ├── location.go       /api/locations/*               (JSON)
│   ├── redirect.go       /redirect/:eventId             (302)
│   ├── cache.go          /api/cache*                    (invalidation)
│   └── base.go           shared JSON and error-page helpers
├── services/             business logic and provider calls
│   ├── google.go         Google Places autocomplete and place details
│   ├── ticketmaster.go   Ticketmaster event list and event lookup
│   ├── event.go          concurrent category fetches, caching
│   ├── ticketlink.go     ticket URL hostname validation
│   └── samplecity.go     ready-to-test city loader
├── models/               request, response and view structs
├── cache/cache.go        shared mutex-protected TTL cache
├── views/                Beego templates
│   ├── partials/         shared header and footer
│   └── home.tpl  listing.tpl  details.tpl  error.tpl
├── static/               css, js, placeholder images
├── config/config.go      environment configuration
├── utils/                shared HTTP client, credential redaction
└── data/sample_cities.json
```

## Routes

| Method | Route | Response |
|---|---|---|
| GET | `/` | home.tpl, city search |
| GET | `/events?city=&countryCode=` | listing.tpl, Music and Sports |
| GET | `/events/:eventId` | details.tpl, event details |
| GET | `/api/locations/autocomplete?input=&sessionToken=` | JSON suggestions |
| GET | `/api/locations/:placeId?sessionToken=` | JSON city and countryCode |
| GET | `/redirect/:eventId` | HTTP 302 to the validated ticket URL |
| GET | `/api/cache` | JSON list of cached keys |
| DELETE | `/api/cache` | clear the whole cache |
| DELETE | `/api/cache/city?city=&countryCode=` | clear one city |
| DELETE | `/api/cache/event/:eventId` | clear one cached event |
| DELETE | `/api/cache/key/:cacheKey` | clear one key |

## Where the Go concepts live

| Concept | File |
|---|---|
| Goroutines and channels | `services/event.go` — `ListByCity` |
| Shared in-memory cache | `cache/cache.go`, used in `services/event.go` |
| Ticket link validation | `services/ticketlink.go` |
| MVC separation | `routers/` → `controllers/` → `services/` → `models/` → `views/` |

## Ready-to-test cities

`data/sample_cities.json` holds cities verified against the Ticketmaster
catalogue, linked from the home page. Toronto, London and New York return
events; Dhaka returns none, which shows the empty state.

## Notes

- Google responses are never cached, as the assignment requires.
- Beego logs `open conf/app.conf: no such file or directory` at debug level when running `go test` from a package directory. It is harmless.