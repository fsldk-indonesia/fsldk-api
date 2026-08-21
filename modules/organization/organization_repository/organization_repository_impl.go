package organization_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"

	"gorm.io/gorm"
)

const selectCols = "o.organizationID, o.organizationTypeCode, o.parentOrganizationID, " +
	"p.organizationName AS parentOrganizationName, o.organizationName, o.organizationCode, " +
	"o.provinceName, o.cityName, o.contactEmail, o.contactPhone, o.isActive, o.photoURL, o.createdDate"

const joinParent = "LEFT JOIN ms_organization p ON p.organizationID = o.parentOrganizationID"

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) FindByID(ctx context.Context, id int64) (organization_model.Organization, error) {
	var o organization_model.Organization
	err := r.db.WithContext(ctx).Table("ms_organization o").
		Select(selectCols).Joins(joinParent).
		Where("o.organizationID = ?", id).Take(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return organization_model.Organization{}, ErrNotFound
	}
	return o, err
}

func (r *RepositoryImpl) ExistsByCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_organization").Where("organizationCode = ?", code).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) TypeCodeByID(ctx context.Context, id int64) (string, error) {
	var code string
	err := r.db.WithContext(ctx).Table("ms_organization").
		Select("organizationTypeCode").Where("organizationID = ?", id).Take(&code).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	return code, err
}

func (r *RepositoryImpl) IsChildOf(ctx context.Context, targetID, parentID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_organization").
		Where("organizationID = ? AND parentOrganizationID = ?", targetID, parentID).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) applyFilter(q *gorm.DB, f organization_dto.ListFilter) *gorm.DB {
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(o.organizationName LIKE ? OR o.organizationCode LIKE ?)", like, like)
	}
	return q
}

func (r *RepositoryImpl) paginate(ctx context.Context, base *gorm.DB, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	orderBy := f.OrderBy
	if orderBy == "" {
		orderBy = "o.organizationName ASC"
	}
	var out []organization_model.Organization
	q := base.Select(selectCols).Order(orderBy)
	if f.Limit > 0 {
		q = q.Limit(f.Limit).Offset(f.Offset)
	}
	err := q.Find(&out).Error
	return out, total, err
}

func (r *RepositoryImpl) ListAll(ctx context.Context, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_organization o").Joins(joinParent)
	if f.OrganizationTypeCode != "" {
		base = base.Where("o.organizationTypeCode = ?", f.OrganizationTypeCode)
	}
	base = r.applyFilter(base, f)
	return r.paginate(ctx, base, f)
}

func (r *RepositoryImpl) ListByTypes(ctx context.Context, types []string, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_organization o").Joins(joinParent).
		Where("o.organizationTypeCode IN ?", types)
	if f.OrganizationTypeCode != "" {
		base = base.Where("o.organizationTypeCode = ?", f.OrganizationTypeCode)
	}
	base = r.applyFilter(base, f)
	return r.paginate(ctx, base, f)
}

func (r *RepositoryImpl) ListSelfAndChildren(ctx context.Context, id int64, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	base := r.db.WithContext(ctx).Table("ms_organization o").Joins(joinParent).
		Where("(o.organizationID = ? OR o.parentOrganizationID = ?)", id, id)
	if f.OrganizationTypeCode != "" {
		base = base.Where("o.organizationTypeCode = ?", f.OrganizationTypeCode)
	}
	base = r.applyFilter(base, f)
	return r.paginate(ctx, base, f)
}

func (r *RepositoryImpl) Children(ctx context.Context, id int64) ([]organization_model.Organization, error) {
	var out []organization_model.Organization
	err := r.db.WithContext(ctx).Table("ms_organization o").
		Select(selectCols).Joins(joinParent).
		Where("o.parentOrganizationID = ?", id).
		Order("o.organizationName ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) ListActiveByType(ctx context.Context, typeCode string) ([]organization_model.Organization, error) {
	var out []organization_model.Organization
	err := r.db.WithContext(ctx).Table("ms_organization o").
		Select(selectCols).Joins(joinParent).
		Where("o.organizationTypeCode = ? AND o.isActive = 1", typeCode).
		Order("o.organizationName ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) RootPuskomnasID(ctx context.Context) (int64, error) {
	var id int64
	err := r.db.WithContext(ctx).Table("ms_organization").
		Select("organizationID").Where("organizationTypeCode = 'PUSKOMNAS'").Take(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrNotFound
	}
	return id, err
}

func (r *RepositoryImpl) Create(ctx context.Context, p organization_model.CreateParams) (int64, error) {
	values := map[string]interface{}{
		"organizationTypeCode": p.OrganizationTypeCode,
		"parentOrganizationID": p.ParentOrganizationID,
		"organizationName":     p.OrganizationName,
		"organizationCode":     p.OrganizationCode,
		"provinceName":         p.ProvinceName,
		"cityName":             p.CityName,
		"contactEmail":         p.ContactEmail,
		"contactPhone":         p.ContactPhone,
		"isActive":             true,
		"createdDate":          time.Now(),
		"createdBy":            p.CreatedBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_organization").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) Update(ctx context.Context, id int64, name, province, city, email, phone, photoURL string, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_organization").Where("organizationID = ?", id).Updates(map[string]interface{}{
		"organizationName": name,
		"provinceName":     province,
		"cityName":         city,
		"contactEmail":     email,
		"contactPhone":     phone,
		"photoURL":         photoURL,
		"updatedDate":      time.Now(),
		"updatedBy":        updatedBy,
	}).Error
}

func (r *RepositoryImpl) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_organization").Where("organizationID = ?", id).Updates(map[string]interface{}{
		"isActive":    active,
		"updatedDate": time.Now(),
		"updatedBy":   updatedBy,
	}).Error
}
