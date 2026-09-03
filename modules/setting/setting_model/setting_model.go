// Package setting_model memuat entitas modul setting (App Settings — konfigurasi
// runtime platform generik). Seluruhnya murni struct data (tanpa function/method).
package setting_model

import "time"

// Setting merepresentasikan satu baris ms_setting. IsHide=true berarti
// baris ini tidak boleh muncul di halaman App Settings CMS (mis. konfigurasi
// sensitif seperti email penerima OTP withdrawal, item 8 revision-prompt-2.md)
// — tetap bisa dibaca langsung via FindByGroupKey oleh kode yang membutuhkannya.
type Setting struct {
	SettingID    int64      `gorm:"column:settingID;primaryKey"`
	SettingGroup string     `gorm:"column:settingGroup"`
	SettingKey   string     `gorm:"column:settingKey"`
	SettingLabel string     `gorm:"column:settingLabel"`
	SettingValue *string    `gorm:"column:settingValue"`
	IsHide       bool       `gorm:"column:isHide"`
	UpdatedDate  *time.Time `gorm:"column:updatedDate"`
}

// Group & key konstan — 1 sumber kebenaran dipakai baik oleh seed migration
// maupun kode yang membaca setting-nya (mis. shortlinkrequest_service, §1a.4).
const (
	GroupLayanan = "layanan"

	KeyShortlinkPICName     = "shortlink_pic_name"
	KeyShortlinkPICWhatsapp = "shortlink_pic_whatsapp"

	// GroupKantongAmal/KeyWithdrawalOtpEmail — email penerima OTP verifikasi
	// withdrawal Kantong Amal (item 8 revision-prompt-2.md), sengaja isHide=true.
	GroupKantongAmal      = "kantong_amal"
	KeyWithdrawalOtpEmail = "withdrawal_otp_email"
)
