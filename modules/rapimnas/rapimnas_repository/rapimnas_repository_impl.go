package rapimnas_repository

import (
	"context"

	"fsldk-api/modules/rapimnas/rapimnas_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Get(ctx context.Context) (*rapimnas_model.Aggregate, error) {
	var setting rapimnas_model.Setting
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_setting").
		Where("id = ?", rapimnas_model.SingletonID).Take(&setting).Error; err != nil {
		return nil, err
	}

	var galleryImages []rapimnas_model.GalleryImage
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_gallery_image").
		Order("sortOrder ASC").Find(&galleryImages).Error; err != nil {
		return nil, err
	}

	var homeCards []rapimnas_model.HomeCard
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_home_card").
		Order("sortOrder ASC").Find(&homeCards).Error; err != nil {
		return nil, err
	}

	var rundownDays []rapimnas_model.RundownDay
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_rundown_day").
		Order("sortOrder ASC").Find(&rundownDays).Error; err != nil {
		return nil, err
	}

	var rundownEvents []rapimnas_model.RundownEvent
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_rundown_event").
		Order("sortOrder ASC").Find(&rundownEvents).Error; err != nil {
		return nil, err
	}

	var resources []rapimnas_model.Resource
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_resource").
		Order("sortOrder ASC").Find(&resources).Error; err != nil {
		return nil, err
	}

	var pickupLocations []rapimnas_model.PickupLocation
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_pickup_location").
		Order("sortOrder ASC").Find(&pickupLocations).Error; err != nil {
		return nil, err
	}

	var contacts []rapimnas_model.Contact
	if err := r.db.WithContext(ctx).Table("ms_rapimnas_contact").
		Order("sortOrder ASC").Find(&contacts).Error; err != nil {
		return nil, err
	}

	return &rapimnas_model.Aggregate{
		Setting:         setting,
		GalleryImages:   galleryImages,
		HomeCards:       homeCards,
		RundownDays:     rundownDays,
		RundownEvents:   rundownEvents,
		Resources:       resources,
		PickupLocations: pickupLocations,
		Contacts:        contacts,
	}, nil
}

func settingValues(s rapimnas_model.Setting) map[string]interface{} {
	return map[string]interface{}{
		"heroBadgeText": s.HeroBadgeText, "heroTitle": s.HeroTitle, "heroDateRangeText": s.HeroDateRangeText,
		"heroTaglineQuote": s.HeroTaglineQuote, "heroImageUrl": s.HeroImageUrl,
		"countdownTargetDate": s.CountdownTargetDate,
		"feature1IconKey": s.Feature1IconKey, "feature1Title": s.Feature1Title, "feature1Desc": s.Feature1Desc,
		"feature2IconKey": s.Feature2IconKey, "feature2Title": s.Feature2Title, "feature2Desc": s.Feature2Desc,
		"ctaTitle": s.CtaTitle, "ctaDescription": s.CtaDescription, "ctaButtonLabel": s.CtaButtonLabel,
		"ctaMascotImageUrl": s.CtaMascotImageUrl,
		"footerContactEmail": s.FooterContactEmail, "footerCopyrightText": s.FooterCopyrightText,
		"footerIgHandle": s.FooterIgHandle, "footerIgUrl": s.FooterIgUrl,
		"footerTiktokHandle": s.FooterTiktokHandle, "footerTiktokUrl": s.FooterTiktokUrl,
		"jadwalHeaderSubtitle": s.JadwalHeaderSubtitle,
		"tentangTaglineQuote": s.TentangTaglineQuote, "tentangDescParagraph1": s.TentangDescParagraph1,
		"tentangDescParagraph2": s.TentangDescParagraph2, "tentangVisiText": s.TentangVisiText,
		"tentangMisiJSON": s.TentangMisiJSON, "tentangTujuanJSON": s.TentangTujuanJSON, "tentangKegiatanJSON": s.TentangKegiatanJSON,
		"pesertaEarlyBirdDateRange": s.PesertaEarlyBirdDateRange, "pesertaRegulerDateRange": s.PesertaRegulerDateRange,
		"pesertaHargaNonSemarangEarlyBird": s.PesertaHargaNonSemarangEarlyBird,
		"pesertaHargaNonSemarangReguler":   s.PesertaHargaNonSemarangReguler,
		"pesertaHargaSemarangEarlyBird":    s.PesertaHargaSemarangEarlyBird,
		"pesertaHargaSemarangReguler":      s.PesertaHargaSemarangReguler,
		"pesertaBankName": s.PesertaBankName, "pesertaBankAccountNumber": s.PesertaBankAccountNumber,
		"pesertaBankAccountHolder": s.PesertaBankAccountHolder,
		"pesertaGuidebookUrl": s.PesertaGuidebookUrl, "pesertaGoogleFormUrl": s.PesertaGoogleFormUrl,
		"pesertaMapEmbedUrl": s.PesertaMapEmbedUrl,
		"panitiaIsOpen": s.PanitiaIsOpen, "panitiaClosedMessage": s.PanitiaClosedMessage,
		"updatedBy": s.UpdatedBy,
	}
}

// Save menjalankan satu transaksi: upsert baris singleton, lalu untuk setiap
// 6 tabel anak, hapus seluruh baris lama dan buat ulang baris yang disubmit.
// RundownDay dibuat ulang SEBELUM RundownEvent, dan RundownEvent.DayID yang
// masuk (correlationID sementara dari rapimnas_service, lihat "Design
// decisions" item 2 di kepala plan ini) dipetakan ke id baru hasil
// auto-increment sesaat sebelum setiap event dibuat.
func (r *RepositoryImpl) Save(ctx context.Context, agg *rapimnas_model.Aggregate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("ms_rapimnas_setting").Where("id = ?", rapimnas_model.SingletonID).
			Updates(settingValues(agg.Setting)).Error; err != nil {
			return err
		}

		if err := tx.Exec("DELETE FROM ms_rapimnas_gallery_image").Error; err != nil {
			return err
		}
		for i := range agg.GalleryImages {
			agg.GalleryImages[i].ID = 0
			if err := tx.Table("ms_rapimnas_gallery_image").Create(&agg.GalleryImages[i]).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM ms_rapimnas_home_card").Error; err != nil {
			return err
		}
		for i := range agg.HomeCards {
			agg.HomeCards[i].ID = 0
			if err := tx.Table("ms_rapimnas_home_card").Create(&agg.HomeCards[i]).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM ms_rapimnas_rundown_event").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM ms_rapimnas_rundown_day").Error; err != nil {
			return err
		}
		dayIDByCorrelationID := make(map[int64]int64, len(agg.RundownDays))
		for i := range agg.RundownDays {
			correlationID := agg.RundownDays[i].ID
			agg.RundownDays[i].ID = 0
			if err := tx.Table("ms_rapimnas_rundown_day").Create(&agg.RundownDays[i]).Error; err != nil {
				return err
			}
			dayIDByCorrelationID[correlationID] = agg.RundownDays[i].ID
		}
		for i := range agg.RundownEvents {
			if newID, ok := dayIDByCorrelationID[agg.RundownEvents[i].DayID]; ok {
				agg.RundownEvents[i].DayID = newID
			}
			agg.RundownEvents[i].ID = 0
			if err := tx.Table("ms_rapimnas_rundown_event").Create(&agg.RundownEvents[i]).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM ms_rapimnas_resource").Error; err != nil {
			return err
		}
		for i := range agg.Resources {
			agg.Resources[i].ID = 0
			if err := tx.Table("ms_rapimnas_resource").Create(&agg.Resources[i]).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM ms_rapimnas_pickup_location").Error; err != nil {
			return err
		}
		for i := range agg.PickupLocations {
			agg.PickupLocations[i].ID = 0
			if err := tx.Table("ms_rapimnas_pickup_location").Create(&agg.PickupLocations[i]).Error; err != nil {
				return err
			}
		}

		if err := tx.Exec("DELETE FROM ms_rapimnas_contact").Error; err != nil {
			return err
		}
		for i := range agg.Contacts {
			agg.Contacts[i].ID = 0
			if err := tx.Table("ms_rapimnas_contact").Create(&agg.Contacts[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
