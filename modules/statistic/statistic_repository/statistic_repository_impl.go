package statistic_repository

import (
	"context"
	"fmt"

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

func (r *repositoryImpl) CountByCity(ctx context.Context) ([]statistic_dto.CityCount, error) {
	var rows []statistic_dto.CityCount
	err := r.db.WithContext(ctx).Table(constants.TableOrganization).
		Select("cityName, COUNT(*) AS count").
		Where("organizationTypeCode = ? AND isActive = 1 AND cityName IS NOT NULL AND cityName != ''", constants.OrgTypeLDK).
		Group("cityName").
		Order("cityName ASC").
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

func (r *repositoryImpl) Directory(ctx context.Context, q dto.ListQuery, f statistic_dto.DirectoryFilter) ([]statistic_dto.DirectoryEntry, int, error) {
	// Self-LEFT JOIN untuk resolve nama induk (parentOrganizationID -> p)
	// — pola sama persis dengan organization_repository_impl.go, supaya
	// frontend bisa menyusun direktori sebagai tree Puskomnas -> Puskomda ->
	// LDK alih-alih daftar datar. LEFT (bukan INNER) karena root Puskomnas
	// tidak punya induk (parentOrganizationID NULL).
	db := r.db.WithContext(ctx).Table(constants.TableOrganization+" o").
		Joins("LEFT JOIN "+constants.TableOrganization+" p ON p.organizationID = o.parentOrganizationID").
		// LEFT JOIN res/l hanya untuk level badge publik — isCurrent=1 AND
		// isPublished=1 sengaja disaring di kondisi JOIN (bukan WHERE) supaya
		// organisasi tanpa hasil levelisasi published (Puskomda/Puskomnas,
		// atau LDK yang belum publish) tetap ikut, cuma levelCode/levelLabel-
		// nya NULL — sama aturannya dengan LevelDistribution() di atas.
		Joins("LEFT JOIN "+constants.TableLevelisasiResult+" res ON res.organizationID = o.organizationID AND res.isCurrent = 1 AND res.isPublished = 1").
		Joins("LEFT JOIN "+constants.TableLevel+" l ON l.levelCode = res.levelCode").
		Where("o.isActive = 1")
	if len(f.OrganizationTypeCodes) > 0 {
		db = db.Where("o.organizationTypeCode IN ?", f.OrganizationTypeCodes)
	}
	if len(f.ProvinceNames) > 0 {
		db = db.Where("o.provinceName IN ?", f.ProvinceNames)
	}
	if len(f.CityNames) > 0 {
		db = db.Where("o.cityName IN ?", f.CityNames)
	}
	if len(f.ParentOrganizationIDs) > 0 {
		db = db.Where("(o.organizationID IN ? OR o.parentOrganizationID IN ?)", f.ParentOrganizationIDs, f.ParentOrganizationIDs)
	}
	if len(f.LevelCodes) > 0 {
		db = db.Where("res.levelCode IN ?", f.LevelCodes)
	}
	if q.Search != "" {
		db = db.Where("o.organizationName LIKE ?", "%"+q.Search+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Puskomnas/Puskomda diurutkan lebih dulu dari LDK secara eksplisit (BUKAN
	// alfabetis murni — "LDK" < "PUSKOMDA" < "PUSKOMNAS" secara string akan
	// menaruh seluruh LDK duluan). Direktori publik menampilkan ketiga tipe
	// sekaligus dibatasi q.Limit (maks 100, lihat base/dto/pagination.go),
	// sedangkan LDK sendiri sudah ratusan baris — tanpa urutan eksplisit ini,
	// baris Puskomnas/Puskomda bisa kehabisan slot sama sekali dan hilang dari
	// peta jaringan (frontend menampilkan grup Puskomda "di luar hasil filter"
	// walau LDK-nya tetap tampak, lihat PuskomdaGroup.puskomda di
	// statistic.index.page.ts). Total Puskomnas+Puskomda jauh di bawah 100
	// (39 per 2026-10), jadi menaruhnya duluan menjamin keduanya selalu ikut.
	orderByTypePriority := fmt.Sprintf(
		"CASE o.organizationTypeCode WHEN '%s' THEN 0 WHEN '%s' THEN 1 WHEN '%s' THEN 2 ELSE 3 END ASC",
		constants.OrgTypePuskomnas, constants.OrgTypePuskomda, constants.OrgTypeLDK,
	)
	// Urutan sekunder: default alfabetis, ATAU level Levelisasi tertinggi
	// dulu (Mandiri -> Madya -> Mula -> Pra Mula -> belum ada hasil) kalau
	// q.Sort="level" (frontend: dropdown "Urutkan" di sebelah tombol Filter).
	// levelCode TIDAK bisa diurutkan alfabetis apa adanya ("MADYA" < "MANDIRI"
	// < "MULA" < "PRA_MULA" secara string, bukan urutan capaian) — makanya
	// pakai CASE, bukan cuma "res.levelCode ASC/DESC". Puskomda/Puskomnas
	// (levelCode selalu NULL) jatuh ke bucket terakhir sama seperti LDK yang
	// belum publish hasil, tapi itu tidak kelihatan karena mereka sudah
	// masuk tier terpisah lewat orderByTypePriority di atas.
	orderBy := orderByTypePriority + ", o.organizationName ASC"
	if q.Sort == "level" {
		orderBy = orderByTypePriority +
			", CASE res.levelCode WHEN 'MANDIRI' THEN 0 WHEN 'MADYA' THEN 1 WHEN 'MULA' THEN 2 WHEN 'PRA_MULA' THEN 3 ELSE 4 END ASC" +
			", o.organizationName ASC"
	}

	var rows []statistic_dto.DirectoryEntry
	err := db.Select("o.organizationID, o.organizationTypeCode, o.organizationName, o.organizationCode, o.provinceName, o.cityName, " +
		"o.contactEmail, o.contactPhone, o.websiteURL, o.photoURL, " +
		"o.parentOrganizationID, p.organizationName AS parentOrganizationName, " +
		"res.levelCode, l.levelLabel").
		Order(orderBy).
		Limit(q.Limit).Offset(q.Offset()).
		Find(&rows).Error
	return rows, int(total), err
}
