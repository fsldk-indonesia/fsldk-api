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
	RoleID  int64
	Limit   int
	Offset  int
	OrderBy string
}
