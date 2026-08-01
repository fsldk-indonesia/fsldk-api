package role_repository

import (
	"context"
	"database/sql"
	"errors"

	"fsldk-api/modules/role/role_model"

	"github.com/jmoiron/sqlx"
)

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) IDByName(ctx context.Context, name string) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, "SELECT roleID FROM ms_role WHERE roleName = ? LIMIT 1", name)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (role_model.Role, error) {
	var role role_model.Role
	err := r.db.GetContext(ctx, &role,
		`SELECT roleID, roleName, roleDescription, isSystemRole, isActive, createdDate, 0 AS userCount
		 FROM ms_role WHERE roleID = ? LIMIT 1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return role_model.Role{}, ErrNotFound
	}
	return role, err
}

func (r *RepositoryImpl) ListWithUserCount(ctx context.Context, search string) ([]role_model.Role, error) {
	q := `SELECT r.roleID, r.roleName, r.roleDescription, r.isSystemRole, r.isActive, r.createdDate,
			(SELECT COUNT(*) FROM ms_user u WHERE u.roleID = r.roleID) AS userCount
		 FROM ms_role r`
	args := []interface{}{}
	if search != "" {
		q += " WHERE r.roleName LIKE ?"
		args = append(args, "%"+search+"%")
	}
	q += " ORDER BY r.roleID ASC"
	var out []role_model.Role
	err := r.db.SelectContext(ctx, &out, q, args...)
	return out, err
}

func (r *RepositoryImpl) Create(ctx context.Context, name, desc string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_role (roleName, roleDescription, isSystemRole, isActive, createdDate)
		 VALUES (?, ?, 0, 1, NOW())`, name, desc)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, name, desc string, isActive bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_role SET roleName = ?, roleDescription = ?, isActive = ?, updatedDate = NOW() WHERE roleID = ?`,
		name, desc, isActive, id)
	return err
}

func (r *RepositoryImpl) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM ms_role WHERE roleID = ?", id)
	return err
}

func (r *RepositoryImpl) CountUsers(ctx context.Context, id int64) (int, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_user WHERE roleID = ?", id)
	return n, err
}

func (r *RepositoryImpl) PermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	var ids []int64
	err := r.db.SelectContext(ctx, &ids, "SELECT permissionID FROM map_role_permission WHERE roleID = ?", roleID)
	return ids, err
}

func (r *RepositoryImpl) PermissionNames(ctx context.Context, roleID int64) ([]string, error) {
	var names []string
	err := r.db.SelectContext(ctx, &names,
		`SELECT p.permissionName FROM lk_permission p
		 JOIN map_role_permission mrp ON mrp.permissionID = p.permissionID
		 WHERE mrp.roleID = ? ORDER BY p.moduleName`, roleID)
	return names, err
}

func (r *RepositoryImpl) UsersByRole(ctx context.Context, roleID int64) ([]role_model.RoleUser, error) {
	var out []role_model.RoleUser
	err := r.db.SelectContext(ctx, &out,
		`SELECT userID, fullName, email, isActive FROM ms_user WHERE roleID = ? ORDER BY fullName`, roleID)
	return out, err
}

func (r *RepositoryImpl) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
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
