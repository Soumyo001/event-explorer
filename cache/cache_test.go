package cache

import (
	"sync"
	"testing"
	"time"
)

func TestSetAndGetReturnsStoredValue(t *testing.T) {
	c := New(time.Minute)
	c.Set("events:toronto:CA:music", []string{"a", "b"})

	got, ok := c.Get("events:toronto:CA:music")
	if !ok {
		t.Fatal("expected a cache hit for a freshly stored key")
	}
	list, isSlice := got.([]string)
	if !isSlice || len(list) != 2 {
		t.Fatalf("stored value came back wrong: %#v", got)
	}
}

func TestGetMissOnUnknownKey(t *testing.T) {
	c := New(time.Minute)

	if _, ok := c.Get("events:nowhere:XX:music"); ok {
		t.Fatal("expected a miss for a key that was never stored")
	}
	if s := c.Stats(); s.Misses != 1 || s.Hits != 0 {
		t.Fatalf("expected 1 miss and 0 hits, got %+v", s)
	}
}

func TestGetExpiredEntryIsAMissAndIsRemoved(t *testing.T) {
	c := New(5 * time.Minute)

	base := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	c.SetClock(func() time.Time { return base })
	c.Set("events:toronto:CA:music", "first")

	// Still inside the window.
	c.SetClock(func() time.Time { return base.Add(4 * time.Minute) })
	if _, ok := c.Get("events:toronto:CA:music"); !ok {
		t.Fatal("entry should still be valid four minutes in")
	}

	// Past the window.
	c.SetClock(func() time.Time { return base.Add(5*time.Minute + time.Second) })
	if _, ok := c.Get("events:toronto:CA:music"); ok {
		t.Fatal("entry should have expired after five minutes")
	}

	if c.Len() != 0 {
		t.Fatalf("expected the expired entry to be removed, %d remain", c.Len())
	}
}

func TestExpiredEntryIsReplacedOnRefresh(t *testing.T) {
	c := New(5 * time.Minute)
	base := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	c.SetClock(func() time.Time { return base })
	c.Set("events:toronto:CA:music", "stale")

	c.SetClock(func() time.Time { return base.Add(6 * time.Minute) })
	c.Get("events:toronto:CA:music")
	c.Set("events:toronto:CA:music", "fresh")

	got, ok := c.Get("events:toronto:CA:music")
	if !ok || got != "fresh" {
		t.Fatalf("expected the refreshed value, got %v (ok=%v)", got, ok)
	}
}

func TestHitsAndMissesAreCounted(t *testing.T) {
	c := New(time.Minute)
	c.Set("k", 1)

	c.Get("k")
	c.Get("k")
	c.Get("missing")

	s := c.Stats()
	if s.Hits != 2 || s.Misses != 1 {
		t.Fatalf("expected 2 hits and 1 miss, got %+v", s)
	}
	if s.Entries != 1 {
		t.Fatalf("expected 1 live entry, got %d", s.Entries)
	}
}

func TestDeleteRemovesASingleEntry(t *testing.T) {
	c := New(time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	if !c.Delete("a") {
		t.Fatal("Delete should report true for an existing key")
	}
	if c.Delete("a") {
		t.Fatal("Delete should report false the second time")
	}
	if _, ok := c.Get("a"); ok {
		t.Fatal("deleted key should miss")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("unrelated key should survive a single delete")
	}
}

func TestClearEmptiesTheCache(t *testing.T) {
	c := New(time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	if n := c.Clear(); n != 2 {
		t.Fatalf("expected Clear to report 2 removed, got %d", n)
	}
	if c.Len() != 0 {
		t.Fatalf("expected an empty cache, %d entries remain", c.Len())
	}
}

func TestDeletePrefixRemovesMatchingKeysOnly(t *testing.T) {
	c := New(time.Minute)
	c.Set(EventsKey("Toronto", "CA", "Music"), 1)
	c.Set(EventsKey("Toronto", "CA", "Sports"), 2)
	c.Set(EventsKey("London", "GB", "Music"), 3)

	if n := c.DeletePrefix("events:toronto:CA:"); n != 2 {
		t.Fatalf("expected 2 Toronto entries removed, got %d", n)
	}
	if _, ok := c.Get(EventsKey("London", "GB", "Music")); !ok {
		t.Fatal("London entry should be untouched")
	}
}

func TestEventsKeyNormalisesCityAndCountry(t *testing.T) {
	a := EventsKey("Toronto", "ca", "Music")
	b := EventsKey("  toronto ", "CA", "music")
	if a != b {
		t.Fatalf("keys should normalise to the same value:\n%q\n%q", a, b)
	}
	if a == EventsKey("Toronto", "CA", "Sports") {
		t.Fatal("different categories must produce different keys")
	}
	if a == EventsKey("Toronto", "US", "Music") {
		t.Fatal("different countries must produce different keys")
	}
}

func TestEventKeyIsPrefixed(t *testing.T) {
	if got := EventKey(" abc123 "); got != "event:abc123" {
		t.Fatalf("unexpected event key: %q", got)
	}
}

func TestKeysExcludesExpiredEntries(t *testing.T) {
	c := New(time.Minute)
	base := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)

	c.SetClock(func() time.Time { return base })
	c.Set("fresh", 1)

	c.SetClock(func() time.Time { return base.Add(2 * time.Minute) })
	c.Set("newer", 2)

	keys := c.Keys()
	if len(keys) != 1 || keys[0] != "newer" {
		t.Fatalf("expected only the unexpired key, got %v", keys)
	}
}

func TestResetStatsClearsCountersButKeepsValues(t *testing.T) {
	c := New(time.Minute)
	c.Set("k", 1)
	c.Get("k")
	c.Get("nope")

	c.ResetStats()

	s := c.Stats()
	if s.Hits != 0 || s.Misses != 0 {
		t.Fatalf("counters should be zero, got %+v", s)
	}
	if _, ok := c.Get("k"); !ok {
		t.Fatal("ResetStats must not drop stored values")
	}
}

func TestNewFallsBackToDefaultTTL(t *testing.T) {
	if got := New(0).TTL(); got != DefaultTTL {
		t.Fatalf("expected the default TTL, got %s", got)
	}
	if got := New(-time.Second).TTL(); got != DefaultTTL {
		t.Fatalf("expected the default TTL for a negative input, got %s", got)
	}
}

func TestSharedReturnsTheSameInstance(t *testing.T) {
	first := Shared()
	second := Shared()
	if first != second {
		t.Fatal("Shared must return one process-wide cache")
	}
}

func TestInitSharedAppliesTheGivenTTL(t *testing.T) {
	c := InitShared(90 * time.Second)
	if c.TTL() != 90*time.Second {
		t.Fatalf("expected the configured TTL, got %s", c.TTL())
	}
	if Shared() != c {
		t.Fatal("Shared should return the cache InitShared built")
	}
}

func TestConcurrentAccessIsSafe(t *testing.T) {
	c := New(time.Minute)
	const workers = 50

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := EventsKey("Toronto", "CA", "Music")
			c.Set(key, n)
			c.Get(key)
			c.Keys()
			c.Stats()
		}(i)
	}
	wg.Wait()

	if _, ok := c.Get(EventsKey("Toronto", "CA", "Music")); !ok {
		t.Fatal("the key should survive concurrent writes")
	}
}
