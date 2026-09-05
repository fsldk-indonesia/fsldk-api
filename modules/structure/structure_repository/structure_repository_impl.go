package structure_repository

import (
	"context"
	"errors"
	"fmt"

	"fsldk-api/modules/structure/structure_dto"
	"fsldk-api/modules/structure/structure_model"

	"gorm.io/gorm"
)

type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new instance of structure Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) List(ctx context.Context, f structure_dto.Filter) ([]structure_model.Structure, int64, error) {
	query := r.db.WithContext(ctx).Model(&structure_model.Structure{})

	if f.Search != "" {
		search := "%" + f.Search + "%"
		query = query.Where("batch LIKE ? OR period LIKE ? OR structureName LIKE ?", search, search, search)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count structures: %w", err)
	}

	if f.SortBy != "" {
		order := "DESC"
		if f.SortOrder == "asc" {
			order = "ASC"
		}
		query = query.Order(f.SortBy + " " + order)
	} else {
		query = query.Order("createdDate DESC")
	}

	if f.Limit > 0 {
		query = query.Limit(f.Limit).Offset(f.Offset)
	}

	var structures []structure_model.Structure
	if err := query.Find(&structures).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch structures: %w", err)
	}

	return structures, total, nil
}

func (r *repositoryImpl) FindByID(ctx context.Context, id int64) (structure_model.Structure, error) {
	var s structure_model.Structure
	err := r.db.WithContext(ctx).First(&s, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s, ErrNotFound
		}
		return s, fmt.Errorf("failed to find structure: %w", err)
	}
	return s, nil
}

func (r *repositoryImpl) Create(ctx context.Context, s structure_model.Structure, authorID int64) (int64, error) {
	s.CreatedBy = &authorID
	if err := r.db.WithContext(ctx).Create(&s).Error; err != nil {
		return 0, fmt.Errorf("failed to create structure: %w", err)
	}
	return s.StructureID, nil
}

func (r *repositoryImpl) Update(ctx context.Context, id int64, s structure_model.Structure, updatedBy int64) error {
	updates := map[string]interface{}{
		"batch":                s.Batch,
		"period":               s.Period,
		"structureName":        s.StructureName,
		"structureDescription": s.StructureDescription,
		"updatedBy":            updatedBy,
	}

	if s.LogoImage != nil {
		updates["logoImage"] = *s.LogoImage
	}
	if s.StructureImage != nil {
		updates["structureImage"] = *s.StructureImage
	}

	res := r.db.WithContext(ctx).Model(&structure_model.Structure{}).Where("structureID = ?", id).Updates(updates)
	if res.Error != nil {
		return fmt.Errorf("failed to update structure: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *repositoryImpl) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&structure_model.Structure{}, id)
	if res.Error != nil {
		return fmt.Errorf("failed to delete structure: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
