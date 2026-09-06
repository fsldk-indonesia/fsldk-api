package statistic_repository

import (
	"context"

	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/statistic/statistic_dto"

	"gorm.io/gorm"
)

type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new instance of statistic Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) CountByType(ctx context.Context) ([]statistic_dto.TypeCount, error) {
	var rows []statistic_dto.TypeCount
	err := r.db.WithContext(ctx).Table(constants.TableOrganization).
		Select("organizationTypeCode, COUNT(*) AS count").
		Where("isActive = 1").
		Group("organizationTypeCode").
		Find(&rows).Error
	return rows, err
}

func (r *repositoryImpl) CountByProvince(ctx context.Context) ([]statistic_dto.ProvinceCount, error) {
	var rows []statistic_dto.ProvinceCount
	err := r.db.WithContext(ctx).Table(constants.TableOrganization).
		Select("provinceName, COUNT(*) AS count").
		Where("organizationTypeCode = ? AND isActive = 1 AND provinceName IS NOT NULL AND provinceName != ''", constants.OrgTypeLDK).
		Group("provinceName").
		Order("count DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repositoryImpl) LevelDistribution(ctx context.Context) ([]statistic_dto.LevelCount, error) {
	var rows []statistic_dto.LevelCount
	// isPublished = 1 sengaja disaring — hasil levelisasi yang belum
	// dipublikasikan masih bisa berubah lewat proses review, tidak layak
	// ditampilkan sebagai data resmi ke publik.
	err := r.db.WithContext(ctx).Table(constants.TableLevelisasiResult+" res").
		Select("res.levelCode, l.levelLabel, COUNT(*) AS count").
		Joins("JOIN "+constants.TableLevel+" l ON l.levelCode = res.levelCode").
		Joins("JOIN "+constants.TableOrganization+" o ON o.organizationID = res.organizationID").
		Where("res.isCurrent = 1 AND res.isPublished = 1 AND o.isActive = 1").
		Group("res.levelCode, l.levelLabel").
		Order("l.sortOrder ASC").
		Find(&rows).Error
	return rows, err
}

func (r *repositoryImpl) CountActiveKader(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(constants.TableKader).
		Where("status = ?", constants.KaderStatusActive).
		Count(&count).Error
	return int(count), err
}

func (r *repositoryImpl) Directory(ctx context.Context, q dto.ListQuery, organizationTypeCode, provinceName string) ([]statistic_dto.DirectoryEntry, int, error) {
	db := r.db.WithContext(ctx).Table(constants.TableOrganization).Where("isActive = 1")
	if organizationTypeCode != "" {
		db = db.Where("organizationTypeCode = ?", organizationTypeCode)
	}
	if provinceName != "" {
		db = db.Where("provinceName = ?", provinceName)
	}
	if q.Search != "" {
		db = db.Where("organizationName LIKE ?", "%"+q.Search+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []statistic_dto.DirectoryEntry
	err := db.Select("organizationID, organizationTypeCode, organizationName, provinceName, cityName, photoURL").
		Order("organizationTypeCode ASC, organizationName ASC").
		Limit(q.Limit).Offset(q.Offset()).
		Find(&rows).Error
	return rows, int(total), err
}
