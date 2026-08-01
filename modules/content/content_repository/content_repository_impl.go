package content_repository

import (
	"context"
	"errors"

	"fsldk-api/modules/content/content_dto"
	"fsldk-api/modules/content/content_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) ListContent(ctx context.Context, activeOnly bool) ([]content_model.Content, error) {
	q := r.db.WithContext(ctx).Table("ms_cms_content")
	if activeOnly {
		q = q.Where("isActive = 1")
	}
	var out []content_model.Content
	err := q.Order("sortOrder, contentKey").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) GetContentByKey(ctx context.Context, key string) (content_model.Content, error) {
	var c content_model.Content
	err := r.db.WithContext(ctx).Table("ms_cms_content").Where("contentKey = ?", key).Take(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return content_model.Content{}, ErrNotFound
	}
	return c, err
}

func (r *RepositoryImpl) UpdateContent(ctx context.Context, key, title, body string, updatedBy int64) error {
	res := r.db.WithContext(ctx).Table("ms_cms_content").Where("contentKey = ?", key).Updates(map[string]interface{}{
		"contentTitle": title,
		"contentBody":  body,
		"updatedBy":    updatedBy,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepositoryImpl) ListOrg(ctx context.Context, activeOnly bool) ([]content_model.OrgMember, error) {
	q := r.db.WithContext(ctx).Table("ms_organization_structure")
	if activeOnly {
		q = q.Where("isActive = 1")
	}
	var out []content_model.OrgMember
	err := q.Order("sortOrder, structureID").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) CreateOrg(ctx context.Context, m content_dto.OrgRequest, createdBy int64) (int64, error) {
	values := map[string]interface{}{
		"memberName": m.MemberName,
		"position":   m.Position,
		"photoURL":   nullStr(m.PhotoURL),
		"level":      nullStr(m.Level),
		"sortOrder":  m.SortOrder,
		"isActive":   true,
		"createdBy":  createdBy,
	}
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_organization_structure").Create(values).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) UpdateOrg(ctx context.Context, id int64, m content_dto.OrgRequest) error {
	res := r.db.WithContext(ctx).Table("ms_organization_structure").Where("structureID = ?", id).Updates(map[string]interface{}{
		"memberName": m.MemberName,
		"position":   m.Position,
		"photoURL":   nullStr(m.PhotoURL),
		"level":      nullStr(m.Level),
		"sortOrder":  m.SortOrder,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepositoryImpl) DeleteOrg(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_organization_structure WHERE structureID = ?", id).Error
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
