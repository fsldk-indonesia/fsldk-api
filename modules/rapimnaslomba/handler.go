package rapimnaslomba

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handler struct {
	db *gorm.DB
}

func newHandler(db *gorm.DB) *handler {
	return &handler{db: db}
}

func decodeJSON(value *string) any {
	if value == nil || *value == "" {
		return nil
	}

	var result any
	if err := json.Unmarshal([]byte(*value), &result); err != nil {
		return nil
	}

	return result
}

func (h *handler) readSetting(c *gin.Context) {
	var setting LombaSetting

	err := h.db.WithContext(c.Request.Context()).
		Where("id = ?", 1).
		First(&setting).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httphelper.Error(
				c,
				apperror.NotFound("Informasi lomba tidak ditemukan"),
			)
			return
		}

		httphelper.Error(c, err)
		return
	}

	httphelper.Success(c, "", gin.H{
		"id":                   setting.ID,
		"namaKegiatan":         setting.NamaKegiatan,
		"deskripsi":            setting.Deskripsi,
		"benefit":              decodeJSON(setting.BenefitJSON),
		"persyaratanUmum":      decodeJSON(setting.PersyaratanUmumJSON),
		"daftarKategori":       decodeJSON(setting.DaftarKategoriJSON),
		"linkGuidebook":        setting.LinkGuidebook,
		"linkPendaftaran":      setting.LinkPendaftaran,
		"linkTwibbon":          setting.LinkTwibbon,
		"linkPengumpulanKarya": setting.LinkPengumpulanKarya,
		"linkTemplateBCC":      setting.LinkTemplateBCC,
		"akunSosmed":           setting.AkunSosmed,
		"contactPersons":       decodeJSON(setting.ContactPersonsJSON),
		"updatedDate":          setting.UpdatedDate,
	})
}

func (h *handler) getPublic(c *gin.Context) {
	h.readSetting(c)
}

func (h *handler) getSetting(c *gin.Context) {
	h.readSetting(c)
}

func (h *handler) updateSetting(c *gin.Context) {
	// Batasi ukuran request maksimal 256 KB.
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		256*1024,
	)

	var payload UpdateSettingPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		httphelper.Error(
			c,
			apperror.BadRequest("Format JSON tidak valid atau terlalu besar"),
		)
		return
	}

	// Validasi seluruh informasi.
	if err := validateSetting(payload); err != nil {
		httphelper.Error(
			c,
			apperror.BadRequest(err.Error()),
		)
		return
	}

	db := h.db.WithContext(c.Request.Context())

	// Pastikan konfigurasi id=1 masih tersedia.
	var current LombaSetting

	if err := db.Where("id = ?", 1).First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httphelper.Error(
				c,
				apperror.NotFound("Pengaturan lomba tidak ditemukan"),
			)
			return
		}

		httphelper.Error(c, err)
		return
	}

	// Gunakan map agar seluruh field bisa diperbarui.
	// Termasuk nilai kosong yang memang diizinkan.
	updates := map[string]interface{}{
		"namaKegiatan":         strings.TrimSpace(payload.NamaKegiatan),
		"deskripsi":            strings.TrimSpace(payload.Deskripsi),
		"benefitJSON":          string(payload.Benefit),
		"persyaratanUmumJSON":  string(payload.PersyaratanUmum),
		"daftarKategoriJSON":   string(payload.DaftarKategori),
		"linkGuidebook":        strings.TrimSpace(payload.LinkGuidebook),
		"linkPendaftaran":      strings.TrimSpace(payload.LinkPendaftaran),
		"linkTwibbon":          strings.TrimSpace(payload.LinkTwibbon),
		"linkPengumpulanKarya": strings.TrimSpace(payload.LinkPengumpulanKarya),
		"linkTemplateBCC":      strings.TrimSpace(payload.LinkTemplateBCC),
		"akunSosmed":           strings.TrimSpace(payload.AkunSosmed),
		"contactPersonsJSON":   string(payload.ContactPersons),
		"updatedDate":          time.Now(),
	}

	if err := db.Model(&LombaSetting{}).
		Where("id = ?", 1).
		Updates(updates).Error; err != nil {
		httphelper.Error(c, err)
		return
	}

	httphelper.Success(
		c,
		"Informasi lomba berhasil diperbarui",
		gin.H{"updated": true},
	)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
