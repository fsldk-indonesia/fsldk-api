package goldprice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const okPayload = `{"data":[
	{"material":"gold","materialType":"Perhiasan","weight":1,"weightUnit":"gr","sellPrice":111},
	{"material":"gold","materialType":"Emas Batangan","weight":1,"weightUnit":"gr","sellPrice":2750000},
	{"material":"gold","materialType":"Emas Batangan","weight":5,"weightUnit":"gr","sellPrice":13000000}
]}`

func TestGet_LiveThenCached(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(okPayload))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2600000, 60)

	first := c.Get(context.Background(), false)
	if !first.Success || first.Price != 2750000 || first.Source != "antam-live" {
		t.Fatalf("unexpected first result: %+v", first)
	}

	second := c.Get(context.Background(), false)
	if second.CachedAt != first.CachedAt || hits != 1 {
		t.Fatalf("expected cache hit (no upstream call), got hits=%d %+v", hits, second)
	}

	if c.Get(context.Background(), true); hits != 2 {
		t.Fatalf("expected forced refresh to re-fetch upstream, got hits=%d", hits)
	}
}

func TestGet_FallbackOnUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 2600000, 60)

	p := c.Get(context.Background(), false)
	if p.Success || p.Price != 2600000 || p.Source != "fallback" {
		t.Fatalf("expected fallback result, got %+v", p)
	}
	// A fallback is not cached as fresh: the next call must retry upstream.
	if !c.fetched.IsZero() {
		t.Fatalf("fallback must leave fetched zero, got %v", c.fetched)
	}
}

func TestGet_ItemNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"material":"silver","materialType":"Emas Batangan","weight":1,"weightUnit":"gr","sellPrice":999}]}`))
	}))
	defer srv.Close()

	if p := NewClient(srv.URL, 2600000, 60).Get(context.Background(), false); p.Success {
		t.Fatalf("expected failure when no matching gold bar item, got %+v", p)
	}
}
