package rapimnaslomba

import "time"

type LombaSetting struct {
	ID                   uint8      `gorm:"column:id;primaryKey" json:"id"`
	NamaKegiatan         string     `gorm:"column:namaKegiatan" json:"namaKegiatan"`
	Deskripsi            string     `gorm:"column:deskripsi" json:"deskripsi"`
	BenefitJSON          *string    `gorm:"column:benefitJSON" json:"-"`
	PersyaratanUmumJSON  *string    `gorm:"column:persyaratanUmumJSON" json:"-"`
	DaftarKategoriJSON   *string    `gorm:"column:daftarKategoriJSON" json:"-"`
	LinkGuidebook        *string    `gorm:"column:linkGuidebook" json:"linkGuidebook"`
	LinkPendaftaran      *string    `gorm:"column:linkPendaftaran" json:"linkPendaftaran"`
	LinkTwibbon          *string    `gorm:"column:linkTwibbon" json:"linkTwibbon"`
	LinkPengumpulanKarya *string    `gorm:"column:linkPengumpulanKarya" json:"linkPengumpulanKarya"`
	LinkTemplateBCC      *string    `gorm:"column:linkTemplateBCC" json:"linkTemplateBCC"`
	AkunSosmed           string     `gorm:"column:akunSosmed" json:"akunSosmed"`
	ContactPersonsJSON   *string    `gorm:"column:contactPersonsJSON" json:"-"`
	UpdatedDate          *time.Time `gorm:"column:updatedDate" json:"updatedDate"`
	UpdatedBy            *int64     `gorm:"column:updatedBy" json:"updatedBy"`
}

func (LombaSetting) TableName() string {
	return "ms_rapimnas_lomba_setting"
}
