package bisatopup

import "math"

// CalculateGrossTotal menghitung total tagihan (yang dibayar donor) dari
// nominal donasi bersih, memakai CEIL (bukan ROUND) — direplikasi persis
// dari ldksyahid-app PublicController.php:215-221. round() menyebabkan
// wallet shortfall Rp1 karena Bisabiller sendiri men-CEIL fee per
// transaksi; ceil() di sisi kita menghindari itu.
func CalculateGrossTotal(donationAmount int64, mdrRatePercent float64) int64 {
	mdrRate := mdrRatePercent / 100
	return int64(math.Ceil(float64(donationAmount) / (1 - mdrRate)))
}

// CalculateAdminFee menghitung MDR (admin fee) dari total tagihan gross,
// memakai CEIL — reuse identik formula ldksyahid-app.
func CalculateAdminFee(grossTotal int64, mdrRatePercent float64) int64 {
	mdrRate := mdrRatePercent / 100
	return int64(math.Ceil(float64(grossTotal) * mdrRate))
}
