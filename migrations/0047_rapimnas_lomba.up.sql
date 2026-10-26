-- ============================================================
-- FSLDK API — Rapimnas 1 FSLDK Indonesia 2026 (ms_rapimnas_lomba_setting)
-- Modul singleton untuk microsite RAPIMNAS (/rapimnas/pendaftaran/lomba)
-- Menyimpan pengaturan umum lomba serta array JSON detail per kategori (Poster & BCC).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_rapimnas_lomba_setting (
    id                  TINYINT UNSIGNED PRIMARY KEY,
    
    namaKegiatan        VARCHAR(255)  NOT NULL DEFAULT '',
    deskripsi           TEXT          NOT NULL,
    
    benefitJSON         LONGTEXT      NULL,
    persyaratanUmumJSON LONGTEXT      NULL,
    daftarKategoriJSON  LONGTEXT      NULL,
    
    linkGuidebook       VARCHAR(500)  NULL,
    linkPendaftaran     VARCHAR(500)  NULL,
    linkTwibbon         VARCHAR(500)  NULL,
    akunSosmed          VARCHAR(100)  NOT NULL DEFAULT '',
    contactPersonsJSON  LONGTEXT      NULL,
    
    updatedDate         DATETIME      NULL,
    updatedBy           BIGINT        NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- ---------- Seed: baris singleton (Data Lomba dari Guidebook) ----------

INSERT IGNORE INTO ms_rapimnas_lomba_setting (
    id, 
    namaKegiatan, 
    deskripsi, 
    benefitJSON, 
    persyaratanUmumJSON, 
    daftarKategoriJSON,
    linkGuidebook, 
    linkPendaftaran, 
    linkTwibbon, 
    akunSosmed, 
    contactPersonsJSON
) VALUES (
    1,
    'LOMBA RAPIMNAS 1 FSLDK INDONESIA 2026',
    'Lomba RAPIMNAS 1 FSLDK Indonesia 2026 merupakan wadah bagi mahasiswa untuk menuangkan ide, kreativitas, dan gagasan melalui kompetisi Business Case Competition (BCC) dan Poster. Melalui kedua kategori lomba ini, peserta diajak untuk menghadirkan solusi inovatif terhadap isu yang diangkat sekaligus menyampaikan gagasan secara kreatif, komunikatif, dan berdampak.',
    
    -- Benefit
    '["Juara 1: Piagam Penghargaan + Uang Pembinaan", "Juara 2: Piagam Penghargaan + Uang Pembinaan", "Juara 3: Piagam Penghargaan + Uang Pembinaan", "Pengalaman: Mengasah kreativitas dan kemampuan menyampaikan gagasan melalui karya visual", "Pengalaman Presentasi: Kesempatan mempresentasikan karya di hadapan dewan juri bagi 5 finalis terbaik", "Portofolio: Menambah pengalaman dan karya untuk mendukung portofolio pribadi", "Networking: Memperluas relasi dengan mahasiswa dari berbagai perguruan tinggi"]',
    
    -- Persyaratan Umum
    '["Peserta merupakan mahasiswa aktif dari perguruan tinggi di Indonesia.", "Setiap peserta/tim hanya diperbolehkan mengirimkan 1 (satu) karya.", "Setiap peserta wajib memfollow media sosial Instagram resmi panitia.", "Setiap peserta wajib membuat twibbon dan mengunggahnya di Instagram.", "Peserta wajib mengikuti seluruh ketentuan dan mekanisme perlombaan."]',
    
    -- Detail per Kategori (Poster & BCC)
    '[
        {
            "namaKategori": "Poster",
            "temaKategori": "Faith in Action: Peran Generasi Muslim dalam Menghadirkan Solusi bagi Tantangan Masa Kini.",
            "tipePeserta": "Individu",
            "biayaPendaftaran": "Rp 20.000 / Peserta",
            "timeline": [
                {"title": "Open Registration", "date": "10 - 24 Okt 2026"},
                {"title": "Extend Pendaftaran", "date": "15 - 26 Okt 2026"},
                {"title": "Pembuatan Karya", "date": "10 - 27 Okt 2026"},
                {"title": "Penilaian & Seleksi", "date": "28 - 31 Okt 2026"},
                {"title": "Pengumuman 3 Finalis", "date": "1 November 2026"}
            ],
            "alurPendaftaran": [
                "Mengisi formulir pendaftaran yang telah disediakan oleh panitia.",
                "Melakukan pembayaran biaya pendaftaran sesuai ketentuan.",
                "Melengkapi data diri dan mengunggah bukti pembayaran.",
                "Karya dikumpulkan melalui media yang ditentukan sebelum batas waktu pengumpulan."
            ]
        },
        {
            "namaKategori": "BCC",
            "temaKategori": "Muda Berdaya, Ekonomi Berjaya: Mendorong Inovasi Generasi Muslim untuk Pertumbuhan Ekonomi yang Inklusif dan Berkelanjutan",
            "tipePeserta": "Kelompok 3-4 Orang",
            "biayaPendaftaran": "Rp 50.000 / Tim",
            "timeline": [
                {"title": "Open Registration", "date": "10 - 24 Okt 2026"},
                {"title": "Extend Pendaftaran", "date": "15 - 26 Okt 2026"},
                {"title": "Pembuatan Proposal", "date": "10 - 27 Okt 2026"},
                {"title": "Penilaian & Seleksi", "date": "28 - 31 Okt 2026"},
                {"title": "Pengumuman 3 Finalis", "date": "1 November 2026"}
            ],
            "alurPendaftaran": [
                "Mengisi formulir pendaftaran yang telah disediakan oleh panitia.",
                "Melakukan pembayaran biaya pendaftaran sesuai ketentuan.",
                "Peserta menyusun proposal solusi business case sesuai dengan studi kasus.",
                "Proposal dikumpulkan melalui media yang ditentukan sebelum batas waktu pengumpulan."
            ]
        }
    ]',
    
    -- Tautan
    'https://bit.ly/GuidebookLombaFSLDK2026',
    'https://bit.ly/DaftarLombaRapimnas2026',
    'https://bit.ly/TwibbonLombaRapimnas',
    '@rapimnas.fsldk',
    
    -- Contact Persons
    '[{"type": "Ikhwan", "name": "Davi", "phone": "0813-2826-5391"}, {"type": "Akhwat", "name": "Yasmine", "phone": "0813-9371-2761"}]'
);