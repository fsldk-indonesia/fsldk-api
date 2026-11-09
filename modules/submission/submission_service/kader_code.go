package submission_service

import (
	"fmt"
	"regexp"
	"strconv"
)

// Format ID Kader (13 digit): YYY PP LLL NNNNN
//   YYY   = tahun registrasi, 2 digit terakhir dengan nol di depan (2026 -> "026")
//   PP    = nomor kode Puskomda (P03 -> "03")
//   LLL   = nomor kode LDK (L017 -> "017")
//   NNNNN = urutan kader secara nasional dalam tahun tersebut
const (
	kaderCodeLength   = 13
	kaderSeqMaxValue  = 99999
	kaderCodeYearSize = 3
)

var (
	puskomdaCodePattern = regexp.MustCompile(`^P(\d{1,2})$`)
	ldkCodePattern      = regexp.MustCompile(`^L(\d{1,3})$`)
)

// kaderYearPrefix mengembalikan segmen tahun 3 digit untuk kode kader.
func kaderYearPrefix(year int) string {
	return fmt.Sprintf("%03d", year%100)
}

// buildKaderCode merakit ID Kader dari kode organisasi (format "P03"/"L017") —
// kode organisasi yang tidak sesuai format ditolak agar tidak menerbitkan ID cacat.
func buildKaderCode(year int, puskomdaCode, ldkCode string, seq int) (string, error) {
	pm := puskomdaCodePattern.FindStringSubmatch(puskomdaCode)
	if pm == nil {
		return "", fmt.Errorf("kode Puskomda %q tidak valid (harus P01–P99)", puskomdaCode)
	}
	lm := ldkCodePattern.FindStringSubmatch(ldkCode)
	if lm == nil {
		return "", fmt.Errorf("kode LDK %q tidak valid (harus L001–L999)", ldkCode)
	}
	if seq < 1 || seq > kaderSeqMaxValue {
		return "", fmt.Errorf("urutan kader %d di luar batas 1–%d", seq, kaderSeqMaxValue)
	}
	p, _ := strconv.Atoi(pm[1])
	l, _ := strconv.Atoi(lm[1])
	code := fmt.Sprintf("%s%02d%03d%05d", kaderYearPrefix(year), p, l, seq)
	if len(code) != kaderCodeLength {
		return "", fmt.Errorf("panjang kode kader %d tidak sesuai", len(code))
	}
	return code, nil
}
