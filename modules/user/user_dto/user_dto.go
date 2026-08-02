// Package user_dto memuat DTO request/response modul user. Seluruhnya murni
// struct data (tanpa function/method) — pemetaan model→DTO berada di service.
package user_dto

// Response adalah representasi pengguna untuk API.
type Response struct {
	UserID        int64  `json:"userID"`
	FullName      string `json:"fullName"`
	Email         string `json:"email"`
	RoleID        int64  `json:"roleID"`
	Role          string `json:"role"`
	EmailVerified bool   `json:"emailVerified"`
	IsActive      bool   `json:"isActive"`
	PhotoURL      string `json:"photoURL,omitempty"`
	HasGoogle     bool   `json:"hasGoogle"`
	HasPassword   bool   `json:"hasPassword"`
}

// CreateRequest adalah body membuat pengguna baru (oleh admin).
type CreateRequest struct {
	FullName string `json:"fullName" validate:"required,min=3,max=150"`
	Email    string `json:"email" validate:"required,email,max=150"`
	RoleID   int64  `json:"roleID" validate:"required"`
	Password string `json:"password" validate:"required,min=8,max=100"`
	IsActive *bool  `json:"isActive"`
}

// UpdateRequest adalah body memperbarui pengguna. Password opsional — bila
// diisi, password pengguna diganti; bila kosong, password lama dipertahankan.
type UpdateRequest struct {
	FullName string `json:"fullName" validate:"required,min=3,max=150"`
	Email    string `json:"email" validate:"required,email,max=150"`
	RoleID   int64  `json:"roleID" validate:"required"`
	IsActive bool   `json:"isActive"`
	Password string `json:"password" validate:"omitempty,min=8,max=100"`
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
