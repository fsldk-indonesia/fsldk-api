package dto

import (
	"strconv"
	"strings"
)

// ParseCSV memecah query param comma-separated (mis. "published,draft")
// menjadi slice string, dipakai filter CMS multi-select (Status, dst.) —
// dipisah dari base/dto/pagination.go supaya tidak diulang identik di
// modul news/article/user. String kosong menghasilkan slice kosong (BUKAN
// slice berisi satu string kosong), begitu juga tiap elemen di-trim &
// elemen kosong hasil trim dibuang (mis. "a,,b" -> ["a","b"]).
func ParseCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ParseInt64CSV sama seperti ParseCSV tapi elemennya di-parse ke int64 —
// dipakai filter ID multi-select (mis. categoryID, roleID). Elemen yang
// gagal di-parse dilewati (best-effort, bukan error) — konsisten dengan
// konvensi query-param parsing lain di codebase ini (strconv.ParseInt yang
// gagal diabaikan via `_`).
func ParseInt64CSV(s string) []int64 {
	parts := ParseCSV(s)
	out := make([]int64, 0, len(parts))
	for _, p := range parts {
		if v, err := strconv.ParseInt(p, 10, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}
