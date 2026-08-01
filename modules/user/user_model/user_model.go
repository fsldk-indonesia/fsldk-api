// Package user_model memuat entitas modul user.
package user_model

import (
	"database/sql"
	"time"
)

// User merepresentasikan satu baris ms_user (dengan kolom join roleName).
type User struct {
	UserID             int64          `db:"userID"`
	RoleID             int64          `db:"roleID"`
	RoleName           string         `db:"roleName"`
	FullName           string         `db:"fullName"`
	Email              string         `db:"email"`
	Username           sql.NullString `db:"username"`
	Password           sql.NullString `db:"password"`
	GoogleID           sql.NullString `db:"googleID"`
	EmailVerifiedDate  sql.NullTime   `db:"emailVerifiedDate"`
	PhoneNumber        sql.NullString `db:"phoneNumber"`
	PhotoURL           sql.NullString `db:"photoURL"`
	MustChangePassword bool           `db:"mustChangePassword"`
	IsActive           bool           `db:"isActive"`
	CreatedDate        time.Time      `db:"createdDate"`
	CreatedBy          sql.NullInt64  `db:"createdBy"`
	UpdatedDate        sql.NullTime   `db:"updatedDate"`
	UpdatedBy          sql.NullInt64  `db:"updatedBy"`
}

// EmailVerified mengembalikan true bila email pengguna telah terverifikasi.
func (u User) EmailVerified() bool { return u.EmailVerifiedDate.Valid }

// HasPassword mengembalikan true bila pengguna memiliki password lokal.
func (u User) HasPassword() bool { return u.Password.Valid && u.Password.String != "" }

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
