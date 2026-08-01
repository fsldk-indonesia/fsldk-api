// Package user_model memuat entitas modul user. Seluruhnya murni struct data
// (tanpa function/method) — logika pemetaan berada di lapisan service.
package user_model

import (
	"database/sql"
	"time"
)

// User merepresentasikan satu baris ms_user. Field RoleName bersifat read-only
// (hasil join ke ms_role), ditandai `->` agar tidak ikut tertulis saat Create/Update.
type User struct {
	UserID             int64          `gorm:"column:userID;primaryKey"`
	RoleID             int64          `gorm:"column:roleID"`
	RoleName           string         `gorm:"column:roleName;->"`
	FullName           string         `gorm:"column:fullName"`
	Email              string         `gorm:"column:email"`
	Username           sql.NullString `gorm:"column:username"`
	Password           sql.NullString `gorm:"column:password"`
	GoogleID           sql.NullString `gorm:"column:googleID"`
	EmailVerifiedDate  sql.NullTime   `gorm:"column:emailVerifiedDate"`
	PhoneNumber        sql.NullString `gorm:"column:phoneNumber"`
	PhotoURL           sql.NullString `gorm:"column:photoURL"`
	MustChangePassword bool           `gorm:"column:mustChangePassword"`
	IsActive           bool           `gorm:"column:isActive"`
	CreatedDate        time.Time      `gorm:"column:createdDate"`
	CreatedBy          sql.NullInt64  `gorm:"column:createdBy"`
	UpdatedDate        sql.NullTime   `gorm:"column:updatedDate"`
	UpdatedBy          sql.NullInt64  `gorm:"column:updatedBy"`
}

// CreateParams menampung data untuk membuat pengguna baru.
type CreateParams struct {
	RoleID        int64
	FullName      string
	Email         string
	Password      sql.NullString
	GoogleID      sql.NullString
	EmailVerified bool
	CreatedBy     sql.NullInt64
}
