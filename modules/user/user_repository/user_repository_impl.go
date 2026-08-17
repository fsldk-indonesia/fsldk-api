package user_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"

	"gorm.io/gorm"
)

const selectCols = "u.userID, u.roleID, r.roleName, u.fullName, u.email, u.username, u.password, " +
	"u.googleID, u.emailVerifiedDate, u.phoneNumber, u.photoURL, u.mustChangePassword, " +
	"u.isActive, u.createdDate, u.createdBy, u.updatedDate, u.updatedBy"

const joinRole = "JOIN ms_role r ON r.roleID = u.roleID"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) findOne(ctx context.Context, where string, arg interface{}) (user_model.User, error) {
	var u user_model.User
	err := r.db.WithContext(ctx).Table("ms_user u").
		Select(selectCols).
		Joins(joinRole).
		Where(where, arg).
		Take(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
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
	var count int64
	err := r.db.WithContext(ctx).Table("ms_user").Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) ExistsByEmailExcept(ctx context.Context, email string, exceptID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_user").Where("email = ? AND userID <> ?", email, exceptID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) Create(ctx context.Context, p user_model.CreateParams) (int64, error) {
	var verifiedAt interface{}
	if p.EmailVerified {
		verifiedAt = time.Now()
	}
	values := map[string]interface{}{
		"roleID":            p.RoleID,
		"fullName":          p.FullName,
		"email":             p.Email,
		"password":          p.Password,
		"googleID":          p.GoogleID,
		"photoURL":          p.PhotoURL,
		"emailVerifiedDate": verifiedAt,
		"isActive":          true,
		"createdDate":       time.Now(),
		"createdBy":         p.CreatedBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_user").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) List(ctx context.Context, f user_dto.ListFilter) ([]user_model.User, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_user u").Joins(joinRole)
	if f.Search != "" {
		like := "%" + f.Search + "%"
		base = base.Where("(u.fullName LIKE ? OR u.email LIKE ?)", like, like)
	}
	if f.RoleID > 0 {
		base = base.Where("u.roleID = ?", f.RoleID)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []user_model.User
	err := base.Select(selectCols).Order(f.OrderBy).Limit(f.Limit).Offset(f.Offset).Find(&out).Error
	return out, total, err
}

// SearchActive returns active users whose fullName matches search (or the
// most recently created ones when search is empty), for the @mention
// autocomplete in comments — deliberately unfiltered by role/permission
// since any verified user may mention any other active user (including
// themselves).
func (r *RepositoryImpl) SearchActive(ctx context.Context, search string, limit int) ([]user_model.User, error) {
	q := r.db.WithContext(ctx).Table("ms_user u").Joins(joinRole).Where("u.isActive = ?", true)
	if search != "" {
		q = q.Where("u.fullName LIKE ?", "%"+search+"%")
	}
	var out []user_model.User
	err := q.Select(selectCols).Order("u.fullName ASC").Limit(limit).Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, fullName, email string, roleID int64, isActive bool, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_user").Where("userID = ?", id).Updates(map[string]interface{}{
		"fullName":    fullName,
		"email":       email,
		"roleID":      roleID,
		"isActive":    isActive,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_user").Where("userID = ?", id).Updates(map[string]interface{}{
		"isActive":    active,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error {
	return r.db.WithContext(ctx).Table("ms_user").Where("userID = ?", id).Updates(map[string]interface{}{
		"password":           hashed,
		"mustChangePassword": mustChange,
		"updatedDate":        time.Now(),
	}).Error
}

func (r *RepositoryImpl) LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error {
	if markVerified {
		// COALESCE menjaga emailVerifiedDate yang sudah terisi (tidak ditimpa),
		// namun mengisi bila sebelumnya masih NULL.
		return r.db.WithContext(ctx).Exec(
			"UPDATE ms_user SET googleID = ?, emailVerifiedDate = COALESCE(emailVerifiedDate, NOW()), updatedDate = NOW() WHERE userID = ?",
			googleID, id).Error
	}
	return r.db.WithContext(ctx).Table("ms_user").Where("userID = ?", id).Updates(map[string]interface{}{
		"googleID":    googleID,
		"updatedDate": time.Now(),
	}).Error
}

func (r *RepositoryImpl) UpdatePhoto(ctx context.Context, id int64, photoURL string) error {
	return r.db.WithContext(ctx).Table("ms_user").Where("userID = ?", id).Updates(map[string]interface{}{
		"photoURL":    photoURL,
		"updatedDate": time.Now(),
	}).Error
}

func (r *RepositoryImpl) MarkEmailVerified(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Table("ms_user").
		Where("userID = ? AND emailVerifiedDate IS NULL", id).
		Updates(map[string]interface{}{"emailVerifiedDate": time.Now(), "updatedDate": time.Now()}).Error
}

func (r *RepositoryImpl) SoftDelete(ctx context.Context, id int64, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_user").Where("userID = ?", id).Updates(map[string]interface{}{
		"isActive":    false,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}

func (r *RepositoryImpl) LogLogin(ctx context.Context, userID int64, ip, ua, status string) error {
	if len(ua) > 255 {
		ua = ua[:255]
	}
	return r.db.WithContext(ctx).Table("tr_user_login_log").Create(map[string]interface{}{
		"userID":      userID,
		"ipAddress":   ip,
		"userAgent":   ua,
		"loginStatus": status,
		"loginDate":   time.Now(),
	}).Error
}
