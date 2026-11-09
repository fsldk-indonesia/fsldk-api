package submission_service

import "testing"

func TestKaderCodePrefix(t *testing.T) {
	got, err := kaderCodePrefix(2026, "P03", "L017")
	if err != nil || got != "02603017" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := kaderCodePrefix(2026, "P03", "LDK-X-001"); err == nil {
		t.Error("kode LDK lama harus ditolak")
	}
}

func TestBuildKaderCode(t *testing.T) {
	tests := []struct {
		name    string
		year    int
		pk, ldk string
		seq     int
		want    string
		wantErr bool
	}{
		{"contoh desain", 2026, "P03", "L017", 123, "0260301700123", false},
		{"Banten", 2026, "P19", "L301", 1, "0261930100001", false},
		{"kode pendek", 2026, "P3", "L7", 5, "0260300700005", false},
		{"seq maksimum", 2026, "P38", "L545", 99999, "0263854599999", false},
		{"tahun 2100", 2100, "P01", "L001", 1, "0000100100001", false},
		{"kode LDK lama", 2026, "P18", "LDK-CIREBONRAYA-001", 1, "", true},
		{"kode Puskomda lama", 2026, "PUSKOMDA-ACEH", "L001", 1, "", true},
		{"seq nol", 2026, "P01", "L001", 0, "", true},
		{"seq melebihi", 2026, "P01", "L001", 100000, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildKaderCode(tt.year, tt.pk, tt.ldk, tt.seq)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
