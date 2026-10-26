package rapimnaslomba

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// Payload lengkap untuk memperbarui informasi lomba.
type UpdateSettingPayload struct {
	NamaKegiatan         string          `json:"namaKegiatan"`
	Deskripsi            string          `json:"deskripsi"`
	Benefit              json.RawMessage `json:"benefit"`
	PersyaratanUmum      json.RawMessage `json:"persyaratanUmum"`
	DaftarKategori       json.RawMessage `json:"daftarKategori"`
	LinkGuidebook        string          `json:"linkGuidebook"`
	LinkPendaftaran      string          `json:"linkPendaftaran"`
	LinkTwibbon          string          `json:"linkTwibbon"`
	LinkPengumpulanKarya string          `json:"linkPengumpulanKarya"`
	LinkTemplateBCC      string          `json:"linkTemplateBCC"`
	AkunSosmed           string          `json:"akunSosmed"`
	ContactPersons       json.RawMessage `json:"contactPersons"`
}

type KategoriPayload struct {
	NamaKategori     string          `json:"namaKategori"`
	TemaKategori     string          `json:"temaKategori"`
	TipePeserta      string          `json:"tipePeserta"`
	BiayaPendaftaran string          `json:"biayaPendaftaran"`
	Timeline         json.RawMessage `json:"timeline"`
	AlurPendaftaran  json.RawMessage `json:"alurPendaftaran"`
}

func validURL(value string, required bool) bool {
	value = strings.TrimSpace(value)

	if value == "" {
		return !required
	}

	if len(value) > 500 {
		return false
	}

	u, err := url.Parse(value)
	if err != nil {
		return false
	}

	return (u.Scheme == "https" || u.Scheme == "http") &&
		u.Hostname() != "" &&
		u.User == nil
}

func validStringArray(raw json.RawMessage) bool {
	var values []string
	if len(raw) == 0 ||
		json.Unmarshal(raw, &values) != nil ||
		values == nil {
		return false
	}

	for _, value := range values {
		if len(strings.TrimSpace(value)) > 2000 {
			return false
		}
	}

	return true
}

func validObjectArray(raw json.RawMessage) bool {
	var values []map[string]json.RawMessage

	if len(raw) == 0 ||
		json.Unmarshal(raw, &values) != nil ||
		values == nil {
		return false
	}

	for _, value := range values {
		if value == nil {
			return false
		}
	}

	return true
}

func validateSetting(p UpdateSettingPayload) error {
	if strings.TrimSpace(p.NamaKegiatan) == "" ||
		len(p.NamaKegiatan) > 255 {
		return errors.New("nama kegiatan wajib diisi (maks. 255 karakter)")
	}

	if strings.TrimSpace(p.Deskripsi) == "" ||
		len(p.Deskripsi) > 20000 {
		return errors.New("deskripsi wajib diisi")
	}

	if len(p.AkunSosmed) > 100 {
		return errors.New("akun media sosial terlalu panjang")
	}

	if !validURL(p.LinkPendaftaran, true) {
		return errors.New("link Google Forms wajib berupa URL http(s) yang valid")
	}

	if !validURL(p.LinkGuidebook, false) ||
		!validURL(p.LinkTwibbon, false) {
		return errors.New("link guidebook atau twibbon tidak valid")
	}

	if !validURL(p.LinkPengumpulanKarya, false) {
		return errors.New("link pengumpulan karya tidak valid")
	}

	if !validURL(p.LinkTemplateBCC, false) {
		return errors.New("link template BCC tidak valid")
	}

	if !validStringArray(p.Benefit) {
		return errors.New("benefit harus berupa array string")
	}

	if !validStringArray(p.PersyaratanUmum) {
		return errors.New("persyaratan umum harus berupa array string")
	}

	if !validObjectArray(p.ContactPersons) {
		return errors.New("contactPersons harus berupa array objek")
	}

	var kategori []KategoriPayload

	if err := json.Unmarshal(p.DaftarKategori, &kategori); err != nil {
		return errors.New("format daftarKategori tidak valid")
	}

	if len(kategori) == 0 || len(kategori) > 20 {
		return errors.New("daftar kategori harus berisi 1–20 kategori")
	}

	seen := map[string]bool{}

	for _, item := range kategori {
		name := strings.ToLower(strings.TrimSpace(item.NamaKategori))

		if name == "" || len(item.NamaKategori) > 150 ||
			strings.TrimSpace(item.TemaKategori) == "" ||
			strings.TrimSpace(item.TipePeserta) == "" ||
			strings.TrimSpace(item.BiayaPendaftaran) == "" {
			return errors.New("data kategori lomba belum lengkap")
		}

		if seen[name] {
			return errors.New("nama kategori lomba tidak boleh duplikat")
		}
		seen[name] = true

		if !validObjectArray(item.Timeline) ||
			!validStringArray(item.AlurPendaftaran) {
			return errors.New("timeline atau alur pendaftaran tidak valid")
		}
	}

	return nil
}
