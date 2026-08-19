// Package auth_dto memuat DTO request/response modul auth.
package auth_dto

// ---------- Request ----------

// RegisterRequest adalah body registrasi mandiri.
type RegisterRequest struct {
	FullName             string `json:"fullName" validate:"required,min=3,max=150"`
	Email                string `json:"email" validate:"required,email,max=150"`
	Password             string `json:"password" validate:"required,min=8,max=100"`
	PasswordConfirmation string `json:"passwordConfirmation" validate:"required,eqfield=Password"`
}

// LoginRequest adalah body login email + password.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// GoogleRequest adalah body login/registrasi via Google.
type GoogleRequest struct {
	IDToken string `json:"idToken" validate:"required"`
}

// RefreshRequest adalah body refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// ChangePasswordRequest adalah body ubah kata sandi.
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8,max=100"`
}

// ForgotPasswordRequest adalah body permintaan reset password.
type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ResetPasswordRequest adalah body penetapan password baru.
type ResetPasswordRequest struct {
	Token                string `json:"token" validate:"required"`
	Password             string `json:"password" validate:"required,min=8,max=100"`
	PasswordConfirmation string `json:"passwordConfirmation" validate:"required,eqfield=Password"`
}

// UpdateContactRequest adalah body memperbarui kontak swadaya (Profil Saya) —
// No Whatsapp (kolom phoneNumber) & Alamat, terpisah dari data identitas/role
// yang hanya boleh diubah admin lewat Kelola Pengguna.
type UpdateContactRequest struct {
	PhoneNumber string `json:"phoneNumber" validate:"omitempty,max=30"`
	Address     string `json:"address" validate:"omitempty,max=255"`
}

// ---------- Response ----------

// UserProfile adalah profil pengguna yang dikembalikan bersama sesi.
type UserProfile struct {
	UserID               int64    `json:"userID"`
	FullName             string   `json:"fullName"`
	Email                string   `json:"email"`
	EmailVerified        bool     `json:"emailVerified"`
	Role                 string   `json:"role"`
	Permissions          []string `json:"permissions"`
	PhotoURL             string   `json:"photoURL,omitempty"`
	PhoneNumber          string   `json:"phoneNumber,omitempty"`
	Address              string   `json:"address,omitempty"`
	OrganizationID       *int64   `json:"organizationID,omitempty"`
	OrganizationTypeCode string   `json:"organizationTypeCode,omitempty"`
	WildcardTierAccess   []string `json:"wildcardTierAccess,omitempty"`
}

// AuthResponse adalah hasil login/google/refresh.
type AuthResponse struct {
	AccessToken  string      `json:"accessToken"`
	RefreshToken string      `json:"refreshToken"`
	ExpiresIn    int         `json:"expiresIn"`
	User         UserProfile `json:"user"`
}

// RegisterResponse adalah hasil registrasi (belum berisi token).
type RegisterResponse struct {
	UserID        int64  `json:"userID"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	Message       string `json:"message"`
}
