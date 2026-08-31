package goldprice

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Client fetches and caches the 1-gram Antam gold bar price. The mutex-guarded
// cache is this package's implementation identity (same idea as pkg/mailer
// holding an SMTP connection).
type Client struct {
	apiURL   string
	fallback int
	ttl      time.Duration

	mu      sync.Mutex
	cached  Price
	fetched time.Time
}

// NewClient builds a Client. cacheMinutes is the TTL of a successful upstream
// result; fallback (Rp/gram) is served when upstream fails.
func NewClient(apiURL string, fallback, cacheMinutes int) *Client {
	return &Client{
		apiURL:   apiURL,
		fallback: fallback,
		ttl:      time.Duration(cacheMinutes) * time.Minute,
	}
}

// Get returns the gold price — from cache while fresh, otherwise from upstream.
// It never returns an invalid Price: on upstream failure it returns
// Success=false with Price=fallback. A fallback result is NOT stored as fresh
// (c.fetched is left zero) so the next call retries upstream immediately once
// the provider recovers, instead of waiting out the TTL.
func (c *Client) Get(ctx context.Context, forceRefresh bool) Price {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !forceRefresh && time.Since(c.fetched) < c.ttl && c.cached.Price > 0 {
		return c.cached
	}

	p := c.fetchUpstream(ctx)
	if !p.Success {
		p = Price{Success: false, Price: c.fallback, Source: "fallback", CachedAt: time.Now()}
		c.cached = p
		c.fetched = time.Time{} // keep a value to serve fast, but not marked fresh
		return p
	}

	c.cached = p
	c.fetched = time.Now()
	return p
}

// upstreamResponse is the shape of the logam-mulia-api payload. Only the
// fields we match on are decoded; weight is float64 because the provider may
// send it either way.
type upstreamResponse struct {
	Data []struct {
		Material     string  `json:"material"`
		MaterialType string  `json:"materialType"`
		Weight       float64 `json:"weight"`
		WeightUnit   string  `json:"weightUnit"`
		SellPrice    int     `json:"sellPrice"`
	} `json:"data"`
}

// fetchUpstream GETs c.apiURL (10s timeout) and pulls sellPrice for the
// standard 1-gram gold bar. Any error (non-200, timeout, bad JSON, item not
// found, non-positive price) yields Price{Success: false}. This is the only
// place that knows the upstream payload shape — swap the provider by editing
// c.apiURL and this parser together.
func (c *Client) fetchUpstream(ctx context.Context) Price {
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return Price{Success: false}
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Price{Success: false}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Price{Success: false}
	}

	var body upstreamResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Price{Success: false}
	}

	for _, item := range body.Data {
		if item.Material == "gold" && item.MaterialType == "Emas Batangan" &&
			item.Weight == 1 && item.WeightUnit == "gr" && item.SellPrice > 0 {
			return Price{Success: true, Price: item.SellPrice, Source: "antam-live", CachedAt: time.Now()}
		}
	}
	return Price{Success: false}
}
