// Package goldprice fetches the 1-gram Antam gold bar price from a public
// provider and serves it from a TTL in-memory cache. Pure data types live
// here; the client and cache logic live in goldprice.go.
package goldprice

import "time"

// Price is the outcome of one gold-price lookup. It is always valid: on
// upstream failure Success is false and Price carries the configured fallback.
type Price struct {
	Success  bool      `json:"success"`  // false = upstream failed, Price is the fallback value
	Price    int       `json:"price"`    // Rp per gram
	Source   string    `json:"source"`   // "antam-live" | "fallback"
	CachedAt time.Time `json:"cachedAt"` // when this value was produced
}
