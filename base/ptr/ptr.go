// Package ptr menyediakan pembantu konversi nilai ke pointer.
package ptr

// Str mengembalikan pointer string, atau nil bila string kosong.
func Str(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// Int mengembalikan pointer int.
func Int(i int) *int { return &i }
