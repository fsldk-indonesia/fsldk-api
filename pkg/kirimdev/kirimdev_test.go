package kirimdev

import "testing"

// TestNormalizePhone mengunci perbaikan bug notifikasi WhatsApp gagal
// terkirim untuk nomor format lokal Indonesia ("08xxx") — dikonfirmasi
// lewat reproduksi manual ke Kirimdev: payload identik, "628xxx" sukses 200,
// "08xxx" ditolak 400 invalid_field_value "Invalid input".
func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"leading zero (format lokal paling umum)", "0895394755672", "62895394755672"},
		{"sudah 62, tidak diubah", "62895394755672", "62895394755672"},
		{"awalan +62 dilucuti", "+62895394755672", "62895394755672"},
		{"cuma awalan 8, ditambah 62", "895394755672", "62895394755672"},
		{"spasi/tanda hubung dibersihkan", "0895-394-755672", "62895394755672"},
		{"kosong tetap kosong", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizePhone(tc.input); got != tc.want {
				t.Fatalf("normalizePhone(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
