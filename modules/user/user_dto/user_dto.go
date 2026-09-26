// Package user_dto memuat DTO request/response modul user. Seluruhnya murni
// struct data (tanpa function/method) — pemetaan model→DTO berada di service.
package user_dto

// Response adalah representasi pengguna untuk API.
type Response struct {
	UserID               int64    `json:"userID"`
	FullName             string   `json:"fullName"`
	Email                string   `json:"email"`
	RoleID               int64    `json:"roleID"`
	Role                 string   `json:"role"`
	OrganizationID       *int64   `json:"organizationID,omitempty"`
	OrganizationTypeCode string   `json:"organizationTypeCode,omitempty"`
	WildcardTierAccess   []string `json:"wildcardTierAccess,omitempty"`
	EmailVerified        bool     `json:"emailVerified"`
	IsActive             bool     `json:"isActive"`
	PhotoURL             string   `json:"photoURL,omitempty"`
	HasGoogle            bool     `json:"hasGoogle"`
	HasPassword          bool     `json:"hasPassword"`
}

// CreateRequest adalah body membuat pengguna baru (oleh admin/tier organisasi).
// OrganizationID/WildcardTierAccess bersifat opsional dan divalidasi ulang di
// lapisan service sesuai kewenangan pemanggil — nilai dari client tidak
// pernah dipercaya langsung untuk menentukan cakupan akses.
type CreateRequest struct {
	FullName           string   `json:"fullName" validate:"required,min=3,max=150"`
	Email              string   `json:"email" validate:"required,email,max=150"`
	RoleID             int64    `json:"roleID" validate:"required"`
	Password           string   `json:"password" validate:"required,min=8,max=100"`
	IsActive           *bool    `json:"isActive"`
	OrganizationID     *int64   `json:"organizationID"`
	WildcardTierAccess []string `json:"wildcardTierAccess" validate:"omitempty,dive,oneof=LDK PUSKOMDA PUSKOMNAS"`
}

// UpdateRequest adalah body memperbarui pengguna. Password opsional — bila
// diisi, password pengguna diganti; bila kosong, password lama dipertahankan.
type UpdateRequest struct {
	FullName           string   `json:"fullName" validate:"required,min=3,max=150"`
	Email              string   `json:"email" validate:"required,email,max=150"`
	RoleID             int64    `json:"roleID" validate:"required"`
	IsActive           bool     `json:"isActive"`
	Password           string   `json:"password" validate:"omitempty,min=8,max=100"`
	OrganizationID     *int64   `json:"organizationID"`
	WildcardTierAccess []string `json:"wildcardTierAccess" validate:"omitempty,dive,oneof=LDK PUSKOMDA PUSKOMNAS"`
}

// StatusRequest adalah body mengubah status aktif pengguna.
type StatusRequest struct {
	IsActive bool `json:"isActive"`
}

// ListFilter menampung parameter penyaringan daftar pengguna.
type ListFilter struct {
	Search  string
	Email   string   // LIKE terhadap email — filter kolom "Email" CMS
	RoleIDs []int64  // exact match (IN) — filter kolom "Role" CMS, multi-select by ID
	Status  []string // "active" | "inactive" — multi-select (IN), kosong = semua status
	Limit   int
	Offset  int
	OrderBy string
}

// CMSFilter menampung parameter filter khusus endpoint CMS list (di luar
// dto.ListQuery yang sudah menampung search/page/limit/sort) — dipisah dari
// ListFilter (dipakai repository) supaya signature service.List tidak terus
// bertambah parameter positional setiap kali kolom filter baru ditambahkan.
// Status/RoleIDs multi-select (checkbox) — dikirim frontend sebagai query
// param comma-separated, di-parse handler lewat dto.ParseCSV/ParseInt64CSV.
type CMSFilter struct {
	Status  []string
	RoleIDs []int64
	Email   string
}

// BulkDeleteRequest adalah body untuk menonaktifkan (soft-delete) banyak
// pengguna sekaligus.
type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required,min=1"`
}

// MentionSearchResult adalah ringkasan pengguna minimal untuk autocomplete
// @mention pada komentar — sengaja tidak menyertakan email/role/dst. karena
// endpoint ini bisa dipanggil siapa pun yang login+verified, bukan hanya
// pemegang permission user.view (lihat modules/comment).
type MentionSearchResult struct {
	UserID   int64  `json:"userID"`
	FullName string `json:"fullName"`
	PhotoURL string `json:"photoURL,omitempty"`
}
