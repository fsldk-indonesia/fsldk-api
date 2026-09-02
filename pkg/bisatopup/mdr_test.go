package bisatopup

import "testing"

// Golden test — angka direplikasi persis dari komentar rasionalisasi
// ldksyahid-app PublicController.php:215-221: ceil(20000/0.99) = 20203,
// ceil(20203*0.01) = 203.
func TestCalculateGrossTotal_Golden(t *testing.T) {
	got := CalculateGrossTotal(20000, 1)
	if got != 20203 {
		t.Fatalf("CalculateGrossTotal(20000, 1) = %d, want 20203", got)
	}
}

func TestCalculateAdminFee_Golden(t *testing.T) {
	got := CalculateAdminFee(20203, 1)
	if got != 203 {
		t.Fatalf("CalculateAdminFee(20203, 1) = %d, want 203", got)
	}
}

// Verifikasi eksplisit alasan CEIL dipilih: donor menerima persis nominal
// yang ia niatkan setelah MDR dipotong gateway, tidak short Rp1 seperti
// yang terjadi bila dipakai ROUND.
func TestCalculateGrossTotal_NoShortfall(t *testing.T) {
	donation := int64(20000)
	mdrRate := 1.0

	gross := CalculateGrossTotal(donation, mdrRate)
	fee := CalculateAdminFee(gross, mdrRate)
	netReceived := gross - fee

	if netReceived != donation {
		t.Fatalf("net received = %d, want %d (donation amount must not be shorted)", netReceived, donation)
	}
}

func TestCalculateGrossTotal_ZeroMDR(t *testing.T) {
	if got := CalculateGrossTotal(10000, 0); got != 10000 {
		t.Fatalf("CalculateGrossTotal(10000, 0) = %d, want 10000", got)
	}
}
