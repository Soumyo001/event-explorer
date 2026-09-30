package models

type CacheKeysResponse struct {
	Count  int      `json:"count"`
	Keys   []string `json:"keys"`
	Hits   int64    `json:"hits"`
	Misses int64    `json:"misses"`
	TTL    string   `json:"ttl"`
}

type CacheInvalidateResponse struct {
	Scope   string `json:"scope"`
	Target  string `json:"target,omitempty"`
	Removed int    `json:"removed"`
	Message string `json:"message"`
}
