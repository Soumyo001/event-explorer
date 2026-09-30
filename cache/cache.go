package cache

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const DefaultTTL = 5 * time.Minute

var (
	initSharedOnce sync.Once
	shared         *Cache
)

type entry struct {
	value     any
	expiresAt time.Time
}

func (e entry) expired(now time.Time) bool {
	return now.After(e.expiresAt)
}

type Stats struct {
	Hits    int64 `json:"hits"`
	Misses  int64 `json:"misses"`
	Entries int   `json:"entries"`
}

type Cache struct {
	mu      sync.RWMutex
	items   map[string]entry
	ttl     time.Duration
	hits    int64
	misses  int64
	nowFunc func() time.Time
}

func New(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Cache{
		items:   make(map[string]entry),
		ttl:     ttl,
		nowFunc: time.Now,
	}
}

func (c *Cache) now() time.Time {
	if c.nowFunc == nil {
		return time.Now()
	}
	return c.nowFunc()
}

func (c *Cache) SetClock(f func() time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nowFunc = f
}

func (c *Cache) TTL() time.Duration { return c.ttl }

func EventsKey(city, countryCode, category string) string {
	return fmt.Sprintf("events:%s:%s:%s",
		strings.ToLower(strings.TrimSpace(city)),
		strings.ToUpper(strings.TrimSpace(countryCode)),
		strings.ToLower(strings.TrimSpace(category)))
}

func EventKey(eventID string) string {
	return "event:" + strings.TrimSpace(eventID)
}

func (c *Cache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = entry{
		value:     value,
		expiresAt: c.now().Add(c.ttl),
	}
	logs.Debug("cache store: %s (ttl %s)", key, c.ttl)
}

func (c *Cache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, existed := c.items[key]
	delete(c.items, key)
	if existed {
		logs.Info("cache invalidated: %s", key)
	}
	return existed
}

func (c *Cache) Clear() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(c.items)
	c.items = make(map[string]entry)
	logs.Info("cache cleared: %d entries removed", n)
	return n
}

func (c *Cache) DeletePrefix(prefix string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for k := range c.items {
		if strings.HasPrefix(k, prefix) {
			delete(c.items, k)
			n++
		}
	}
	if n > 0 {
		logs.Info("cache invalidated %d entries with the prefix %q", n, prefix)
	}
	return n
}

func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := c.now()
	keys := make([]string, 0, len(c.items))
	for k, v := range c.items {
		if !v.expired(now) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

func (c *Cache) Len() int {
	return len(c.Keys())
}

func (c *Cache) Stats() Stats {
	c.mu.RLock()
	hits, misses := c.hits, c.misses
	c.mu.RUnlock()
	return Stats{Hits: hits, Misses: misses, Entries: c.Len()}
}

func (c *Cache) ResetStats() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hits, c.misses = 0, 0
}

func (c *Cache) countHit() {
	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
}

func (c *Cache) countMiss() {
	c.mu.Lock()
	c.misses++
	c.mu.Unlock()
}

func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	found, ok := c.items[key]
	now := c.now()
	c.mu.RUnlock()

	if !ok {
		c.countMiss()
		logs.Debug("cache miss: %s", key)
		return nil, false
	}

	if found.expired(now) {
		c.Delete(key)
		c.countMiss()
		logs.Info("cache expired: %s", key)
		return nil, false
	}

	c.countHit()
	logs.Info("cache hit: %s", key)
	return found.value, true
}

func Shared() *Cache {
	initSharedOnce.Do(func() {
		shared = New(DefaultTTL)
	})
	return shared
}

func InitShared(ttl time.Duration) *Cache {
	shared = New(ttl)
	initSharedOnce.Do(func() {})
	return shared
}
