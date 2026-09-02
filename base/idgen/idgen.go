// Package idgen menyediakan pembantu pembuatan identifier non-enumerable.
package idgen

import (
	"crypto/rand"
	"fmt"
)

// NewUUIDv4 menghasilkan UUID v4 (RFC 4122) memakai crypto/rand — dipakai
// untuk kolom publicRef entity finansial (referensi non-enumerable di URL
// publik, terpisah dari primary key autoincrement).
func NewUUIDv4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
