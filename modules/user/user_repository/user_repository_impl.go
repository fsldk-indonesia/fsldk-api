package user_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/user/user_model"

	"github.com/jmoiron/sqlx"
)

const userSelectCols = `u.userID, u.roleID, r.roleName, u.fullName, u.email, u.username, u.password,
	u.googleID, u.emailVerifiedDate, u.phoneNumber, u.photoURL, u.mustChangePassword,
	u.isActive, u.createdDate, u.createdBy, u.updatedDate, u.updatedBy`

// RepositoryImpl adalah implementasi Repository berbasis sqlx.
type RepositoryImpl struct{ db *sqlx.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *sqlx.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (user_model.User, error) {
	var u user_model.User
	q := `SELECT ` + userSelectCols + ` FROM ms_user u JOIN ms_role r ON r.roleID = u.roleID WHERE ` + where + ` LIMIT 1`
	err := r.db.GetContext(ctx, &u, q, arg)
	if errors.Is(err, sql.ErrNoRows) {
		return user_model.User{}, ErrNotFound
	}
	return u, err
}

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (user_model.User, error) {
	return r.findOne(ctx, "u.userID = ?", id)
}

func (r *RepositoryImpl) FindByEmail(ctx context.Context, email string) (user_model.User, error) {
	return r.findOne(ctx, "u.email = ?", email)
}

func (r *RepositoryImpl) FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error) {
	return r.findOne(ctx, "u.googleID = ?", googleID)
}

func (r *RepositoryImpl) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var n int
	err := r.db.GetContext(ctx, &n, "SELECT COUNT(*) FROM ms_user WHERE email = ?", email)
	return n > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, p user_model.CreateParams) (int64, error) {
	var verified interface{}
	if p.EmailVerified {
		verified = time.Now()
	}
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO ms_user (roleID, fullName, email, password, googleID, emailVerifiedDate, isActive, createdDate, createdBy)
		 VALUES (?, ?, ?, ?, ?, ?, 1, NOW(), ?)`,
		p.RoleID, p.FullName, p.Email, p.Password, p.GoogleID, verified, p.CreatedBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RepositoryImpl) List(ctx context.Context, search string, roleID int64, limit, offset int, orderBy string) ([]user_model.User, int, error) {
	where := "WHERE u.isActive IN (0,1)"
	args := []interface{}{}
	if search != "" {
		where += " AND (u.fullName LIKE ? OR u.email LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if roleID > 0 {
		where += " AND u.roleID = ?"
		args = append(args, roleID)
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM ms_user u `+where, args...); err != nil {
		return nil, 0, err
	}

	q := `SELECT ` + userSelectCols + ` FROM ms_user u JOIN ms_role r ON r.roleID = u.roleID ` +
		where + ` ORDER BY ` + orderBy + ` LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	var out []user_model.User
	if err := r.db.SelectContext(ctx, &out, q, args...); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, fullName string, roleID int64, isActive bool, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_user SET fullName = ?, roleID = ?, isActive = ?, updatedDate = NOW(), updatedBy = ? WHERE userID = ?`,
		fullName, roleID, isActive, updatedBy, id)
	return err
}

func (r *RepositoryImpl) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_user SET isActive = ?, updatedDate = NOW(), updatedBy = ? WHERE userID = ?`, active, updatedBy, id)
	return err
}

func (r *RepositoryImpl) SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_user SET password = ?, mustChangePassword = ?, updatedDate = NOW() WHERE userID = ?`, hashed, mustChange, id)
	return err
}

func (r *RepositoryImpl) LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error {
	if markVerified {
		_, err := r.db.ExecContext(ctx,
			`UPDATE ms_user SET googleID = ?, emailVerifiedDate = COALESCE(emailVerifiedDate, NOW()), updatedDate = NOW() WHERE userID = ?`,
			googleID, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_user SET googleID = ?, updatedDate = NOW() WHERE userID = ?`, googleID, id)
	return err
}

func (r *RepositoryImpl) MarkEmailVerified(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_user SET emailVerifiedDate = NOW(), updatedDate = NOW() WHERE userID = ? AND emailVerifiedDate IS NULL`, id)
	return err
}

func (r *RepositoryImpl) SoftDelete(ctx context.Context, id int64, updatedBy int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE ms_user SET isActive = 0, updatedDate = NOW(), updatedBy = ? WHERE userID = ?`, updatedBy, id)
	return err
}

func (r *RepositoryImpl) LogLogin(ctx context.Context, userID int64, ip, ua, status string) error {
	if len(ua) > 255 {
		ua = ua[:255]
	}
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tr_user_login_log (userID, ipAddress, userAgent, loginStatus) VALUES (?, ?, ?, ?)`,
		userID, ip, ua, status)
	return err
}
