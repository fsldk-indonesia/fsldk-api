// Package welcomepopup_dto memuat DTO request/response modul welcomepopup.
// Seluruhnya murni struct data (tanpa function/method).
package welcomepopup_dto

// Response adalah representasi lengkap untuk CMS (Super Admin) — termasuk
// metadata audit.
type Response struct {
	IsEnabled   bool   `json:"isEnabled"`
	HTMLContent string `json:"htmlContent"`
	JSContent   string `json:"jsContent"`
	CSSContent  string `json:"cssContent"`
	UpdatedDate string `json:"updatedDate"`
}

// PublicResponse adalah representasi untuk endpoint publik (tanpa auth) —
// SENGAJA cuma 4 field ini, tidak pernah membawa UpdatedDate/UpdatedBy.
type PublicResponse struct {
	IsEnabled   bool   `json:"isEnabled"`
	HTMLContent string `json:"htmlContent"`
	JSContent   string `json:"jsContent"`
	CSSContent  string `json:"cssContent"`
}

// UpdateRequest adalah body memperbarui Welcome Popup. Tidak ada
// validate:"required" pada field kontennya dengan sengaja — mengosongkan
// field adalah state yang sah (mis. mematikan tanpa menghapus histori).
type UpdateRequest struct {
	IsEnabled   bool   `json:"isEnabled"`
	HTMLContent string `json:"htmlContent"`
	JSContent   string `json:"jsContent"`
	CSSContent  string `json:"cssContent"`
}
