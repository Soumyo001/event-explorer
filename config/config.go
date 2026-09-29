package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// whole .env config loaded for dynamic access
type Config struct {
	GoogleAPIKey       string
	GooglePlacesBase   string
	TicketmasterAPIKey string
	TicketmasterBase   string
	CacheTTL           time.Duration
	HTTPTimeout        time.Duration
	EventsPerCategory  int
	AllowedTicketHosts []string
}

var current *Config

func (c *Config) Validate() error {
	var missing []string

	if c.GoogleAPIKey == "" {
		missing = append(missing, "GOOGLE_API_KEY")
	}
	if c.TicketmasterAPIKey == "" {
		missing = append(missing, "TICKETMASTER_API_KEY")
	}
	if len(missing) > 0 {
		return errors.New("missing required environment variables: " + strings.Join(missing, ", "))
	}
	return nil
}

func newConfig() *Config {
	return &Config{
		GoogleAPIKey:       getEnv("GOOGLE_API_KEY", ""),
		GooglePlacesBase:   getEnv("GOOGLE_PLACES_BASE_URL", "https://places.googleapis.com"),
		TicketmasterAPIKey: getEnv("TICKETMASTER_API_KEY", ""),
		TicketmasterBase:   getEnv("TICKETMASTER_BASE_URL", "https://app.ticketmaster.com/discovery/v2"),
		CacheTTL:           getEnvDuration("CACHE_TTL", 5*time.Minute),
		HTTPTimeout:        getEnvDuration("HTTP_TIMEOUT", 10*time.Second),
		EventsPerCategory:  getEnvInt("EVENTS_PER_CATEGORY", 6),
		AllowedTicketHosts: getEnvList("ALLOWED_TICKET_HOSTS", []string{
			"ticketmaster.com", "www.ticketmaster.com",
			"ticketmaster.ca", "www.ticketmaster.ca",
		}),
	}
}

func LoadConfig() *Config {
	_ = godotenv.Load()
	current = newConfig()
	return current
}

func GetConfig() *Config {
	if current == nil {
		return LoadConfig()
	}
	return current
}

func SetConfig(c *Config) { current = c }

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, err := strconv.Atoi(getEnv(key, "")); err == nil && v > 0 {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, err := time.ParseDuration(getEnv(key, "")); err == nil && v > 0 {
		return v
	}
	return fallback
}

func getEnvList(key string, fallback []string) []string {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, strings.ToLower(trimmed))
		}
	}

	if len(out) == 0 {
		return fallback
	}
	return out
}
