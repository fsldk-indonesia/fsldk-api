// Package user_repository adalah lapisan akses data modul user (GORM).
package user_repository

import (
	"context"
	"database/sql"
	"errors"

	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
)

// ErrNotFound dikembalikan bila pengguna tidak ditemukan.
var ErrNotFound = errors.New("user tidak ditemukan")

// Repository adalah kontrak akses data pengguna.
type Repository interface {
	FindByID(ctx context.Context, id int64) (user_model.User, error)
	FindByEmail(ctx context.Context, email string) (user_model.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByEmailExcept(ctx context.Context, email string, exceptID int64) (bool, error)
	Create(ctx context.Context, p user_model.CreateParams) (int64, error)
	List(ctx context.Context, f user_dto.ListFilter) ([]user_model.User, int64, error)
	// SearchActive returns active users whose fullName matches search (or the
	// most recently created ones when search is empty), for the @mention
	// autocomplete in comments — deliberately unfiltered by role/permission
	// since any verified user may mention any other active user (including
	// themselves).
	SearchActive(ctx context.Context, search string, limit int) ([]user_model.User, error)
	Update(ctx context.Context, id int64, fullName, email string, roleID int64, isActive bool, organizationID sql.NullInt64, wildcardTierAccess sql.NullString, updatedBy int64) error
	// UpdateContactInfo memperbarui kolom kontak swadaya (No Whatsapp/Alamat)
	// yang dapat diubah pemilik akun sendiri lewat Profil Saya — terpisah dari
	// Update() (dipakai admin Kelola Pengguna) supaya self-service tidak perlu
	// ikut mengirim field identitas/role yang tidak relevan.
	UpdateContactInfo(ctx context.Context, id int64, phoneNumber, address sql.NullString) error
	SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error
	SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error
	LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error
	UpdatePhoto(ctx context.Context, id int64, photoURL string) error
	MarkEmailVerified(ctx context.Context, id int64) error
	SoftDelete(ctx context.Context, id int64, updatedBy int64) error
	LogLogin(ctx context.Context, userID int64, ip, ua, status string) error
}
