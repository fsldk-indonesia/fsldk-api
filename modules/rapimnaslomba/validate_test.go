package rapimnaslomba

import (
	"encoding/json"
	"testing"
)

func validTestPayload() UpdateSettingPayload {
	return UpdateSettingPayload{
		NamaKegiatan: "Lomba RAPIMNAS 2026",
		Deskripsi:    "Kompetisi Poster dan BCC",
		Benefit: json.RawMessage(
			`["Sertifikat", "Uang Pembinaan"]`,
		),
		PersyaratanUmum: json.RawMessage(
			`["Mahasiswa aktif"]`,
		),
		DaftarKategori: json.RawMessage(`[
			{
				"namaKategori": "Poster",
				"temaKategori": "Faith in Action",
				"tipePeserta": "Individu",
				"biayaPendaftaran": "Rp 20.000",
				"timeline": [],
				"alurPendaftaran": ["Mengisi formulir"]
			},
			{
				"namaKategori": "BCC",
				"temaKategori": "Muda Berdaya",
				"tipePeserta": "Kelompok 3-4 Orang",
				"biayaPendaftaran": "Rp 50.000 / Tim",
				"timeline": [],
				"alurPendaftaran": ["Mengisi formulir"]
			}
		]`),
		LinkPendaftaran: "https://docs.google.com/forms/",
		ContactPersons:  json.RawMessage(`[]`),
	}
}

func TestValidateSetting(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*UpdateSettingPayload)
		wantErr bool
	}{
		{
			name:    "data valid",
			modify:  func(p *UpdateSettingPayload) {},
			wantErr: false,
		},
		{
			name: "nama kegiatan kosong",
			modify: func(p *UpdateSettingPayload) {
				p.NamaKegiatan = ""
			},
			wantErr: true,
		},
		{
			name: "URL pendaftaran tidak valid",
			modify: func(p *UpdateSettingPayload) {
				p.LinkPendaftaran = "bukan-url"
			},
			wantErr: true,
		},
		{
			name: "JSON kategori tidak valid",
			modify: func(p *UpdateSettingPayload) {
				p.DaftarKategori = json.RawMessage(`{}`)
			},
			wantErr: true,
		},
		{
			name: "nama kategori duplikat",
			modify: func(p *UpdateSettingPayload) {
				p.DaftarKategori = json.RawMessage(`[
					{
						"namaKategori": "Poster",
						"temaKategori": "Tema",
						"tipePeserta": "Individu",
						"biayaPendaftaran": "Rp 20.000",
						"timeline": [],
						"alurPendaftaran": []
					},
					{
						"namaKategori": "poster",
						"temaKategori": "Tema",
						"tipePeserta": "Individu",
						"biayaPendaftaran": "Rp 20.000",
						"timeline": [],
						"alurPendaftaran": []
					}
				]`)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := validTestPayload()
			tc.modify(&payload)

			err := validateSetting(payload)

			if (err != nil) != tc.wantErr {
				t.Fatalf(
					"validateSetting() err=%v, wantErr=%v",
					err,
					tc.wantErr,
				)
			}
		})
	}
}
