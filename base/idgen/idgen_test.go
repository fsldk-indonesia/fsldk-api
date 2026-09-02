package idgen

import (
	"regexp"
	"testing"
)

var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewUUIDv4_Format(t *testing.T) {
	got := NewUUIDv4()
	if !uuidV4Pattern.MatchString(got) {
		t.Fatalf("NewUUIDv4() = %q, does not match UUID v4 format", got)
	}
}

func TestNewUUIDv4_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := NewUUIDv4()
		if seen[id] {
			t.Fatalf("duplicate UUID generated: %s", id)
		}
		seen[id] = true
	}
}
