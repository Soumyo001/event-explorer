package services

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempJSON(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample_cities.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("could not write the fixture: %v", err)
	}
	return path
}

func TestLoadSampleCitiesReadsTheFile(t *testing.T) {
	path := writeTempJSON(t, `{
		"note": "test data",
		"cities": [
			{"city":"Toronto","countryCode":"CA","label":"Toronto","hasEvents":true},
			{"city":"Dhaka","countryCode":"BD","label":"Dhaka","hasEvents":false}
		]
	}`)

	got := loadSampleCities(path)
	if len(got) != 2 {
		t.Fatalf("expected 2 cities, got %d", len(got))
	}
	if got[0].City != "Toronto" || got[0].CountryCode != "CA" {
		t.Errorf("unexpected first city: %+v", got[0])
	}

	if got[1].HasEvents {
		t.Error("Dhaka should be marked as having no events")
	}
}

func TestLoadSampleCitiesMissingFile(t *testing.T) {
	got := loadSampleCities(filepath.Join(t.TempDir(), "absent.json"))
	if got == nil {
		t.Fatal("expected an empty slice rather than nil")
	}
	if len(got) != 0 {
		t.Errorf("expected no cities, got %d", len(got))
	}
}

func TestLoadSampleCitiesMalformedFile(t *testing.T) {
	path := writeTempJSON(t, `{"cities": [ broken`)

	got := loadSampleCities(path)
	if got == nil || len(got) != 0 {
		t.Errorf("malformed json should yield an empty list, got %+v", got)
	}
}

func TestLoadSampleCitiesDropsInvalidEntries(t *testing.T) {
	path := writeTempJSON(t, `{"cities":[
		{"city":"Toronto","countryCode":"CA"},
		{"city":"","countryCode":"GB"},
		{"city":"Nowhere","countryCode":"TOOLONG"}
	]}`)

	got := loadSampleCities(path)
	if len(got) != 1 {
		t.Fatalf("expected only the valid entry, got %+v", got)
	}
}

func TestSampleCitiesIsSafeToCall(t *testing.T) {
	if got := SampleCities(); got == nil {
		t.Error("SampleCities should never return nil")
	}
}
