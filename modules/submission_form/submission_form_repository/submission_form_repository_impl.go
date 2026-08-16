package submission_form_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/submission_form/submission_form_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

// ---------- Form ----------

func (r *RepositoryImpl) CreateForm(ctx context.Context, formCode, formName, description string, createdBy sql.NullInt64) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_submission_form").Create(map[string]interface{}{
			"formCode":    formCode,
			"formName":    formName,
			"description": sql.NullString{String: description, Valid: description != ""},
			"isActive":    true,
			"createdDate": time.Now(),
			"createdBy":   createdBy,
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) ExistsFormCode(ctx context.Context, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table("ms_submission_form").Where("formCode = ?", code).Count(&count).Error
	return count > 0, err
}

func (r *RepositoryImpl) ListForms(ctx context.Context) ([]submission_form_model.Form, error) {
	var out []submission_form_model.Form
	err := r.db.WithContext(ctx).Table("ms_submission_form").Order("formName ASC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) FindFormByID(ctx context.Context, id int64) (submission_form_model.Form, error) {
	var f submission_form_model.Form
	err := r.db.WithContext(ctx).Table("ms_submission_form").Where("formID = ?", id).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_form_model.Form{}, ErrNotFound
	}
	return f, err
}

// ---------- Version ----------

func (r *RepositoryImpl) CreateVersion(ctx context.Context, formID int64, versionNumber int, createdBy sql.NullInt64) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_submission_form_version").Create(map[string]interface{}{
			"formID":        formID,
			"versionNumber": versionNumber,
			"status":        "DRAFT",
			"createdDate":   time.Now(),
			"createdBy":     createdBy,
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) MaxVersionNumber(ctx context.Context, formID int64) (int, error) {
	var max sql.NullInt64
	err := r.db.WithContext(ctx).Table("ms_submission_form_version").
		Select("MAX(versionNumber)").Where("formID = ?", formID).Scan(&max).Error
	if err != nil {
		return 0, err
	}
	return int(max.Int64), nil
}

func (r *RepositoryImpl) FindVersionByID(ctx context.Context, id int64) (submission_form_model.Version, error) {
	var v submission_form_model.Version
	err := r.db.WithContext(ctx).Table("ms_submission_form_version").Where("versionID = ?", id).Take(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_form_model.Version{}, ErrNotFound
	}
	return v, err
}

func (r *RepositoryImpl) ListVersionsByForm(ctx context.Context, formID int64) ([]submission_form_model.Version, error) {
	var out []submission_form_model.Version
	err := r.db.WithContext(ctx).Table("ms_submission_form_version").
		Where("formID = ?", formID).Order("versionNumber DESC").Find(&out).Error
	return out, err
}

func (r *RepositoryImpl) PublishVersion(ctx context.Context, id int64, publishedBy int64) error {
	return r.db.WithContext(ctx).Table("ms_submission_form_version").Where("versionID = ?", id).Updates(map[string]interface{}{
		"status":        "PUBLISHED",
		"publishedDate": time.Now(),
		"publishedBy":   publishedBy,
		"updatedDate":   time.Now(),
		"updatedBy":     publishedBy,
	}).Error
}

func (r *RepositoryImpl) ArchiveOtherPublished(ctx context.Context, formID, exceptVersionID int64) error {
	return r.db.WithContext(ctx).Table("ms_submission_form_version").
		Where("formID = ? AND versionID <> ? AND status = 'PUBLISHED'", formID, exceptVersionID).
		Update("status", "ARCHIVED").Error
}

// ---------- Section ----------

func (r *RepositoryImpl) CreateSection(ctx context.Context, versionID int64, code, label string, sortOrder int, description sql.NullString) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_submission_form_section").Create(map[string]interface{}{
			"versionID":    versionID,
			"sectionCode":  code,
			"sectionLabel": label,
			"sortOrder":    sortOrder,
			"description":  description,
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindSectionByID(ctx context.Context, id int64) (submission_form_model.Section, error) {
	var s submission_form_model.Section
	err := r.db.WithContext(ctx).Table("ms_submission_form_section").Where("sectionID = ?", id).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_form_model.Section{}, ErrNotFound
	}
	return s, err
}

func (r *RepositoryImpl) UpdateSection(ctx context.Context, id int64, label string, sortOrder int, description sql.NullString) error {
	return r.db.WithContext(ctx).Table("ms_submission_form_section").Where("sectionID = ?", id).Updates(map[string]interface{}{
		"sectionLabel": label,
		"sortOrder":    sortOrder,
		"description":  description,
	}).Error
}

func (r *RepositoryImpl) DeleteSection(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_submission_form_section WHERE sectionID = ?", id).Error
}

func (r *RepositoryImpl) ListSectionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Section, error) {
	var out []submission_form_model.Section
	err := r.db.WithContext(ctx).Table("ms_submission_form_section").
		Where("versionID = ?", versionID).Order("sortOrder ASC").Find(&out).Error
	return out, err
}

// ---------- Field ----------

func fieldValues(p submission_form_model.FieldParams) map[string]interface{} {
	return map[string]interface{}{
		"sectionID":            p.SectionID,
		"fieldCode":            p.FieldCode,
		"fieldLabel":           p.FieldLabel,
		"fieldType":            p.FieldType,
		"isRequired":           p.IsRequired,
		"sortOrder":            p.SortOrder,
		"validationRuleJSON":   p.ValidationRuleJSON,
		"conditionalOnFieldID": p.ConditionalOnFieldID,
		"conditionalRuleJSON":  p.ConditionalRuleJSON,
		"helpText":             p.HelpText,
	}
}

func (r *RepositoryImpl) CreateField(ctx context.Context, p submission_form_model.FieldParams) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_submission_form_field").Create(fieldValues(p)).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindFieldByID(ctx context.Context, id int64) (submission_form_model.Field, error) {
	var f submission_form_model.Field
	err := r.db.WithContext(ctx).Table("ms_submission_form_field").Where("fieldID = ?", id).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_form_model.Field{}, ErrNotFound
	}
	return f, err
}

func (r *RepositoryImpl) UpdateField(ctx context.Context, id int64, p submission_form_model.FieldParams) error {
	values := fieldValues(p)
	delete(values, "sectionID")
	delete(values, "fieldCode")
	return r.db.WithContext(ctx).Table("ms_submission_form_field").Where("fieldID = ?", id).Updates(values).Error
}

func (r *RepositoryImpl) DeleteField(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_submission_form_field WHERE fieldID = ?", id).Error
}

func (r *RepositoryImpl) ListFieldsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Field, error) {
	var out []submission_form_model.Field
	err := r.db.WithContext(ctx).Table("ms_submission_form_field f").
		Select("f.*").
		Joins("JOIN ms_submission_form_section s ON s.sectionID = f.sectionID").
		Where("s.versionID = ?", versionID).
		Order("f.sortOrder ASC").Find(&out).Error
	return out, err
}

// ---------- Option ----------

func (r *RepositoryImpl) CreateOption(ctx context.Context, fieldID int64, value, label string, sortOrder int) (int64, error) {
	var newID int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_submission_form_field_option").Create(map[string]interface{}{
			"fieldID":     fieldID,
			"optionValue": value,
			"optionLabel": label,
			"sortOrder":   sortOrder,
			"isActive":    true,
		}).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newID).Error
	})
	return newID, err
}

func (r *RepositoryImpl) FindOptionByID(ctx context.Context, id int64) (submission_form_model.Option, error) {
	var o submission_form_model.Option
	err := r.db.WithContext(ctx).Table("ms_submission_form_field_option").Where("optionID = ?", id).Take(&o).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return submission_form_model.Option{}, ErrNotFound
	}
	return o, err
}

func (r *RepositoryImpl) UpdateOption(ctx context.Context, id int64, value, label string, sortOrder int, isActive bool) error {
	return r.db.WithContext(ctx).Table("ms_submission_form_field_option").Where("optionID = ?", id).Updates(map[string]interface{}{
		"optionValue": value,
		"optionLabel": label,
		"sortOrder":   sortOrder,
		"isActive":    isActive,
	}).Error
}

func (r *RepositoryImpl) DeleteOption(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM ms_submission_form_field_option WHERE optionID = ?", id).Error
}

func (r *RepositoryImpl) ListOptionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Option, error) {
	var out []submission_form_model.Option
	err := r.db.WithContext(ctx).Table("ms_submission_form_field_option o").
		Select("o.*").
		Joins("JOIN ms_submission_form_field f ON f.fieldID = o.fieldID").
		Joins("JOIN ms_submission_form_section s ON s.sectionID = f.sectionID").
		Where("s.versionID = ?", versionID).
		Order("o.sortOrder ASC").Find(&out).Error
	return out, err
}

// ---------- Lookup untuk validasi status DRAFT ----------

func (r *RepositoryImpl) VersionStatusBySectionID(ctx context.Context, sectionID int64) (string, error) {
	var status string
	err := r.db.WithContext(ctx).Table("ms_submission_form_version v").
		Select("v.status").
		Joins("JOIN ms_submission_form_section s ON s.versionID = v.versionID").
		Where("s.sectionID = ?", sectionID).Take(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	return status, err
}

func (r *RepositoryImpl) VersionStatusByFieldID(ctx context.Context, fieldID int64) (string, error) {
	var status string
	err := r.db.WithContext(ctx).Table("ms_submission_form_version v").
		Select("v.status").
		Joins("JOIN ms_submission_form_section s ON s.versionID = v.versionID").
		Joins("JOIN ms_submission_form_field f ON f.sectionID = s.sectionID").
		Where("f.fieldID = ?", fieldID).Take(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	return status, err
}

func (r *RepositoryImpl) VersionIDByFieldID(ctx context.Context, fieldID int64) (int64, error) {
	var versionID int64
	err := r.db.WithContext(ctx).Table("ms_submission_form_section s").
		Select("s.versionID").
		Joins("JOIN ms_submission_form_field f ON f.sectionID = s.sectionID").
		Where("f.fieldID = ?", fieldID).Take(&versionID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, ErrNotFound
	}
	return versionID, err
}

// ---------- Clone ----------

func (r *RepositoryImpl) CloneVersionStructure(ctx context.Context, fromVersionID, toVersionID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sections []submission_form_model.Section
		if err := tx.Table("ms_submission_form_section").Where("versionID = ?", fromVersionID).
			Order("sortOrder ASC").Find(&sections).Error; err != nil {
			return err
		}

		sectionIDMap := make(map[int64]int64, len(sections))
		for _, sec := range sections {
			var newSectionID int64
			if err := tx.Table("ms_submission_form_section").Create(map[string]interface{}{
				"versionID":    toVersionID,
				"sectionCode":  sec.SectionCode,
				"sectionLabel": sec.SectionLabel,
				"sortOrder":    sec.SortOrder,
				"description":  sec.Description,
			}).Error; err != nil {
				return err
			}
			if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newSectionID).Error; err != nil {
				return err
			}
			sectionIDMap[sec.SectionID] = newSectionID
		}

		var fields []submission_form_model.Field
		if err := tx.Table("ms_submission_form_field f").
			Select("f.*").
			Joins("JOIN ms_submission_form_section s ON s.sectionID = f.sectionID").
			Where("s.versionID = ?", fromVersionID).
			Order("f.sortOrder ASC").Find(&fields).Error; err != nil {
			return err
		}

		fieldIDMap := make(map[int64]int64, len(fields))
		for _, fld := range fields {
			var newFieldID int64
			if err := tx.Table("ms_submission_form_field").Create(map[string]interface{}{
				"sectionID":           sectionIDMap[fld.SectionID],
				"fieldCode":           fld.FieldCode,
				"fieldLabel":          fld.FieldLabel,
				"fieldType":           fld.FieldType,
				"isRequired":          fld.IsRequired,
				"sortOrder":           fld.SortOrder,
				"validationRuleJSON":  fld.ValidationRuleJSON,
				"conditionalRuleJSON": fld.ConditionalRuleJSON,
				"helpText":            fld.HelpText,
			}).Error; err != nil {
				return err
			}
			if err := tx.Raw("SELECT LAST_INSERT_ID()").Scan(&newFieldID).Error; err != nil {
				return err
			}
			fieldIDMap[fld.FieldID] = newFieldID
		}
		// Pemetaan conditionalOnFieldID dilakukan setelah seluruh field baru
		// dibuat, karena field pemicu (conditional trigger) bisa saja muncul
		// setelah field yang bergantung padanya dalam urutan sortOrder.
		for _, fld := range fields {
			if !fld.ConditionalOnFieldID.Valid {
				continue
			}
			newConditionalID, ok := fieldIDMap[fld.ConditionalOnFieldID.Int64]
			if !ok {
				continue
			}
			if err := tx.Table("ms_submission_form_field").Where("fieldID = ?", fieldIDMap[fld.FieldID]).
				Update("conditionalOnFieldID", newConditionalID).Error; err != nil {
				return err
			}
		}

		var options []submission_form_model.Option
		if err := tx.Table("ms_submission_form_field_option o").
			Select("o.*").
			Joins("JOIN ms_submission_form_field f ON f.fieldID = o.fieldID").
			Joins("JOIN ms_submission_form_section s ON s.sectionID = f.sectionID").
			Where("s.versionID = ?", fromVersionID).Find(&options).Error; err != nil {
			return err
		}
		for _, opt := range options {
			newFieldID, ok := fieldIDMap[opt.FieldID]
			if !ok {
				continue
			}
			if err := tx.Table("ms_submission_form_field_option").Create(map[string]interface{}{
				"fieldID":     newFieldID,
				"optionValue": opt.OptionValue,
				"optionLabel": opt.OptionLabel,
				"sortOrder":   opt.SortOrder,
				"isActive":    opt.IsActive,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
