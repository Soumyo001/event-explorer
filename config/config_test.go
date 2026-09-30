package config

import (
	"strings"
	"testing"
	"time"
)

func TestGetEnvFallsBackWhenUnsetOrBlank(t *testing.T) {
	t.Setenv("EE_TEST_VALUE", "")
	if got := getEnv("EE_TEST_VALUE", "fallback"); got != "fallback" {
		t.Errorf("expected the fallback, got %q", got)
	}

	t.Setenv("EE_TEST_VALUE", "   ")
	if got := getEnv("EE_TEST_VALUE", "fallback"); got != "fallback" {
		t.Errorf("whitespace should count as unset, got %q", got)
	}

	t.Setenv("EE_TEST_VALUE", "  real  ")
	if got := getEnv("EE_TEST_VALUE", "fallback"); got != "real" {
		t.Errorf("expected a trimmed value, got %q", got)
	}
}

func TestGetEnvInt(t *testing.T) {
	t.Setenv("EE_TEST_INT", "12")
	if got := getEnvInt("EE_TEST_INT", 6); got != 12 {
		t.Errorf("expected 12, got %d", got)
	}

	for _, bad := range []string{"", "abc", "0", "-3"} {
		t.Setenv("EE_TEST_INT", bad)
		if got := getEnvInt("EE_TEST_INT", 6); got != 6 {
			t.Errorf("%q should fall back to 6, got %d", bad, got)
		}
	}
}

func TestGetEnvDuration(t *testing.T) {
	t.Setenv("EE_TEST_DUR", "90s")
	if got := getEnvDuration("EE_TEST_DUR", time.Minute); got != 90*time.Second {
		t.Errorf("expected 90s, got %s", got)
	}

	for _, bad := range []string{"", "soon", "0s", "-5m"} {
		t.Setenv("EE_TEST_DUR", bad)
		if got := getEnvDuration("EE_TEST_DUR", time.Minute); got != time.Minute {
			t.Errorf("%q should fall back to 1m, got %s", bad, got)
		}
	}
}

func TestGetEnvList(t *testing.T) {
	fallback := []string{"default.com"}

	t.Setenv("EE_TEST_LIST", " Ticketmaster.com , , LIVENATION.com ")
	got := getEnvList("EE_TEST_LIST", fallback)
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %v", got)
	}
	if got[0] != "ticketmaster.com" || got[1] != "livenation.com" {
		t.Errorf("entries should be trimmed and lowercased, got %v", got)
	}

	t.Setenv("EE_TEST_LIST", "")
	if got := getEnvList("EE_TEST_LIST", fallback); len(got) != 1 {
		t.Errorf("an empty value should fall back, got %v", got)
	}

	t.Setenv("EE_TEST_LIST", " , , ")
	if got := getEnvList("EE_TEST_LIST", fallback); len(got) != 1 {
		t.Errorf("a list of blanks should fall back, got %v", got)
	}
}

func TestNewConfigReadsTheEnvironment(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "g-key")
	t.Setenv("TICKETMASTER_API_KEY", "tm-key")
	t.Setenv("CACHE_TTL", "2m")
	t.Setenv("HTTP_TIMEOUT", "3s")
	t.Setenv("EVENTS_PER_CATEGORY", "4")

	cfg := newConfig()

	if cfg.GoogleAPIKey != "g-key" || cfg.TicketmasterAPIKey != "tm-key" {
		t.Errorf("keys were not read: %+v", cfg)
	}
	if cfg.CacheTTL != 2*time.Minute || cfg.HTTPTimeout != 3*time.Second {
		t.Errorf("durations were not read: %+v", cfg)
	}
	if cfg.EventsPerCategory != 4 {
		t.Errorf("expected 4 events per category, got %d", cfg.EventsPerCategory)
	}
}

func TestNewConfigUsesDefaultsForUnsetValues(t *testing.T) {
	t.Setenv("GOOGLE_PLACES_BASE_URL", "")
	t.Setenv("TICKETMASTER_BASE_URL", "")
	t.Setenv("CACHE_TTL", "")
	t.Setenv("ALLOWED_TICKET_HOSTS", "")

	cfg := newConfig()

	if !strings.Contains(cfg.GooglePlacesBase, "googleapis.com") {
		t.Errorf("unexpected google base: %q", cfg.GooglePlacesBase)
	}
	if !strings.Contains(cfg.TicketmasterBase, "ticketmaster.com") {
		t.Errorf("unexpected ticketmaster base: %q", cfg.TicketmasterBase)
	}
	if cfg.CacheTTL != 5*time.Minute {
		t.Errorf("the assignment requires a five minute default, got %s", cfg.CacheTTL)
	}
	if len(cfg.AllowedTicketHosts) == 0 {
		t.Error("expected a default allowlist")
	}
}

func TestValidateReportsMissingKeys(t *testing.T) {
	empty := &Config{}
	err := empty.Validate()
	if err == nil {
		t.Fatal("expected an error when both keys are missing")
	}
	if !strings.Contains(err.Error(), "GOOGLE_API_KEY") ||
		!strings.Contains(err.Error(), "TICKETMASTER_API_KEY") {
		t.Errorf("both missing keys should be named: %v", err)
	}

	partial := &Config{GoogleAPIKey: "g"}
	if err := partial.Validate(); err == nil || strings.Contains(err.Error(), "GOOGLE_API_KEY") {
		t.Errorf("only the missing key should be named: %v", err)
	}

	complete := &Config{GoogleAPIKey: "g", TicketmasterAPIKey: "t"}
	if err := complete.Validate(); err != nil {
		t.Errorf("a complete config should validate, got %v", err)
	}
}

func TestGetConfigLoadsOnceAndSetConfigReplacesIt(t *testing.T) {
	custom := &Config{GoogleAPIKey: "injected"}
	SetConfig(custom)

	if got := GetConfig(); got != custom {
		t.Fatal("SetConfig should replace the active config")
	}

	SetConfig(nil)
	if GetConfig() == nil {
		t.Fatal("GetConfig should load a config when none is set")
	}
}
