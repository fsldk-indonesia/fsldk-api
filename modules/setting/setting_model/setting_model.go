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
	// SettingValue2 — kolom nilai kedua generik, dipakai pertama kali oleh
	// group "deadline" untuk estimasi biaya (digit polos, mis. "1500000").
	// NULL untuk setting lain yang tidak membutuhkannya.
	SettingValue2 *string    `gorm:"column:settingValue2"`
	IsHide        bool       `gorm:"column:isHide"`
	UpdatedDate  *time.Time `gorm:"column:updatedDate"`
}

// Group & key konstan — 1 sumber kebenaran dipakai baik oleh seed migration
// maupun kode yang membaca setting-nya (mis. shortlinkrequest_service, §1a.4).
const (
	GroupLayanan = "layanan"

	KeyShortlinkPICName     = "shortlink_pic_name"
	KeyShortlinkPICWhatsapp = "shortlink_pic_whatsapp"

	KeyQRCodePICName     = "qrcode_pic_name"
	KeyQRCodePICWhatsapp = "qrcode_pic_whatsapp"

	// GroupKantongAmal/KeyWithdrawalOtpEmail — email penerima OTP verifikasi
	// withdrawal Kantong Amal (item 8 revision-prompt-2.md), sengaja isHide=true.
	GroupKantongAmal      = "kantong_amal"
	KeyWithdrawalOtpEmail = "withdrawal_otp_email"

	// GroupNotifikasi/KeyWhatsAppEnabled — saklar global kirim/tidaknya
	// SEMUA notifikasi WhatsApp (donasi, withdrawal OTP, balasan shortlink
	// request, dst — lihat jobqueue_service.executeWhatsAppTemplate, satu
	// titik gate untuk seluruh pengiriman). isHide=false (sengaja tampil di
	// App Settings CMS, beda dari KeyWithdrawalOtpEmail).
	GroupNotifikasi    = "notifikasi"
	KeyWhatsAppEnabled = "whatsapp_enabled"

	// GroupKontak/KeyContactEmail — email kontak publik yang ditampilkan di
	// section "Hubungi Kami" beranda dan halaman /tentang/kontak (dulu
	// hardcoded di frontend). isHide=false, dan satu-satunya key yang punya
	// endpoint publik (lihat setting_handler.PublicContactEmail) karena
	// dikonsumsi langsung oleh SPA publik, bukan cuma dibaca server-side.
	GroupKontak     = "kontak"
	KeyContactEmail = "contact_email"

	// KeyContactWhatsapp — nomor WhatsApp kontak publik (floating button +
	// section "Hubungi Kami" beranda), disimpan format tampilan biasa
	// ("+62 851-1133-2861") supaya gampang diedit admin — frontend yang
	// membuang karakter non-digit untuk membentuk link wa.me. isHide=false,
	// publik lewat endpoint yang sama dengan KeyContactEmail (lihat
	// setting_handler.PublicContactWhatsapp).
	KeyContactWhatsapp = "contact_whatsapp"

	// GroupDeadline/KeyDeadline* — tanggal jatuh tempo operasional (format
	// "YYYY-MM-DD"), dibaca dashboard_service untuk banner peringatan
	// closeable di dashboard CMS Utama saat sisa waktu menipis atau sudah
	// lewat (lihat dashboard_dto.DeadlineAlert). isHide=false, settingValue
	// kosong by default — diisi admin lewat App Settings.
	GroupDeadline                 = "deadline"
	KeyDeadlineWhatsAppIntegration = "whatsapp_integration"
	KeyDeadlineHostingServer       = "hosting_server"
	KeyDeadlineRentDomain          = "rent_domain"
	KeyDeadlineCertificateWebsite  = "certificate_website"
)
