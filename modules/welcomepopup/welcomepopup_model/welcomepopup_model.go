// Package welcomepopup_model memuat entitas modul welcomepopup (popup
// selamat datang di halaman Beranda publik — satu baris singleton, HTML/JS/
// CSS mentah dikelola Super Admin lewat CMS). Seluruhnya murni struct data
// (tanpa function/method), kecuali konstanta SingletonID.
package welcomepopup_model

import "time"

// SingletonID adalah satu-satunya id baris yang pernah ada di ms_welcome_popup —
// tabel ini sengaja tidak pernah punya baris kedua.
const SingletonID int64 = 1

// WelcomePopup merepresentasikan satu baris ms_welcome_popup.
type WelcomePopup struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	IsEnabled   bool       `gorm:"column:isEnabled"`
	HTMLContent *string    `gorm:"column:htmlContent"`
	JSContent   *string    `gorm:"column:jsContent"`
	CSSContent  *string    `gorm:"column:cssContent"`
	UpdatedDate *time.Time `gorm:"column:updatedDate"`
	UpdatedBy   *int64     `gorm:"column:updatedBy"`
}
