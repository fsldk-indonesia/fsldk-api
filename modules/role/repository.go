package role

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound dikembalikan bila role tidak ditemukan.
var ErrNotFound = errors.New("role tidak ditemukan")

// Repository adalah kontrak akses data role.
type Repository interface {
	IDByName(ctx context.Context, name string) (int64, error)
	FindByID(ctx context.Context, id int64) (Role, error)
	ListWithUserCount(ctx context.Context, search string) ([]Role, error)
	Create(ctx context.Context, name, desc string) (int64, error)
	Update(ctx context.Context, id int64, name, desc string, isActive bool) error
	Delete(ctx context.Context, id int64) error
	CountUsers(ctx context.Context, id int64) (int, error)
	PermissionIDs(ctx context.Context, roleID int64) ([]int64, error)
	PermissionNames(ctx context.Context, roleID int64) ([]string, error)
	SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error
	UsersByRole(ctx context.Context, roleID int64) ([]RoleUser, error)
}

// RoleUser adalah ringkasan pengguna pemilik sebuah role.
type RoleUser struct {
	UserID   int64  `db:"userID" json:"userID"`
	FullName string `db:"fullName" json:"fullName"`
	Email    string `db:"email" json:"email"`
	IsActive bool   `db:"isActive" json:"isActive"`
}

type repository struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &repository{db: db} }

func (r *repository) IDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, "SELECT roleID FROM ms_role WHERE roleName = ? LIMIT 1", name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (r *repository) FindByID(ctx context.Context, id int64) (Role, error) {
	var role Role
	err := r.db.GetContext(ctx, &role,
		`SELECT roleID, roleName, roleDescription, isSystemRole, isActive, createdDate, 0 AS userCount
		 FROM ms_role WHERE roleID = ? LIMIT 1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, ErrNotFound
	}
	return role, err
}

func (r *repository) ListWithUserCount(ctx context.Context, search string) ([]Role, error) {
	q := `SELECT r.roleID, r.roleName, r.roleDescription, r.isSystemRole, r.isActive, r.createdDate,
			(SELECT COUNT(*) FROM ms_user u WHERE u.roleID = r.roleID) AS userCount
		 FROM ms_role r`
	args := []interface{}{}
	if search != "" {
		q += " WHERE r.roleName LIKE ?"
		args = append(args, "%"+search+"%")
	}
	q += " ORDER BY r.roleID ASC"
	var out []Role
	err := r.db.SelectContext(ctx, &out, q, args...)
	return out, err
}

func (r *repository) Create(ctx context.Context, name, desc string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_role (roleName, roleDescription, isSystemRole, isActive, createdDate)
		 VALUES (?, ?, 0, 1, NOW())`, name, desc)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *repository) Update(ctx context.Context, id int64, name, desc string, isActive bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_role SET roleName = ?, roleDescription = ?, isActive = ?, updatedDate = NOW() WHERE roleID = ?`,
		name, desc, isActive, id)
	return err
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_role WHERE roleID = ?", id)
	return err
}

func (r *repository) CountUsers(ctx context.Context, id int64) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_user WHERE roleID = ?", id)
	return n, err
}

func (r *repository) PermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	var ids []int64
	err := r.db.SelectContext(ctx, &ids,
		"SELECT permissionID FROM map_role_permission WHERE roleID = ?", roleID)
	return ids, err
}

func (r *repository) PermissionNames(ctx context.Context, roleID int64) ([]string, error) {
	var names []string
	err := r.db.SelectContext(ctx, &names,
		`SELECT p.permissionName FROM lk_permission p
		 JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID
		 WHERE mrp.roleID = ? ORDER BY p.moduleName`, roleID)
	return names, err
}

func (r *repository) UsersByRole(ctx context.Context, roleID int64) ([]RoleUser, error) {
	var out []RoleUser
	err := r.db.SelectContext(ctx, &out,
		`SELECT userID, fullName, email, isActive FROM ms_user WHERE roleID = ? ORDER BY fullName`, roleID)
	return out, err
}

func (r *repository) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "DELETE FROM map_role_permission WHERE roleID = ?", roleID); err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO map_role_permission (roleID, permissionID) VALUES (?, ?)", roleID, pid); err != nil {
			return err
		}
	}
	return tx.Commit()
}
