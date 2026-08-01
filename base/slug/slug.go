// Package slug menghasilkan slug URL yang ramah dari sebuah judul.
package slug

import (
	"regexp"
	"strings"
)

var (
	nonAlnum   = regexp.MustCompile(`[^a-z0-9]+`)
	trimHyphen = regexp.MustCompile(`^-+|-+$`)
)

// Make mengubah teks menjadi slug (lowercase, dipisah tanda hubung).
func Make(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = nonAlnum.ReplaceAllString(s, "-")
	s = trimHyphen.ReplaceAllString(s, "")
	if s == "" {
		s = "item"
	}
	return s
}
